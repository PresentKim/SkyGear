package dragonfactory

import (
	"math/rand/v2"
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

// placeCore puts a core of a type on a base block (on bedrock, so sand and
// gravel bases do not fall).
func placeCore(tx *world.Tx, pos cube.Pos, typ string, base world.Block) {
	tx.SetBlock(pos.Add(cube.Pos{0, -2, 0}), block.Bedrock{}, nil)
	tx.SetBlock(pos.Side(cube.FaceDown), base, nil)
	tx.SetBlock(pos, Core{Type: typ}, nil)
}

func TestCoreGeneratesAboveItself(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { placeCore(tx, startCorePos, "t1", block.Cobblestone{}) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, coreBaseInterval+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(above).(block.Cobblestone); !ok {
			t.Fatalf("core on cobblestone made %T", tx.Block(above))
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

// TestNewSaveCoreMakesLogs: the start core stands on an oak log.
func TestNewSaveCoreMakesLogs(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { (&Game{}).Init(tx, nil) })
	tick(s, coreBaseInterval+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(startCorePos.Side(cube.FaceUp)).(block.Log); !ok {
			t.Fatalf("start core made %T", tx.Block(startCorePos.Side(cube.FaceUp)))
		}
	})
}

// TestCoreWithoutRecipeWaits: only oak logs and cobblestone are bases;
// an old tier 4 core on another block turns into the core and waits too.
func TestCoreWithoutRecipeWaits(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	bricks, sand := cube.Pos{2, 64, 2}, cube.Pos{6, 64, 2}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, bricks, "t4", block.StoneBricks{})
		placeCore(tx, sand, "t1", block.Sand{})
	})
	tick(s, coreBaseInterval+2)
	s.world.Do(func(tx *world.Tx) {
		for _, p := range []cube.Pos{bricks, sand} {
			if _, air := tx.Block(p.Side(cube.FaceUp)).(block.Air); !air {
				t.Errorf("core at %v made %T without a recipe", p, tx.Block(p.Side(cube.FaceUp)))
			}
		}
		if c := tx.Block(bricks).(Core); c.Type != "t1" {
			t.Errorf("old tier 4 core is %q", c.Type)
		}
	})
}

func TestManyCoresAndSpeed(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a, b := cube.Pos{2, 64, 2}, cube.Pos{5, 64, 5}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, a, "t1", oakLog)
		placeCore(tx, b, "t1", oakLog)
	})
	s.state.TierSpeed = map[int]int{1: maxTierSpeed}
	fast := s.coreInterval()
	if fast >= coreBaseInterval {
		t.Fatalf("interval at full speed: %d", fast)
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
	s.world.Do(func(tx *world.Tx) { placeCore(tx, startCorePos, "earth", block.Grass{}) })
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if c, ok := tx.Block(startCorePos).(Core); !ok || c.Type != "t1" {
			t.Fatalf("old earth core is %#v", tx.Block(startCorePos))
		}
		if _, ok := tx.Block(startCorePos.Side(cube.FaceDown)).(block.Log); !ok {
			t.Fatal("the old start core did not get its log base")
		}
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(0, item.NewStack(Core{Type: "iron"}, 2))
		// A chest item is not comparable: converting must not compare it.
		_ = p.Inventory().SetItem(1, item.NewStack(block.NewChest(), 1))
		convertLegacyCores(p.Inventory())
		if st, _ := p.Inventory().Item(0); st.Count() != 2 || st.Item().(Core).Type != "t1" {
			t.Fatalf("old iron core item became %v", st)
		}
	})
}

func TestCoreRecipesAreValid(t *testing.T) {
	for _, r := range coreRecipes {
		if _, ok := r.Output.(world.Item); !ok {
			t.Errorf("recipe %q makes a block that is no item", r.ID)
		}
		for _, b := range []world.Block{r.Base, r.Output} {
			if _, ok := koreanNames[blockKey(b)]; !ok {
				t.Errorf("recipe %q: no Korean name for %s", r.ID, blockKey(b))
			}
		}
	}
}

func TestCoreMenuBuysSpeed(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.state.CorePoints = 2
		_, a := s.coreMenu()
		_, b := s.coreMenu() // a second, stale copy
		press(t, p, tx, a, "§b코어 속도 Lv.0")
		press(t, p, tx, b, "§b코어 속도 Lv.0")
		if s.state.TierSpeed[1] != 1 || s.state.CorePoints != 1 {
			t.Fatalf("speed %d points %d", s.state.TierSpeed[1], s.state.CorePoints)
		}
		_, c := s.coreMenu()
		press(t, p, tx, c, "§b코어 속도 Lv.1")
		if s.state.TierSpeed[1] != 1 {
			t.Fatal("bought Lv.2 without enough points")
		}
	})
}

func TestCoreActivateShowsInfo(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { placeCore(tx, startCorePos, "t1", oakLog) })
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
	s.world.Do(func(tx *world.Tx) { placeCore(tx, startCorePos, "t1", oakLog) })
	above := startCorePos.Side(cube.FaceUp)
	tick(s, coreBaseInterval+1)
	tick(s, 200) // the generated block stays for a while
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(above, nil, nil) })
	tick(s, 10)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(above).(block.Air); !air {
			t.Fatal("core regenerated right after the block was mined")
		}
	})
	tick(s, coreBaseInterval)
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
	if c, ok := got.Item().(Core); !ok || c.Type != "t1" {
		t.Fatalf("decoded %v", got)
	}
}

// TestNoGrassUnderCore: grass does not spread to dirt under a core, and
// grass under a core turns back into dirt (Dragonfly patch).
func TestNoGrassUnderCore(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	dirtBase, grassBase, source := cube.Pos{2, 64, 2}, cube.Pos{4, 64, 2}, cube.Pos{3, 63, 2}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, dirtBase, "t1", block.Dirt{})
		placeCore(tx, grassBase, "t1", block.Grass{})
		tx.SetBlock(source, block.Grass{}, nil)
		r := rand.New(rand.NewPCG(1, 2))
		for range 500 {
			block.Grass{}.RandomTick(source, tx, r)
			if g, ok := tx.Block(grassBase.Side(cube.FaceDown)).(block.Grass); ok {
				g.RandomTick(grassBase.Side(cube.FaceDown), tx, r)
			}
		}
		if _, ok := tx.Block(dirtBase.Side(cube.FaceDown)).(block.Dirt); !ok {
			t.Errorf("dirt under a core became %T", tx.Block(dirtBase.Side(cube.FaceDown)))
		}
		if _, ok := tx.Block(grassBase.Side(cube.FaceDown)).(block.Dirt); !ok {
			t.Errorf("grass under a core stayed %T", tx.Block(grassBase.Side(cube.FaceDown)))
		}
	})
}
