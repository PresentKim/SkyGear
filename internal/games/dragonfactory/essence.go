package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// Essences and the mob simulator (docs/design/07-progression-expansion.md §6):
// the island has no mobs, so what they drop comes from the simulator instead. An
// essence of a mob is found by sifting dust with a better mesh; one essence
// charges the simulator for simCharges runs, each run (an interval of energy)
// gives the drops of the mob, rolled.

// runSimulator runs the mob simulator: a charge (from an essence in the input
// slot) is spent every interval, and the drops of the mob are rolled into the
// output; it waits for room, and for energy.
func (s *session) runSimulator(st *machineState) {
	if st.charges == 0 {
		slot, in, ok := st.firstInput()
		if !ok {
			st.progress = 0
			return
		}
		e := in.Item().(dfitem.Essence)
		_ = st.inv().SetItem(slot, in.Grow(-1))
		st.mob, st.charges = e.Mob, dfitem.SimCharges
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	var drops []item.Stack
	var all []item.Stack
	for _, d := range dfitem.SimDrops[st.mob] {
		all = append(all, item.NewStack(d.Item, d.Max))
		if s.rand.IntN(100) < d.Chance {
			drops = append(drops, item.NewStack(d.Item, d.Min+s.rand.IntN(d.Max-d.Min+1)))
		}
	}
	if !st.roomFor(all) {
		return
	}
	for _, d := range drops {
		st.emit(d)
	}
	st.progress = 0
	if st.charges--; st.charges == 0 {
		st.mob = ""
	}
}

func init() {
	for _, m := range dfitem.MobKinds {
		world.RegisterItem(dfitem.Essence{Mob: m})
	}
}
