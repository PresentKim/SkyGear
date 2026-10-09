package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
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
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "stone"}, nil) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, coreTiers[1]+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(above).(block.Stone); !ok {
			t.Fatalf("stone core made %T", tx.Block(above))
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

func TestManyCoresAndTierSpeed(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a, b := cube.Pos{2, 64, 2}, cube.Pos{5, 64, 5}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(a, Core{Type: "wood"}, nil)
		tx.SetBlock(b, Core{Type: "wood"}, nil)
	})
	s.state.TierSpeed = map[int]int{1: maxTierSpeed}
	fast := s.coreInterval(coreTypes["wood"])
	if fast >= coreTiers[1] || s.coreInterval(coreTypes["coal"]) != coreTiers[2] {
		t.Fatalf("tier 1 speed: %d, tier 2: %d", fast, s.coreInterval(coreTypes["coal"]))
	}
	tick(s, fast+1)
	s.world.Do(func(tx *world.Tx) {
		for _, p := range []cube.Pos{a, b} {
			if _, ok := tx.Block(p.Side(cube.FaceUp)).(block.Log); !ok {
				t.Fatalf("core at %v made %T", p, tx.Block(p.Side(cube.FaceUp)))
			}
		}
	})
}

func TestLegacyCoreBecomesNewCore(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "earth"}, nil) })
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if c, ok := tx.Block(startCorePos).(Core); !ok || c.Type != "wood" {
			t.Fatalf("old earth core is %#v", tx.Block(startCorePos))
		}
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(0, item.NewStack(Core{Type: "life"}, 2))
		convertLegacyCores(p.Inventory())
		if st, _ := p.Inventory().Item(0); st.Count() != 2 || st.Item().(Core).Type != "melon" {
			t.Fatalf("old life core item became %v", st)
		}
	})
}

func TestCoreMenuBuysTierSpeed(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.state.CorePoints = 2
		_, a := s.coreMenu()
		_, b := s.coreMenu() // a second, stale copy
		press(t, p, tx, a, "§b1티어 코어 속도 Lv.0")
		press(t, p, tx, b, "§b1티어 코어 속도 Lv.0")
		if s.state.TierSpeed[1] != 1 || s.state.CorePoints != 1 {
			t.Fatalf("speed %d points %d", s.state.TierSpeed[1], s.state.CorePoints)
		}
		_, c := s.coreMenu()
		press(t, p, tx, c, "§b1티어 코어 속도 Lv.1")
		if s.state.TierSpeed[1] != 1 {
			t.Fatal("bought Lv.2 without enough points")
		}
	})
}

func TestCoreActivateShowsInfo(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "wood"}, nil) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		c := tx.Block(startCorePos).(Core)
		if !c.Activate(startCorePos, cube.FaceUp, tx, p, nil) {
			t.Fatal("core activation not handled")
		}
	})
}

// TestCoreWaitsFullIntervalAfterMining: time spent with the block above
// still in place does not count towards the next block.
func TestCoreWaitsFullIntervalAfterMining(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(startCorePos, Core{Type: "wood"}, nil) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, coreTiers[1]+1)
	tick(s, 200) // the generated block stays for a while
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(above, nil, nil) })
	tick(s, 10)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); !air {
			t.Fatal("core regenerated right after the block was mined")
		}
	})
	tick(s, coreTiers[1])
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); air {
			t.Fatal("core did not regenerate after a full interval")
		}
	})
}

// TestLegacyCoreInLostAndFound: an old core kept in the lost-and-found box
// comes back as the core that replaces it.
func TestLegacyCoreInLostAndFound(t *testing.T) {
	got := decodeItem(encodeItem(item.NewStack(Core{Type: "life"}, 1)))
	if c, ok := got.Item().(Core); !ok || c.Type != "melon" {
		t.Fatalf("decoded %v", got)
	}
}
