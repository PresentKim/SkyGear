package skygear

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// The alloy smelter: two input
// slots, one output; it needs energy. A recipe wants a count of each of two
// ingredients (in either slot) and makes a count of its product.

// runAlloy works on the recipe the two input slots make: it needs energy every
// tick of the interval, and finishes when the output has room.
func (st *machineState) runAlloy() {
	a, _ := st.inv().Item(st.kind.input[0])
	b, _ := st.inv().Item(st.kind.input[1])
	r, ok := machinerecipe.MatchAlloy(a, b)
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
			if !it.Empty() && naming.ItemKey(it.Item()) == naming.ItemKey(want) && it.Count() >= n {
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
