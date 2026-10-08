package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func loadAround(s *session) {
	s.world.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.world, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
}

func spawnPlayer(s *session) *world.EntityHandle {
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Position: mgl64.Vec3{0.5, 64, 2.5}})
	s.world.Do(func(tx *world.Tx) { tx.AddEntity(h) })
	return h
}

// withPlayer runs f with the player in a transaction. A panic in f fails
// the test (Dragonfly would only log it).
func withPlayer(t *testing.T, s *session, h *world.EntityHandle, f func(p *player.Player, tx *world.Tx)) {
	t.Helper()
	finished := false
	s.world.Do(func(tx *world.Tx) {
		e, _ := h.Entity(tx)
		f(e.(*player.Player), tx)
		finished = true
	})
	if !finished {
		t.Fatal("transaction did not finish (panicked)")
	}
}

func tick(s *session, n int) {
	for range n {
		s.world.AdvanceTick()
	}
}

func TestCoreGeneratesAboveItself(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, 61)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); air {
			t.Fatal("core did not generate a block after its interval")
		}
		tx.SetBlock(above, nil, nil)
	})
	tick(s, 30)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); !air {
			t.Fatal("core generated again before the interval passed")
		}
	})
}

func TestStaleCoreIsRemoved(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	other := cube.Pos{5, 64, 5}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil)
		tx.SetBlock(other, Core{Type: "earth", Generation: 1}, nil) // a second copy
	})
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(other).(Core); ok {
			t.Fatal("a second core of the same type was not removed")
		}
		if _, ok := tx.Block(startCorePos).(Core); !ok {
			t.Fatal("the recorded core was removed")
		}
	})
}

func TestReissueAndPlace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// The recorded core is placed: no reissue.
		tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil)
		s.reissueCore(p, tx, "earth")
		if s.state.Cores["earth"].Generation != 1 {
			t.Fatal("placed core was reissued")
		}
		// Mine it: the record becomes unplaced and the item has generation 1.
		s.breakCore(Core{Type: "earth", Generation: 1}, startCorePos)
		tx.SetBlock(startCorePos, nil, nil)
		old := coreItem("earth", 1)
		_, _ = p.Inventory().AddItem(old)

		s.reissueCore(p, tx, "earth")
		if s.state.Cores["earth"].Generation != 2 {
			t.Fatalf("generation = %d, want 2", s.state.Cores["earth"].Generation)
		}
		n := 0
		for _, st := range p.Inventory().Items() {
			if _, gen, ok := coreItemGeneration(st); ok {
				n++
				if gen != 2 {
					t.Fatalf("stale core item of generation %d kept", gen)
				}
			}
		}
		if n != 1 {
			t.Fatalf("player has %d core items, want 1", n)
		}

		if s.placeCore(p, tx, cube.Pos{3, 64, 3}, old) {
			t.Fatal("an outdated core item could be placed")
		}
		if !s.placeCore(p, tx, cube.Pos{3, 64, 3}, coreItem("earth", 2)) {
			t.Fatal("the current core item could not be placed")
		}
		if s.placeCore(p, tx, cube.Pos{4, 64, 3}, coreItem("earth", 2)) {
			t.Fatal("a second core of the same type could be placed")
		}
	})
	s.world.Do(func(tx *world.Tx) {
		if c, ok := tx.Block(cube.Pos{3, 64, 3}).(Core); !ok || c.Generation != 2 {
			t.Fatalf("placed core = %#v", tx.Block(cube.Pos{3, 64, 3}))
		}
	})
}

// TestCoreWaitsFullIntervalAfterMining: time spent with the block above
// still in place does not count towards the next block.
func TestCoreWaitsFullIntervalAfterMining(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, 61)
	tick(s, 200) // the generated block stays for a while
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(above, nil, nil) })
	tick(s, 10)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); !air {
			t.Fatal("core regenerated right after the block was mined")
		}
	})
	tick(s, 60)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); air {
			t.Fatal("core did not regenerate after a full interval")
		}
	})
}
