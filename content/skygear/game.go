// Package skygear is the factory building game.
package skygear

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/catalog"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/wave"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

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
	catalog.RegisterCreativeItems()
	registerItemIO()
	registerAdmin(g)
	registerDebug(g)
	go codex.Index(i18n.Default)
	return g
}

// Init builds the starting island: a
// 3×3×3 sky island in the middle of the first chunk, bedrock at the bottom,
// dirt, grass on top, with an oak tree in the middle.
func (*Game) Init(tx *world.Tx, _ *instance.Instance) {
	for x := island.Centre - 1; x <= island.Centre+1; x++ {
		for z := island.Centre - 1; z <= island.Centre+1; z++ {
			tx.SetBlock(cube.Pos{x, island.IslandY - 3, z}, block.Bedrock{}, nil)
			tx.SetBlock(cube.Pos{x, island.IslandY - 2, z}, block.Dirt{}, nil)
			tx.SetBlock(cube.Pos{x, island.IslandY - 1, z}, block.Grass{}, nil)
		}
	}
	farm.GrowTreeAt(tx, cube.Pos{island.Centre, island.IslandY, island.Centre})
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
		Session: c, g: g,
	}
	s.state = savedState.Of(c)
	s.tickReach = inst.LoadRadius() * 16
	farm.LoadCrops(s.Session)
	island.SetKeep(c, func(tx *world.Tx, c island.ChunkPos) { inst.KeepLoaded(tx, c[0], c[1]) })
	inst.World.Do(func(tx *world.Tx) {
		for _, c := range island.StateOf(s.Session).Chunks {
			island.KeepLoaded(s.Session, tx, c)
		}
		climate.Apply(s.Session, tx)
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
	// diedFood is the food level each player died with, given back when it
	// respawns (handler.go).
	diedFood map[*world.EntityHandle]int
	g        *Game
	// state is the saved state (savedState).
	state *State
	// tickReach is how far from a viewer blocks tick, in blocks (the load
	// radius in chunks).
	tickReach int
	// fastCount counts the Fast calls (an item step is every second one,
	// conduit.go).
	fastCount int
}

// Spawn is where a player without a position starts (island.Spawn).
func (s *session) Spawn() mgl64.Vec3 { return island.Spawn(s.Session) }

func (s *session) Join(p *player.Player, first bool) {
	if first {
		// Bare hands: only the menu item.
		_ = p.Inventory().SetItem(8, screens.MenuItem(i18n.Of(p)))
	}
	screens.RenameMenuItems(p)
	fairy.BackfillDust(s.Session, p, p.Tx())
	climate.ShowWeather(s.Session, p)
	s.SendShapes(p)
	island.PurgeLost(s.Session)
	progress.SendRecipesTo(s.Session, p)
	screens.UpdateBoard(s.Session, p)
}

func (s *session) Leave(p *player.Player) {
	s.Session.Leave(p)
	expand.Stop(s.Session, p, p.Tx())
	delete(s.diedFood, p.H())
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
	island.PurgeLost(s.Session)
	s.Broadcast(tx, "game.newDay", day)
}

func (s *session) Autosave(tx *world.Tx) {
	s.Flush()
}

func (s *session) Close(tx *world.Tx) {
	storage.RemoveDisplays(s.Session, tx)
	conduit.RemoveVisuals(s.Session, tx)
	s.Session.Close()
	core.Unregister(s.Session)
	s.g.sessions.Delete(s.World)
}

// activeGameSession is the session of the world of tx, nil if it has none.
func activeGameSession(tx *world.Tx) *session {
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		return s
	}
	return nil
}

// EntityTypes returns the default entity types plus the game's own.
func EntityTypes() []world.EntityType {
	return append(append(entity.DefaultRegistry.Types(), storage.HolderType, fairy.EntityType, conduit.VisualType, conduit.PlateType, wave.GatePortalType), wave.MonsterTypes()...)
}

// MenuBack shows the game screen a platform screen opened in the save goes
// back to (instance.Session.MenuBack): the island management screen for the
// members, the game menu for the rest.
func (s *session) MenuBack(p *player.Player, from string) {
	if from == instance.ScreenMembers {
		screens.ShowIslandManage(s.Session, p)
		return
	}
	screens.ShowMenu(s.Session, p)
}
