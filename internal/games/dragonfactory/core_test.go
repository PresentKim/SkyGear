package dragonfactory

import (
	"math/rand/v2"
	"strings"
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
	tick(s, coreTiers[1]+1)
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
	tick(s, coreTiers[1]+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(startCorePos.Side(cube.FaceUp)).(block.Log); !ok {
			t.Fatalf("start core made %T", tx.Block(startCorePos.Side(cube.FaceUp)))
		}
	})
}

// TestCoreRecipesUseSurroundingsAndTier: the most specific recipe the core's
// tier allows wins; a hidden one is discovered when first made.
func TestCoreRecipesUseSurroundingsAndTier(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	stoneCore, t1Sand, t2Sand := cube.Pos{2, 64, 2}, cube.Pos{8, 64, 2}, cube.Pos{14, 64, 2}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, stoneCore, "t1", block.Cobblestone{})
		tx.SetBlock(stoneCore.Add(cube.Pos{1, -1, 0}), baseWater, nil)
		for _, c := range []struct {
			pos cube.Pos
			typ string
		}{{t1Sand, "t1"}, {t2Sand, "t2"}} {
			placeCore(tx, c.pos, c.typ, block.Sand{})
			for _, f := range cube.HorizontalFaces() {
				tx.SetBlock(c.pos.Side(cube.FaceDown).Side(f), baseWater, nil)
			}
		}
	})
	tick(s, coreTiers[2]+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(stoneCore.Side(cube.FaceUp)).(block.Stone); !ok {
			t.Errorf("cobblestone by water made %T, want stone", tx.Block(stoneCore.Side(cube.FaceUp)))
		}
		if _, ok := tx.Block(t1Sand.Side(cube.FaceUp)).(block.Sand); !ok {
			t.Errorf("tier 1 core on wet sand made %T, want sand", tx.Block(t1Sand.Side(cube.FaceUp)))
		}
		if _, ok := tx.Block(t2Sand.Side(cube.FaceUp)).(block.Clay); !ok {
			t.Errorf("tier 2 core on wet sand made %T, want clay", tx.Block(t2Sand.Side(cube.FaceUp)))
		}
	})
	if !s.state.Discovered["smooth_stone"] || !s.state.Discovered["wet_sand"] {
		t.Fatalf("discovered %v", s.state.Discovered)
	}
}

func TestCoreWithoutRecipeWaits(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	bricks, iron := cube.Pos{2, 64, 2}, cube.Pos{6, 64, 2}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, bricks, "t4", block.StoneBricks{})
		placeCore(tx, iron, "t1", block.Iron{})
	})
	tick(s, coreTiers[4]+1)
	s.world.Do(func(tx *world.Tx) {
		for _, p := range []cube.Pos{bricks, iron} {
			if _, air := tx.Block(p.Side(cube.FaceUp)).(block.Air); !air {
				t.Errorf("core at %v made %T without a recipe", p, tx.Block(p.Side(cube.FaceUp)))
			}
		}
		if r, need := matchRecipe(tx, iron, 1); r != nil || need != 3 {
			t.Errorf("iron block base on tier 1: recipe %v, need tier %d", r, need)
		}
	})
}

func TestManyCoresAndTierSpeed(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a, b := cube.Pos{2, 64, 2}, cube.Pos{5, 64, 5}
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, a, "t1", oakLog)
		placeCore(tx, b, "t1", oakLog)
	})
	s.state.TierSpeed = map[int]int{1: maxTierSpeed}
	fast := s.coreInterval(1)
	if fast >= coreTiers[1] || s.coreInterval(2) != coreTiers[2] {
		t.Fatalf("tier 1 speed: %d, tier 2: %d", fast, s.coreInterval(2))
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
		if st, _ := p.Inventory().Item(0); st.Count() != 2 || st.Item().(Core).Type != "t3" {
			t.Fatalf("old iron core item became %v", st)
		}
	})
}

func TestCoreBookHidesUndiscoveredRecipes(t *testing.T) {
	s := testSession(t)
	smooth := recipeByID("smooth_stone")
	body, _ := s.coreBook()
	if strings.Contains(body, smooth.text()) || strings.Contains(body, smooth.Hint) {
		t.Fatalf("undiscovered recipe or its hint shown without a research note:\n%s", body)
	}
	if !strings.Contains(body, recipeByID("oak_log").text()) {
		t.Fatal("a public recipe is missing")
	}
	s.state.Hinted = map[string]bool{"smooth_stone": true}
	if body, _ := s.coreBook(); !strings.Contains(body, smooth.Hint) || strings.Contains(body, smooth.text()) {
		t.Fatal("a hinted recipe must show its hint only")
	}
	s.state.Discovered = map[string]bool{"smooth_stone": true}
	if body, _ := s.coreBook(); !strings.Contains(body, smooth.text()) {
		t.Fatal("a discovered recipe is not shown")
	}
}

func TestCoreRecipesAreValid(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range coreRecipes {
		if seen[r.ID] || r.Tier < 1 || r.Tier > len(coreTiers) || r.Base == nil || r.Output == nil {
			t.Errorf("recipe %q is invalid", r.ID)
		}
		seen[r.ID] = true
		if r.Hidden && r.Hint == "" {
			t.Errorf("hidden recipe %q has no hint", r.ID)
		}
		if _, ok := r.Output.(world.Item); !ok {
			t.Errorf("recipe %q makes a block that is no item", r.ID)
		}
		for _, b := range append([]world.Block{r.Base, r.Output}, func() []world.Block {
			var out []world.Block
			for _, n := range r.Around {
				out = append(out, n.Block)
			}
			return out
		}()...) {
			if _, ok := koreanNames[blockKey(b)]; !ok {
				t.Errorf("recipe %q: no Korean name for %s", r.ID, blockKey(b))
			}
		}
	}
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
	if c, ok := got.Item().(Core); !ok || c.Type != "t2" {
		t.Fatalf("decoded %v", got)
	}
}

// TestGrassBaseCountsAsDirt: a dirt base that turned into grass keeps the
// core making dirt.
func TestGrassBaseCountsAsDirt(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) { placeCore(tx, startCorePos, "t1", block.Grass{}) })
	tick(s, coreTiers[1]+1)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(startCorePos.Side(cube.FaceUp)).(block.Dirt); !ok {
			t.Errorf("core on grass made %T, want dirt", tx.Block(startCorePos.Side(cube.FaceUp)))
		}
	})
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
