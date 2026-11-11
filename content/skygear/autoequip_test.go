package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// TestAutoCrafterMakesARecipe: a sample on the recipe slot chooses the recipe,
// the store takes only its ingredients, and the result comes out.
func TestAutoCrafterMakesARecipe(t *testing.T) {
	s := testSession(t)
	progress.LocalOf(s.Session).RecipesFree = true
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindCrafter)
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
	st := placeTestMachine(s, machinePos, machine.KindCrafter)
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
	crusher := placeTestMachine(s, machinePos, machine.KindCrusher)
	crusher.Progress = crusher.Def.Interval
	if lines, _ := machine.MachineLines(i18n.Default, machine.Machine{Kind: machine.KindCrusher, State: crusher}); !hasLine(lines, "출력 슬롯 부족") {
		t.Errorf("a blocked crusher says %q", lines)
	}
	crafter := placeTestMachine(s, machinePos.Add(cube.Pos{3, 0, 0}), machine.KindCrafter)
	if lines, _ := machine.MachineLines(i18n.Default, machine.Machine{Kind: machine.KindCrafter, State: crafter}); !hasLine(lines, "조합법 미설정") {
		t.Errorf("a crafter without a recipe says %q", lines)
	}
	crafter.SetRecipe(machine.CrafterCandidates(s.Session, machine.Item(machine.KindCrusher))[0])
	if lines, _ := machine.MachineLines(i18n.Default, machine.Machine{Kind: machine.KindCrafter, State: crafter}); !hasLine(lines, "재료 부족") {
		t.Errorf("a crafter without ingredients says %q", lines)
	}
}

// rangeBoxAt is where the range box of the machine at pos stands, and false
// with no box in the world.
func rangeBoxAt(s *session, pos cube.Pos) (at mgl64.Vec3, ok bool) {
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if b, isBox := e.(*machine.RangeBox); isBox {
				at, ok = b.Position(), true
			}
		}
	})
	return at, ok
}
