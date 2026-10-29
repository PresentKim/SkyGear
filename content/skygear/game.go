// Package skygear is the factory building game.
package skygear

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// IslandY is the Y level of the island surface.
const IslandY = 64

// Game is the skygear game type.
type Game struct {
	log *slog.Logger
	db  *db.DB

	// isOperator reports whether a player may use the operator tools.
	isOperator func(*player.Player) bool

	// sessions maps each active instance world to its session. It is read
	// by every block tick and written only when an instance starts or stops,
	// so reads take no lock.
	sessions sync.Map // *world.World -> *session
}

func init() {
	// The quest panel follows the progress.
	progress.OnChange = func(c *core.Session, tx *world.Tx) {
		if s, ok := activeGame.sessionOf(c.World); ok {
			s.updateBoards(tx)
		}
	}
}

// activeGame is the game used by blocks, whose methods have no other way to
// reach their session.
var activeGame = &Game{}

// New creates the game and registers its commands. There is one game per
// process. isOperator, if not nil, allows players to use the operator tools.
func New(log *slog.Logger, d *db.DB, isOperator func(*player.Player) bool) *Game {
	g := &Game{log: log, db: d, isOperator: isOperator}
	activeGame = g
	registerCommands(g)
	registerCreativeItems()
	registerItemIO()
	registerAdmin(g)
	registerDebug(g)
	go codexIndex(i18n.Default)
	return g
}

// islandCentre is the block (x and z) at the middle of the starting chunk,
// where the starting island is built.
const islandCentre = 8

// Init builds the starting island: a
// 3×3×3 sky island in the middle of the first chunk, bedrock at the bottom,
// dirt, grass on top, with an oak tree in the middle.
func (*Game) Init(tx *world.Tx, _ *instance.Instance) {
	for x := islandCentre - 1; x <= islandCentre+1; x++ {
		for z := islandCentre - 1; z <= islandCentre+1; z++ {
			tx.SetBlock(cube.Pos{x, IslandY - 3, z}, block.Bedrock{}, nil)
			tx.SetBlock(cube.Pos{x, IslandY - 2, z}, block.Dirt{}, nil)
			tx.SetBlock(cube.Pos{x, IslandY - 1, z}, block.Grass{}, nil)
		}
	}
	growTreeAt(tx, cube.Pos{islandCentre, IslandY, islandCentre})
}

// Start begins a session for an activated save.
func (g *Game) Start(inst *instance.Instance) instance.Session {
	doc, err := g.db.SaveState(inst.Save.ID, stateKey)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		g.log.Error("load skygear state; starting from a new state", "save", inst.Save.ID, "err", err)
	}
	saveID := inst.Save.ID
	c := core.New(core.Config{
		World: inst.World, Clock: inst.Clock, SaveID: saveID, SaveName: inst.Save.Name, HostID: inst.Save.Owner,
		Log: g.log, IsOperator: g.isOperator, State: doc,
		Store: func(b []byte) error { return g.db.SetSaveState(saveID, stateKey, b) },
	})
	s := &session{
		Session: c, g: g, displays: map[cube.Pos]*drawerDisplay{}, stats: newProdStats(), machines: map[cube.Pos]*machineState{},
		lastUse: map[*world.EntityHandle]drawerUse{}, lastPunch: map[*world.EntityHandle]drawerPunch{},
	}
	s.state = savedState.Of(c)
	st := s.state
	s.tickReach = inst.LoadRadius() * 16
	s.loadCrops()
	s.keep = func(tx *world.Tx, c ChunkPos) { inst.KeepLoaded(tx, c[0], c[1]) }
	inst.World.Do(func(tx *world.Tx) {
		for _, c := range st.Chunks {
			s.keepLoaded(tx, c)
		}
		s.applyEnvironment(tx)
	})
	inst.World.Handle(&worldHandler{s: s})
	g.sessions.Store(inst.World, s)
	core.Register(c)
	return s
}

// sessionOf returns the session running in w.
func (g *Game) sessionOf(w *world.World) (*session, bool) {
	v, ok := g.sessions.Load(w)
	if !ok {
		return nil, false
	}
	return v.(*session), true
}

