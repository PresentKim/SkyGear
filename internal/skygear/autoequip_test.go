package skygear

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// TestElectricFurnace: it smelts what a furnace smelts and takes nothing else.
func TestElectricFurnace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	if !st.accepts(item.NewStack(block.Cobblestone{}, 1)) || st.accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("the furnace takes the wrong inputs")
	}
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 2))
	tick(s, 2*(machineKinds[KindFurnace].interval+2))
	if outputCount(st, block.Stone{}) == 0 {
		t.Fatal("nothing was smelted")
	}
}

// TestItemVacuum: it takes the dropped items in reach and leaves the ones
// out of reach.
func TestItemVacuum(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindVacuum, "")
	near, far := machinePos.Vec3Centre().Add(mgl64.Vec3{3, 0, 0}), machinePos.Vec3Centre().Add(mgl64.Vec3{30, 0, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: near}, item.NewStack(block.Cobblestone{}, 5)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: far}, item.NewStack(block.Cobblestone{}, 7)))
	})
	tick(s, 2*(machineKinds[KindVacuum].interval+2))
	if got := outputCount(st, block.Cobblestone{}); got != 5 {
		t.Fatalf("collected %d, want the 5 in reach", got)
	}
	left := 0
	s.world.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if ent, ok := e.(*entity.Ent); ok {
				if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
					left += ib.Item().Count()
				}
			}
		}
	})
	if left != 7 {
		t.Fatalf("%d items left lying, want the 7 out of reach", left)
	}
}

// TestTreeFeller: it fells the highest log first, then plants a sapling
// when no log is left.
func TestTreeFeller(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFeller, "")
	log := func() world.Block { return block.Log{Wood: block.OakWood()} }
	trunk := machinePos.Add(cube.Pos{2, 0, 0})
	s.world.Do(func(tx *world.Tx) {
		for dy := range 3 {
			tx.SetBlock(trunk.Add(cube.Pos{0, dy, 0}), log(), nil)
		}
		for dx := -3; dx <= 3; dx += 3 {
			for dz := -3; dz <= 3; dz += 3 {
				at := machinePos.Add(cube.Pos{dx, -1, dz})
				if dx != 0 || dz != 0 {
					tx.SetBlock(at, block.Grass{}, nil)
					tx.SetBlock(at.Side(cube.FaceUp), block.Air{}, nil)
				}
			}
		}
	})
	_ = st.inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	tick(s, 3*(machineKinds[KindFeller].interval+2))
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(trunk.Add(cube.Pos{0, 2, 0})).(block.Log); ok {
			t.Error("the top log was not felled first")
		}
	})
	if outputCount(st, block.Log{Wood: block.OakWood()}) < 2 {
		t.Fatalf("logs collected: %d", outputCount(st, block.Log{Wood: block.OakWood()}))
	}
	tick(s, 6*(machineKinds[KindFeller].interval+2))
	planted := 0
	s.world.Do(func(tx *world.Tx) {
		for dx := -3; dx <= 3; dx += 3 {
			for dz := -3; dz <= 3; dz += 3 {
				if _, ok := tx.Block(machinePos.Add(cube.Pos{dx, 0, dz})).(block.Sapling); ok {
					planted++
				}
			}
		}
	})
	if planted == 0 {
		t.Fatal("no sapling was planted once the logs were gone")
	}
}

func TestTreeFellerBoneMealsSaplings(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFeller, "")
	sapling := machinePos.Add(cube.Pos{2, 0, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(sapling.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(sapling, block.Sapling{Type: block.OakSapling()}, nil)
	})
	if !st.insert(item.NewStack(item.BoneMeal{}, 1)) {
		t.Fatal("feller took no bone meal")
	}
	if got, _ := st.inv().Item(st.kind.charge[0]); got.Count() != 1 {
		t.Fatalf("bone meal slot holds %v", got)
	}
	_ = st.inv().SetItem(st.kind.charge[0], item.NewStack(item.BoneMeal{}, 64))
	tick(s, 40*(machineKinds[KindFeller].interval+2))
	if got, _ := st.inv().Item(st.kind.charge[0]); got.Count() == 64 {
		t.Fatal("no bone meal was used")
	}
	if outputCount(st, block.Log{Wood: block.OakWood()}) == 0 {
		t.Fatal("the sapling did not grow into a tree that was felled")
	}
}

func TestTreeFellerPlantsOnAnySoilInReach(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFeller, "")
	grassy := machinePos.Add(cube.Pos{2, 1, 1})
	bare := machinePos.Add(cube.Pos{-4, -1, 2})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(grassy.Side(cube.FaceDown), block.Grass{}, nil)
		tx.SetBlock(grassy, block.ShortGrass{}, nil)
		tx.SetBlock(bare.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(bare, block.Air{}, nil)
	})
	_ = st.inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	tick(s, 3*(machineKinds[KindFeller].interval+2))
	s.world.Do(func(tx *world.Tx) {
		for _, at := range []cube.Pos{grassy, bare} {
			if _, ok := tx.Block(at).(block.Sapling); !ok {
				t.Errorf("no sapling at %v", at)
			}
		}
	})
}

