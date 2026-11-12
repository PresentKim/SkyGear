package skygear

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/content/skygear/naming"
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

func spawnPlayer(s *session) *world.EntityHandle {
	h := coretest.SpawnPlayer(s.Lifecycle)
	pinNotePlayer(h)
	return h
}

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

// droppedItems counts the item entities in the world by item key.
func droppedItems(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if it, ok := e.(*entity.Ent); ok {
			if b, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
				out[naming.ItemKey(b.Item().Item())] += b.Item().Count()
			}
		}
	}
	return out
}

// chestWithBundle puts a chest at pos and an item conduit bundle east of it
// whose west side is set.
func chestWithBundle(tx *world.Tx, pos cube.Pos, mode itemnet.Mode, extract, insert int) cube.Pos {
	tx.SetBlock(pos, block.NewChest(), nil)
	b := conduit.Bundle{Item: 1}
	b.Modes[cube.FaceWest] = mode
	b.Extract[cube.FaceWest], b.Insert[cube.FaceWest] = extract, insert
	at := pos.Add(cube.Pos{1, 0, 0})
	tx.SetBlock(at, b, nil)
	b.Refresh(at, tx)
	return at
}

// run advances n steps of four ticks, calling Fast (which moves the energy)
// after each, as the instance does.
func runSteps(s *session, n int) {
	for range n {
		tick(s, 4)
		s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
	}
}

// runSeconds runs the monitor's second n times.
func runSeconds(s *session, n int) {
	for range n {
		s.World.Do(func(tx *world.Tx) { machine.TickMonitor(s.Session, tx) })
	}
}
