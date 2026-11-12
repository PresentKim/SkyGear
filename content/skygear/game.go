// Package skygear is the factory building game.
package skygear

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/catalog"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/migrate"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// Game is the skygear game type.
type Game struct {
	log *slog.Logger
	db  *db.DB
	// release is the running server release, recorded in new saves.
	release string

	// isOperator reports whether a player may use the operator tools.
	isOperator func(*player.Player) bool

	// sessions maps each active instance world to its session, so the
	// platform's calls that carry only a world find the session. It is read by
	// every block tick and written only when an instance starts or stops, so
	// reads take no lock.
	sessions sync.Map // *world.World -> *session
}

// New creates the game and registers its commands. There is one game per
// process. isOperator, if not nil, allows players to use the operator tools.
// release is the running server release.
func New(log *slog.Logger, d *db.DB, isOperator func(*player.Player) bool, release string) *Game {
	g := &Game{log: log, db: d, isOperator: isOperator, release: release}
	for _, c := range core.Commands() {
		cmd.Register(c)
	}
	catalog.RegisterCreativeItems()
	registerAdmin(g)
	registerDebug(g)
	go codex.Index(i18n.Default)
	return g
}

// Init starts a new save: it records the release the save's data is in, so
// no migration patch applies to it, and builds the starting island.
func (g *Game) Init(tx *world.Tx, inst *instance.Instance) {
	if err := g.db.SetSaveState(inst.Save.ID, migrate.VersionKey, []byte(migrate.Stamp(g.release))); err != nil {
		g.log.Error("record the save version", "save", inst.Save.ID, "err", err)
	}
	buildIsland(tx)
}

// buildIsland builds a 3×3×3 sky island in the middle of the first chunk,
// bedrock at the bottom, dirt, grass on top, with an oak tree in the middle.
func buildIsland(tx *world.Tx) {
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
	c.FlushInterval = inst.StateFlushInterval()
	s := &session{
		Lifecycle: core.Lifecycle{Session: c}, g: g,
	}
	s.state = savedState.Of(c)
	s.tickReach = inst.LoadRadius() * 16
	core.Load.Emit(core.Tick{S: c})
	island.SetKeep(c, func(tx *world.Tx, c island.ChunkPos) { inst.KeepLoaded(tx, c[0], c[1]) })
	inst.World.Do(func(tx *world.Tx) {
		core.Start.Emit(core.Tick{S: c, Tx: tx})
	})
	inst.World.Handle(&core.WorldHandler{S: c})
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

// session is one active save as the platform sees it (instance.Session): the
// core session (the Session of core.Lifecycle), which runs the lifecycle and
// the player handler, plus the platform glue the game answers itself: Spawn,
// Relocate, MenuBack, OpenScreen and Close. It holds no feature step; those are
// the subscriptions of the feature packages. All fields are owned by the
// instance world.
type session struct {
	core.Lifecycle
	g *Game
	// state is the saved state (savedState).
	state *State
	// tickReach is how far from a viewer blocks tick, in blocks (the load
	// radius in chunks).
	tickReach int
}

// Spawn is where a player without a position starts (island.Spawn).
func (s *session) Spawn() mgl64.Vec3 { return island.Spawn(s.Session) }

// Relocate sends a player stored in the second dimension (the server stopped
// during its run) to the island instead (instance.Relocator).
func (s *session) Relocate(pos mgl64.Vec3) bool { return core.InArena(pos) }

// Close emits Close and stores the state (core.Lifecycle), then forgets the
// session.
func (s *session) Close(tx *world.Tx) {
	s.Lifecycle.Close(tx)
	s.g.sessions.Delete(s.World)
}

// EntityTypes returns the default entity types plus those of the listed
// features (Features). The registry is keyed by name, so the order is free.
func EntityTypes() []world.EntityType {
	return append(entity.DefaultRegistry.Types(), core.EntityTypes()...)
}

// MenuBack shows the game screen a platform screen opened in the save goes
// back to (instance.Session.MenuBack): the island management screen for the
// members and the island screens, the game menu for the rest.
func (s *session) MenuBack(p *player.Player, from string) {
	if from == instance.ScreenMembers || from == instance.ScreenIsland {
		screens.ShowIslandManage(s.Session, p)
		return
	}
	screens.ShowMenu(s.Session, p)
}

// OpenScreen shows a game screen a platform screen's tab opens
// (instance.Session.OpenScreen): the island expansion, from the last tab of
// the island management.
func (s *session) OpenScreen(p *player.Player, screen string) {
	if screen == instance.ScreenExpand {
		expand.Show(s.Session, p)
		return
	}
	screens.ShowMenu(s.Session, p)
}
