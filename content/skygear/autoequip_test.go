package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// TestElectricFurnace: it smelts what a furnace smelts and takes nothing else.
func TestElectricFurnace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	if !st.Accepts(item.NewStack(block.Cobblestone{}, 1)) || st.Accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("the furnace takes the wrong inputs")
	}
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 2))
	tick(s, 2*(machine.Kinds[machine.KindFurnace].Interval+2))
	if machine.OutputCount(st, block.Stone{}) == 0 {
		t.Fatal("nothing was smelted")
	}
}

// TestItemVacuum: it takes the dropped items in reach and leaves the ones
// out of reach.
func TestItemVacuum(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindVacuum, "")
	near, far := machinePos.Vec3Centre().Add(mgl64.Vec3{3, 0, 0}), machinePos.Vec3Centre().Add(mgl64.Vec3{30, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: near}, item.NewStack(block.Cobblestone{}, 5)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: far}, item.NewStack(block.Cobblestone{}, 7)))
	})
	tick(s, 2*(machine.Kinds[machine.KindVacuum].Interval+2))
	if got := machine.OutputCount(st, block.Cobblestone{}); got != 5 {
		t.Fatalf("collected %d, want the 5 in reach", got)
	}
	left := 0
	s.World.Do(func(tx *world.Tx) {
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
	st := placeTestMachine(s, machinePos, machine.KindFeller, "")
	log := func() world.Block { return block.Log{Wood: block.OakWood()} }
	trunk := machinePos.Add(cube.Pos{2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
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
	_ = st.Inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	tick(s, 3*(machine.Kinds[machine.KindFeller].Interval+2))
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(trunk.Add(cube.Pos{0, 2, 0})).(block.Log); ok {
			t.Error("the top log was not felled first")
		}
	})
	if machine.OutputCount(st, block.Log{Wood: block.OakWood()}) < 2 {
		t.Fatalf("logs collected: %d", machine.OutputCount(st, block.Log{Wood: block.OakWood()}))
	}
	tick(s, 6*(machine.Kinds[machine.KindFeller].Interval+2))
	planted := 0
	s.World.Do(func(tx *world.Tx) {
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
	st := placeTestMachine(s, machinePos, machine.KindFeller, "")
	sapling := machinePos.Add(cube.Pos{2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(sapling.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(sapling, block.Sapling{Type: block.OakSapling()}, nil)
	})
	if !st.Insert(item.NewStack(item.BoneMeal{}, 1)) {
		t.Fatal("feller took no bone meal")
	}
	if got, _ := st.Inv().Item(st.Def.Charge[0]); got.Count() != 1 {
		t.Fatalf("bone meal slot holds %v", got)
	}
	_ = st.Inv().SetItem(st.Def.Charge[0], item.NewStack(item.BoneMeal{}, 64))
	tick(s, 40*(machine.Kinds[machine.KindFeller].Interval+2))
	if got, _ := st.Inv().Item(st.Def.Charge[0]); got.Count() == 64 {
		t.Fatal("no bone meal was used")
	}
	if machine.OutputCount(st, block.Log{Wood: block.OakWood()}) == 0 {
		t.Fatal("the sapling did not grow into a tree that was felled")
	}
}

func TestTreeFellerPlantsOnAnySoilInReach(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFeller, "")
	grassy := machinePos.Add(cube.Pos{2, 1, 1})
	bare := machinePos.Add(cube.Pos{-4, -1, 2})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(grassy.Side(cube.FaceDown), block.Grass{}, nil)
		tx.SetBlock(grassy, block.ShortGrass{}, nil)
		tx.SetBlock(bare.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(bare, block.Air{}, nil)
	})
	_ = st.Inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	tick(s, 3*(machine.Kinds[machine.KindFeller].Interval+2))
	s.World.Do(func(tx *world.Tx) {
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
	progress.LocalOf(s.Session).RecipesFree = true
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindCrafter, "")
	h := st.Inv().Handler()
	if st.Accepts(item.NewStack(block.Cobblestone{}, 1)) {
		t.Fatal("the store takes items before a recipe is chosen")
	}
	// A sample put on the recipe slot goes back to the player and chooses.
	ctx := event.C(inventory.Holder(nil))
	h.HandlePlace(ctx, machine.CrafterRecipeSlot, machine.Item(machine.KindCrusher))
	if !ctx.Cancelled() {
		t.Fatal("the recipe slot kept the sample")
	}
	tick(s, 2)
	if st.Recipe == nil {
		t.Fatal("the sample did not choose a recipe")
	}
	if !st.Accepts(item.NewStack(block.Cobblestone{}, 1)) || !st.Accepts(item.NewStack(item.IronIngot{}, 1)) || st.Accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("the store takes the wrong items")
	}
	if shown, _ := st.Inv().Item(machine.CrafterRecipeSlot); naming.ItemKey(shown.Item()) != naming.ItemKey(machine.Machine{Kind: machine.KindCrusher}) {
		t.Fatalf("the recipe slot shows %v", shown)
	}
	// Not enough yet: nothing is made.
	_ = st.Inv().SetItem(0, item.NewStack(block.Cobblestone{}, 6))
	_ = st.Inv().SetItem(1, item.NewStack(item.IronIngot{}, 2))
	tick(s, 2*(machine.Kinds[machine.KindCrafter].Interval+2))
	if machine.OutputCount(st, machine.Machine{Kind: machine.KindCrusher}) != 0 {
		t.Fatal("it made the result without all the ingredients")
	}
	_ = st.Inv().SetItem(2, item.NewStack(item.IronIngot{}, 1))
	tick(s, 2*(machine.Kinds[machine.KindCrafter].Interval+2))
	if machine.OutputCount(st, machine.Machine{Kind: machine.KindCrusher}) != 1 {
		t.Fatalf("made %d crushers, want 1", machine.OutputCount(st, machine.Machine{Kind: machine.KindCrusher}))
	}
	if left, _ := st.Inv().Item(0); !left.Empty() {
		t.Errorf("cobblestone left: %v", left)
	}
	// The recipe is saved with the machine; taking it off the slot clears it.
	back := machine.Machine{Kind: machine.KindCrafter}.DecodeNBT(machine.Machine{Kind: machine.KindCrafter, State: st}.EncodeNBT()).(machine.Machine)
	if back.State.RecipeID == "" || back.State.RecipeID != st.RecipeID {
		t.Fatalf("recipe after NBT: %q", back.State.RecipeID)
	}
	ctx = event.C(inventory.Holder(nil))
	h.HandleTake(ctx, machine.CrafterRecipeSlot, item.Stack{})
	tick(s, 2)
	if !ctx.Cancelled() || st.Recipe != nil {
		t.Fatal("taking the recipe did not clear it")
	}
}

// TestAutoCrafterOnlyMakesResearchedRecipes: without the research the sample
// chooses nothing.
func TestAutoCrafterOnlyMakesResearchedRecipes(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindCrafter, "")
	st.Sample = machine.Item(machine.KindCrusher)
	tick(s, 2)
	if st.Recipe != nil {
		t.Fatal("chose a recipe that is not researched")
	}
	progress.StateOf(s.Session).Researched = map[string]bool{"stick": true, "pebbles": true, "furnace": true, "hammer": true, "iron": true, "generator": true, "crusher": true}
	st.Sample = machine.Item(machine.KindCrusher)
	tick(s, 2)
	if st.Recipe == nil {
		t.Fatal("did not choose the researched recipe")
	}
}

// TestMachineStatusLines: the panel says why a machine does not go on.
func TestMachineStatusLines(t *testing.T) {
	s := testSession(t)
	progress.LocalOf(s.Session).RecipesFree = true
	loadAround(s)
	crusher := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	crusher.Progress = crusher.Def.Interval
	if lines, _ := machineLines(i18n.Default, machine.Machine{Kind: machine.KindCrusher, State: crusher}); !hasLine(lines, "출력 슬롯 부족") {
		t.Errorf("a blocked crusher says %q", lines)
	}
	crafter := placeTestMachine(s, machinePos.Add(cube.Pos{3, 0, 0}), machine.KindCrafter, "")
	if lines, _ := machineLines(i18n.Default, machine.Machine{Kind: machine.KindCrafter, State: crafter}); !hasLine(lines, "조합법 미설정") {
		t.Errorf("a crafter without a recipe says %q", lines)
	}
	crafter.SetRecipe(machine.CrafterCandidates(s.Session, machine.Item(machine.KindCrusher))[0])
	if lines, _ := machineLines(i18n.Default, machine.Machine{Kind: machine.KindCrafter, State: crafter}); !hasLine(lines, "재료 부족") {
		t.Errorf("a crafter without ingredients says %q", lines)
	}
}