// session is one active save: the core session (core.Session) and the
// features' data that is not in core parts yet. All fields are owned by the
// instance world.
type session struct {
	*core.Session
	// lastLook is when each player's WAILA panel was last updated.
	lastLook map[*world.EntityHandle]time.Time
	// veinOutlines are the vein outlines players are shown, and veinFood
	// the fraction of a food point each owes for vein mining (vein.go).
	veinOutlines map[*world.EntityHandle]veinOutline
	veinFood     map[*world.EntityHandle]float64
	// diedFood is the food level each player died with, given back when it
	// respawns (handler.go).
	diedFood map[*world.EntityHandle]int
	// codexNav is where each player is in the codex and came from, and
	// codexRecent the items they looked at last (codextabs.go).
	codexNav    map[*world.EntityHandle]*codexNav
	codexRecent map[*world.EntityHandle][]item.Stack
	// helpTab is the help tab each player looked at last (help.go).
	helpTab map[*world.EntityHandle]helpTab
	// urgentBarShown is whether the urgent mission boss bar is up.
	urgentBarShown bool
	// fairy is the gift fairy visiting now, or nil.
	fairy *world.EntityHandle
	g     *Game
	// growInit and growAt are whether the growth clock started and the
	// instance time it last ran to; rain and snow are whether it rains or
	// snows now (fairyspell.go).
	growInit bool
	growAt   time.Duration
	rain     bool
	snow     bool
	// keep keeps an owned chunk loaded (instance.KeepLoaded); nil in tests.
	keep func(tx *world.Tx, c ChunkPos)
	// state is the saved state (savedState).
	state *State
	// displays are the drawer display entities by drawer position.
	displays map[cube.Pos]*drawerDisplay
	// holderPose is the hand-equipped item pose tried on display entities.
	holderPose holderPose
	// lastUse is each player's last drawer or storage controller use, and
	// lastPunch the last drawer punch.
	lastUse   map[*world.EntityHandle]drawerUse
	lastPunch map[*world.EntityHandle]drawerPunch
	// storageOrders are how each player sorts storage controller lists.
	storageOrders map[*world.EntityHandle]storageOrder
	// lastStick is when each player last used a stick on the ground.
	lastStick map[*world.EntityHandle]time.Time
	// chestStreaks are each player's sneaking stick uses on a chest in a row
	// (stickchest.go).
	chestStreaks map[*world.EntityHandle]chestStreak
	// boards are the HUDs players were last sent (hud.Board.Key).
	boards map[*world.EntityHandle]string
	// shapes are debug shapes shown to everyone in the instance (shapes.go).
	shapes map[any]debug.Shape
	// cropQueue holds the next growth update of every crop (crops.go).
	cropQueue cropQueue
	// expanding are the players in island expansion mode (expand.go).
	expanding map[*world.EntityHandle]*expansion
	// tickReach is how far from a viewer blocks tick, in blocks (the load
	// radius in chunks).
	tickReach int
	// grid shares out the energy of the island's conduits (conduit.go).
	grid *energy.Grid
	// items moves items over the item conduits; fastCount counts the Fast
	// calls (an item step is every second one, conduit.go).
	items     *itemnet.Grid
	fastCount int
	// wave is the wave of the dimension gate being fought (wave.go).
	wave *waveState
	// fluids moves fluids over the fluid conduits (fluidmachine.go).
	fluids *fluidnet.Grid
	// visuals are the display entities of the conduit bundles by position
	// (conduitvisual.go).
	visuals map[cube.Pos]*world.EntityHandle
	// plates are the connector plate entities of the bundles (conduitplate.go).
	plates map[plateKey]*world.EntityHandle
	// plateTurn is the side of the last connector plate form each player was
	// shown by clicking a conduit where no plate sits (nextPlateSide).
	plateTurn map[uuid.UUID]int
	// stats is the production record, machines the machines that tick by
	// position; now the seconds the island ran, lastTick the last machine
	// tick seen and uptime the seconds without a stall (stats.go,
	// monitor.go).
	stats    *prodStats
	machines map[cube.Pos]*machineState
	now      int
	lastTick int64
	uptime   int
}

// Spawn is next to the tree of the starting island: Home is the island's
// centre, where the trunk grows, so the spawn is one block east of it. An old
// save without Home has its island around the origin.
func (s *session) Spawn() mgl64.Vec3 {
	home := [2]float64{1.5, 1.5}
	if s.state != nil && s.state.Home != ([2]float64{}) {
		home = s.state.Home
	}
	return mgl64.Vec3{home[0] + 1, IslandY, home[1]}
}

func (s *session) Join(p *player.Player, first bool) {
	if first {
		// Bare hands: only the menu item.
		_ = p.Inventory().SetItem(8, menuItem(i18n.Of(p)))
	}
	renameMenuItems(p)
	s.backfillFairyDust(p, p.Tx())
	s.showWeather(p)
	s.sendShapes(p)
	s.purgeLost()
	progress.SendRecipesTo(s.Session, p)
	delete(s.boards, p.H())
	s.updateBoard(p)
}

func (s *session) Leave(p *player.Player) {
	s.Session.Leave(p)
	delete(s.lastUse, p.H())
	delete(s.lastPunch, p.H())
	delete(s.storageOrders, p.H())
	s.stopExpand(p, p.Tx())
	delete(s.lastLook, p.H())
	delete(s.boards, p.H())
	delete(s.veinOutlines, p.H())
	delete(s.veinFood, p.H())
	delete(s.codexNav, p.H())
	delete(s.codexRecent, p.H())
	delete(s.helpTab, p.H())
	delete(s.diedFood, p.H())
	delete(s.lastStick, p.H())
	delete(s.chestStreaks, p.H())
	waila.Clear(p)
	// Shapes are client-side: clear them before the player leaves the
	// instance world.
	p.RemoveAllDebugShapes()
	p.RemoveScoreboard()
	// Locked recipes belong to this island (recipes.go).
	p.SetRecipeFilter(nil)
}

func (s *session) Handler(*player.Player) player.Handler { return &handler{s: s} }

func (s *session) NewDay(tx *world.Tx, day int) {
	s.purgeLost()
	s.Broadcast(tx, "game.newDay", day)
}

func (s *session) Autosave(tx *world.Tx) {
	s.Flush()
}

func (s *session) Close(tx *world.Tx) {
	s.removeDisplays(tx)
	s.removeVisuals(tx)
	s.Session.Close()
	core.Unregister(s.Session)
	s.g.sessions.Delete(s.World)
}
