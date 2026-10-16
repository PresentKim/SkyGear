// Package dragonfactory is the factory building game (docs/design/02-dragonfactory.md).
package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemnet"
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
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/waila"
)

// IslandY is the Y level of the island surface.
const IslandY = 64

// Game is the dragonfactory game type.
type Game struct {
	log *slog.Logger
	db  *db.DB

	// isOperator reports whether a player may use the operator tools.
	isOperator func(*player.Player) bool

	mu       sync.Mutex
	sessions map[*world.World]*session
}

// activeGame is the game used by blocks, whose methods have no other way to
// reach their session.
var activeGame = &Game{sessions: map[*world.World]*session{}}

// New creates the game and registers its commands. There is one game per
// process. isOperator, if not nil, allows players to use the operator tools.
func New(log *slog.Logger, d *db.DB, isOperator func(*player.Player) bool) *Game {
	g := &Game{log: log, db: d, isOperator: isOperator, sessions: map[*world.World]*session{}}
	activeGame = g
	registerCommands(g)
	registerCreativeItems()
	registerItemIO()
	registerAdmin(g)
	return g
}

func (*Game) ID() string   { return "dragonfactory" }
func (*Game) Name() string { return i18n.T(i18n.Default, "name.Game") }

// islandCentre is the block (x and z) at the middle of the starting chunk,
// where the starting island is built.
const islandCentre = 8

// Init builds the starting island (docs/design/02-dragonfactory.md §2): a
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
		g.log.Error("load dragonfactory state; starting from a new state", "save", inst.Save.ID, "err", err)
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
	})
	inst.World.Handle(&worldHandler{s: s})
	g.mu.Lock()
	g.sessions[inst.World] = s
	g.mu.Unlock()
	return s
}

// sessionOf returns the session running in w.
func (g *Game) sessionOf(w *world.World) (*session, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.sessions[w]
	return s, ok
}

// session is one active save. All fields are owned by the instance world.
type session struct {
	// lastLook is when each player's WAILA panel was last updated.
	lastLook map[*world.EntityHandle]time.Time
	// urgentBarShown is whether the urgent mission boss bar is up.
	urgentBarShown bool
	// fairy is the gift fairy visiting now, or nil.
	fairy *world.EntityHandle
	g     *Game
	world *world.World
	clock *instance.Clock
	// keep keeps an owned chunk loaded (instance.KeepLoaded); nil in tests.
	keep     func(tx *world.Tx, c ChunkPos)
	saveID   int64
	saveName string
	hostID   string
	state    *State
	rand     *rand.Rand
	// displays are the drawer display entities by drawer position.
	displays map[cube.Pos]*drawerDisplay
	// lastUse and lastPunch are each player's last drawer use and punch.
	lastUse   map[*world.EntityHandle]drawerUse
	lastPunch map[*world.EntityHandle]drawerPunch
	// lastOpen debounces block interactions that open a screen: the client
	// repeats the use packet while the button is held.
	lastOpen map[*world.EntityHandle]time.Time
	// lastStick is when each player last used a stick on the ground.
	lastStick map[*world.EntityHandle]time.Time
	// walked is the part of a block each player walked since the last whole
	// block was counted (counter.go).
	walked map[*world.EntityHandle]float64
	// boardMinute is the play time minute shown on the HUD.
	boardMinute int
	// shapes are debug shapes shown to everyone in the instance (shapes.go).
	shapes map[any]debug.Shape
	// cropQueue holds the next growth update of every crop (crops.go).
	cropQueue cropQueue
	// expanding are the players in island expansion mode (expand.go).
	expanding map[*world.EntityHandle]*expansion
	// recipesFree opens every recipe until the instance closes
	// (/dfadmin research free, recipes.go).
	recipesFree bool
	// expandView is how expansion mode shows the chunks (expandViews; ""
	// is the floor), set with /dfadmin expandview.
	expandView string
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

// Spawn is next to the tree of the starting island. A save made before the
// island moved to the middle of its chunk has no Home and keeps its old
// island around the origin.
func (s *session) Spawn() mgl64.Vec3 {
	if s.state != nil && s.state.Home != ([2]float64{}) {
		return mgl64.Vec3{s.state.Home[0], IslandY, s.state.Home[1]}
	}
	return mgl64.Vec3{1.5, IslandY, 1.5}
}

func (s *session) Join(p *player.Player, first bool) {
	_, day, _ := s.clock.Now()
	p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.game.Join.1"), s.saveName, day)
	if first {
		// Bare hands: only the menu item (§2).
		_ = p.Inventory().SetItem(8, menuItem())
	}
	migrateConduitItems(p)
	s.sendShapes(p)
	s.purgeLost()
	s.sendRecipesTo(p)
	s.updateBoard(p)
	s.applyRPG(p)
}

// dropAt drops an item stack at pos with a small random push, so drops
// never just fall dead still (docs/design/02-dragonfactory.md §18.1).
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
	s.stopExpand(p, p.Tx())
	delete(s.lastLook, p.H())
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
	s.broadcast(tx, fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.game.NewDay.1"), day))
	s.updateBoards(tx)
}

func (s *session) Autosave(tx *world.Tx) {
	s.refreshPlayTime(tx)
	s.storeState()
}

func (s *session) Close(tx *world.Tx) {
	s.removeDisplays(tx)
	s.removeVisuals(tx)
	s.Autosave(tx)
	s.g.mu.Lock()
	delete(s.g.sessions, s.world)
	s.g.mu.Unlock()
}

func (s *session) storeState() {
	if err := s.state.save(s.g.db, s.saveID); err != nil {
		s.g.log.Error("store dragonfactory state", "save", s.saveID, "err", err)
	}
}

// broadcast messages every player in the instance.
func (s *session) broadcast(tx *world.Tx, msg string) {
	for e := range tx.Players() {
		e.(*player.Player).Message(msg)
	}
}
