package machine

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// The alloy smelter: two input
// slots, one output; it needs energy. A recipe wants a count of each of two
// ingredients (in either slot) and makes a count of its product.

// runAlloy works on the recipe the two input slots make: it needs energy every
// tick of the interval, and finishes when the output has room.
func (st *MachineState) runAlloy() {
	a, _ := st.Inv().Item(st.Def.Input[0])
	b, _ := st.Inv().Item(st.Def.Input[1])
	r, ok := machinerecipe.MatchAlloy(a, b)
	if !ok {
		st.Progress = 0
		return
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	out := item.NewStack(r.Out, r.OutCount)
	if !st.RoomFor([]item.Stack{out}) {
		return
	}
	// take the ingredients out of the slot that holds each
	take := func(want world.Item, n int) {
		for _, slot := range st.Def.Input {
			it, _ := st.Inv().Item(slot)
			if !it.Empty() && naming.ItemKey(it.Item()) == naming.ItemKey(want) && it.Count() >= n {
				_ = st.Inv().SetItem(slot, it.Grow(-n))
				return
			}
		}
	}
	take(r.A, r.CountA)
	take(r.B, r.CountB)
	st.Emit(out)
	st.Progress = 0
}
