package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// The alloy smelter (docs/design/07-progression-expansion.md §4): two input
// slots, one output; it needs energy. A recipe wants a count of each of two
// ingredients (in either slot) and makes a count of its product.

// AlloyRecipe is one recipe of the alloy smelter.
type AlloyRecipe struct {
	A, B           world.Item
	CountA, CountB int
	Out            world.Item
	OutCount       int
}

// alloyRecipes are the recipes, drafts.
var alloyRecipes = []AlloyRecipe{
	{A: item.CopperIngot{}, CountA: 3, B: Ingot{Metal: MetalTin}, CountB: 1, Out: Ingot{Metal: MetalBronze}, OutCount: 4},
	{A: item.GoldIngot{}, CountA: 1, B: Ingot{Metal: MetalSilver}, CountB: 1, Out: Ingot{Metal: MetalElectrum}, OutCount: 2},
	{A: item.IronIngot{}, CountA: 2, B: Ingot{Metal: MetalNickel}, CountB: 1, Out: Ingot{Metal: MetalInvar}, OutCount: 3},
	{A: item.IronIngot{}, CountA: 1, B: item.Coal{}, CountB: 4, Out: Ingot{Metal: MetalSteel}, OutCount: 1},
}

// alloyAccepts reports whether an item is an ingredient of some recipe.
func alloyAccepts(it item.Stack) bool {
	k := itemKey(it.Item())
	for _, r := range alloyRecipes {
		if itemKey(r.A) == k || itemKey(r.B) == k {
			return true
		}
	}
	return false
}

// matchAlloy returns the recipe the two slots make (in either order), if any.
func matchAlloy(a, b item.Stack) (AlloyRecipe, bool) {
	has := func(it item.Stack, want world.Item, n int) bool {
		return !it.Empty() && itemKey(it.Item()) == itemKey(want) && it.Count() >= n
	}
	for _, r := range alloyRecipes {
		if (has(a, r.A, r.CountA) && has(b, r.B, r.CountB)) || (has(a, r.B, r.CountB) && has(b, r.A, r.CountA)) {
			return r, true
		}
	}
	return AlloyRecipe{}, false
}

// runAlloy works on the recipe the two input slots make: it needs energy every
// tick of the interval, and finishes when the output has room.
func (st *machineState) runAlloy() {
	a, _ := st.inv().Item(st.kind.input[0])
	b, _ := st.inv().Item(st.kind.input[1])
	r, ok := matchAlloy(a, b)
	if !ok {
		st.progress = 0
		return
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	out := item.NewStack(r.Out, r.OutCount)
	if !st.roomFor([]item.Stack{out}) {
		return
	}
	// take the ingredients out of the slot that holds each
	take := func(want world.Item, n int) {
		for _, slot := range st.kind.input {
			it, _ := st.inv().Item(slot)
			if !it.Empty() && itemKey(it.Item()) == itemKey(want) && it.Count() >= n {
				_ = st.inv().SetItem(slot, it.Grow(-n))
				return
			}
		}
	}
	take(r.A, r.CountA)
	take(r.B, r.CountB)
	st.emit(out)
	st.progress = 0
}