// TestAutoCrafterMakesARecipe: a sample on the recipe slot chooses the recipe,
// the store takes only its ingredients, and the result comes out.
func TestAutoCrafterMakesARecipe(t *testing.T) {
	s := testSession(t)
	s.recipesFree = true
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrafter, "")
	h := st.inv().Handler()
	if st.accepts(item.NewStack(block.Cobblestone{}, 1)) {
		t.Fatal("the store takes items before a recipe is chosen")
	}
	// A sample put on the recipe slot goes back to the player and chooses.
	ctx := event.C(inventory.Holder(nil))
	h.HandlePlace(ctx, crafterRecipeSlot, machineItem(KindCrusher))
	if !ctx.Cancelled() {
		t.Fatal("the recipe slot kept the sample")
	}
	tick(s, 2)
	if st.recipe == nil {
		t.Fatal("the sample did not choose a recipe")
	}
	if !st.accepts(item.NewStack(block.Cobblestone{}, 1)) || !st.accepts(item.NewStack(item.IronIngot{}, 1)) || st.accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("the store takes the wrong items")
	}
	if shown, _ := st.inv().Item(crafterRecipeSlot); naming.ItemKey(shown.Item()) != naming.ItemKey(Machine{Kind: KindCrusher}) {
		t.Fatalf("the recipe slot shows %v", shown)
	}
	// Not enough yet: nothing is made.
	_ = st.inv().SetItem(0, item.NewStack(block.Cobblestone{}, 6))
	_ = st.inv().SetItem(1, item.NewStack(item.IronIngot{}, 2))
	tick(s, 2*(machineKinds[KindCrafter].interval+2))
	if outputCount(st, Machine{Kind: KindCrusher}) != 0 {
		t.Fatal("it made the result without all the ingredients")
	}
	_ = st.inv().SetItem(2, item.NewStack(item.IronIngot{}, 1))
	tick(s, 2*(machineKinds[KindCrafter].interval+2))
	if outputCount(st, Machine{Kind: KindCrusher}) != 1 {
		t.Fatalf("made %d crushers, want 1", outputCount(st, Machine{Kind: KindCrusher}))
	}
	if left, _ := st.inv().Item(0); !left.Empty() {
		t.Errorf("cobblestone left: %v", left)
	}
	// The recipe is saved with the machine; taking it off the slot clears it.
	back := Machine{Kind: KindCrafter}.DecodeNBT(Machine{Kind: KindCrafter, st: st}.EncodeNBT()).(Machine)
	if back.st.recipeID == "" || back.st.recipeID != st.recipeID {
		t.Fatalf("recipe after NBT: %q", back.st.recipeID)
	}
	ctx = event.C(inventory.Holder(nil))
	h.HandleTake(ctx, crafterRecipeSlot, item.Stack{})
	tick(s, 2)
	if !ctx.Cancelled() || st.recipe != nil {
		t.Fatal("taking the recipe did not clear it")
	}
}

// TestAutoCrafterOnlyMakesResearchedRecipes: without the research the sample
// chooses nothing.
func TestAutoCrafterOnlyMakesResearchedRecipes(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrafter, "")
	st.sample = machineItem(KindCrusher)
	tick(s, 2)
	if st.recipe != nil {
		t.Fatal("chose a recipe that is not researched")
	}
	s.state.Researched = map[string]bool{"stick": true, "pebbles": true, "furnace": true, "hammer": true, "iron": true, "generator": true, "crusher": true}
	st.sample = machineItem(KindCrusher)
	tick(s, 2)
	if st.recipe == nil {
		t.Fatal("did not choose the researched recipe")
	}
}

// TestMachineStatusLines: the panel says why a machine does not go on.
func TestMachineStatusLines(t *testing.T) {
	s := testSession(t)
	s.recipesFree = true
	loadAround(s)
	crusher := placeTestMachine(s, machinePos, KindCrusher, "")
	crusher.progress = crusher.kind.interval
	if lines, _ := machineLines(i18n.Default, Machine{Kind: KindCrusher, st: crusher}); !hasLine(lines, "출력 슬롯 부족") {
		t.Errorf("a blocked crusher says %q", lines)
	}
	crafter := placeTestMachine(s, machinePos.Add(cube.Pos{3, 0, 0}), KindCrafter, "")
	if lines, _ := machineLines(i18n.Default, Machine{Kind: KindCrafter, st: crafter}); !hasLine(lines, "조합법 미설정") {
		t.Errorf("a crafter without a recipe says %q", lines)
	}
	crafter.setRecipe(s.crafterCandidates(machineItem(KindCrusher))[0])
	if lines, _ := machineLines(i18n.Default, Machine{Kind: KindCrafter, st: crafter}); !hasLine(lines, "재료 부족") {
		t.Errorf("a crafter without ingredients says %q", lines)
	}
}
