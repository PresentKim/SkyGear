// Package skygear is the factory building game.
package skygear

import (
	"encoding/json"
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemnet"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
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
	st, err := loadState(g.db, inst.Save.ID)
	if err != nil {
		g.log.Error("load skygear state; starting from a new state", "save", inst.Save.ID, "err", err)
		st = newState()
	}
	s := &session{
		g: g, world: inst.World, clock: inst.Clock,
		saveID: inst.Save.ID, saveName: inst.Save.Name, hostID: inst.Save.Owner,
		state: st, displays: map[cube.Pos]*drawerDisplay{}, stats: newProdStats(), machines: map[cube.Pos]*machineState{},
		lastUse: map[*world.EntityHandle]drawerUse{}, lastPunch: map[*world.EntityHandle]drawerPunch{}, rand: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(inst.Save.ID))),
	}
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

// session is one active save. All fields are owned by the instance world.
type session struct {
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
	codexNav map[*world.EntityHandle]*codexNav
	// copies are test copies of researches for the research screen and
	// copiesDone the ones marked done (research.go isCopy).
	copies      []*Research
	copiesDone  map[string]bool
	codexRecent map[*world.EntityHandle][]item.Stack
	// helpTab is the help tab each player looked at last (help.go).
	helpTab map[*world.EntityHandle]helpTab
	// urgentBarShown is whether the urgent mission boss bar is up.
	urgentBarShown bool
	// fairy is the gift fairy visiting now, or nil.
	fairy *world.EntityHandle
	g     *Game
	world *world.World
	clock *instance.Clock
	// growInit and growAt are whether the growth clock started and the
	// instance time it last ran to; rain and snow are whether it rains or
	// snows now (fairyspell.go).
	growInit bool
	growAt   time.Duration
	rain     bool
	snow     bool
	// keep keeps an owned chunk loaded (instance.KeepLoaded); nil in tests.
	keep     func(tx *world.Tx, c ChunkPos)
	saveID   int64
	saveName string
	hostID   string
	state    *State
	// stateDirty is whether state changed since it was last handed to
	// writer, which writes it to the database (storeState).
	stateDirty bool
	writer     *stateWriter
	rand       *rand.Rand
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
	// lastOpen debounces block interactions that open a screen: the client
	// repeats the use packet while the button is held.
	lastOpen map[*world.EntityHandle]time.Time
	// lastStick is when each player last used a stick on the ground.
	lastStick map[*world.EntityHandle]time.Time
	// chestStreaks are each player's sneaking stick uses on a chest in a row
	// (stickchest.go).
	chestStreaks map[*world.EntityHandle]chestStreak
	// walked is the part of a block each player walked since the last whole
	// block was counted (counter.go).
	walked map[*world.EntityHandle]float64
	// boards are the HUDs players were last sent (hud.Board.Key).
	boards map[*world.EntityHandle]string
	// shapes are debug shapes shown to everyone in the instance (shapes.go).
	shapes map[any]debug.Shape
	// cropQueue holds the next growth update of every crop (crops.go).
	cropQueue cropQueue
	// expanding are the players in island expansion mode (expand.go).
	expanding map[*world.EntityHandle]*expansion
	// recipesFree opens every recipe until the instance closes
	// (/sgadmin research free, recipes.go).
	recipesFree bool
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

// debounced reports whether p opened a screen less than 500ms ago, and
// otherwise records this opening.
func (s *session) debounced(p *player.Player) bool {
	if s.lastOpen == nil {
		s.lastOpen = map[*world.EntityHandle]time.Time{}
	}
	if t, ok := s.lastOpen[p.H()]; ok && time.Since(t) < 500*time.Millisecond {
		return true
	}
	s.lastOpen[p.H()] = time.Now()
	return false
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
	s.sendRecipesTo(p)
	delete(s.boards, p.H())
	s.updateBoard(p)
}

// dropAt drops an item stack at pos with a small random push, so drops
// never just fall dead still.
// Every item the game drops goes through here unless it is thrown on
// purpose (the fairy's gift).
func (s *session) dropAt(tx *world.Tx, pos mgl64.Vec3, it item.Stack) {
	a := s.rand.Float64() * 2 * math.Pi
	speed := 0.06 + s.rand.Float64()*0.08
	vel := mgl64.Vec3{speed * math.Cos(a), 0.22 + s.rand.Float64()*0.1, speed * math.Sin(a)}
	tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: pos, Velocity: vel}, it))
}

func (s *session) Leave(p *player.Player) {
	delete(s.lastOpen, p.H())
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
	delete(s.walked, p.H())
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
	s.broadcast(tx, "game.newDay", day)
}

func (s *session) Autosave(tx *world.Tx) {
	s.flushState()
}

func (s *session) Close(tx *world.Tx) {
	s.removeDisplays(tx)
	s.removeVisuals(tx)
	s.closeState()
	s.g.sessions.Delete(s.world)
}

// storeState marks the state to be written on the next second.
func (s *session) storeState() {
	s.stateDirty = true
}

// flushState hands the state to the state writer.
func (s *session) flushState() {
	b, err := json.Marshal(s.state)
	if err != nil {
		s.g.log.Error("store skygear state", "save", s.saveID, "err", err)
		return
	}
	s.stateDirty = false
	if s.writer == nil {
		s.writer = newStateWriter(s.writeState)
	}
	s.writer.put(b)
}

// closeState stops the state writer and writes the state before returning.
func (s *session) closeState() {
	if s.writer != nil {
		s.writer.stop()
		s.writer = nil
	}
	b, err := json.Marshal(s.state)
	if err != nil {
		s.g.log.Error("store skygear state", "save", s.saveID, "err", err)
		return
	}
	s.stateDirty = false
	s.writeState(b)
}

func (s *session) writeState(b []byte) {
	if err := s.g.db.SetSaveState(s.saveID, stateKey, b); err != nil {
		s.g.log.Error("store skygear state", "save", s.saveID, "err", err)
	}
}

// broadcast messages every player in the instance.
func (s *session) broadcast(tx *world.Tx, key string, args ...any) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.Message(i18n.T(i18n.Of(p), key, args...))
	}
}
