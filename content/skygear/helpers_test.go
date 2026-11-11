package skygear

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/platform/db"
)

// The helpers below are the root package's names for the test support
// packages (core/coretest, machine/machinetest), over the root's session.

// testSession returns a session over a synchronous world, without an
// instance manager: coretest.Session with the game, a database and the save
// the database holds for it.
func testSession(t testing.TB) *session {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.TouchAccount("host", "", "host"); err != nil {
		t.Fatal(err)
	}
	save, err := d.CreateSave("host", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	l := coretest.Session(t)
	l.SaveID, l.SaveName = save.ID, save.Name
	g := &Game{log: slog.Default(), db: d}
	s := &session{Lifecycle: l, g: g}
	s.state = savedState.Of(l.Session)
	g.sessions.Store(l.World, s)
	return s
}

func init() { world.DefaultBlockRegistry.Finalize() }

func loadAround(s *session) { coretest.LoadAround(s.Lifecycle) }

func spawnPlayer(s *session) *world.EntityHandle { return coretest.SpawnPlayer(s.Lifecycle) }

// withPlayer runs f with the player in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func withPlayer(t testing.TB, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	coretest.WithPlayer(t, s.Lifecycle, h, f)
}

func tick(s *session, n int) { coretest.Tick(s.Lifecycle, n) }

// shapeAs is the shape s keeps under key (nil if none).
func shapeAs(s *session, key any) debug.Shape { return coretest.ShapeAs(s.Lifecycle, key) }

var machinePos = machinetest.Pos

// placeTestMachine places a machine and returns its state.
func placeTestMachine(s *session, pos cube.Pos, kind string) *machine.State {
	return machinetest.Place(s.Lifecycle, pos, kind)
}
