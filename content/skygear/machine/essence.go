package machine

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Essences and the mob simulator:
// the island has no mobs, so what they drop comes from the simulator instead. An
// essence of a mob is found by sifting dust with a better mesh; one essence
// charges the simulator for sgitem.SimCharges runs, each run (an interval of energy)
// gives the drops of the mob, rolled.

func init() {
	for _, m := range sgitem.MobKinds {
		world.RegisterItem(sgitem.Essence{Mob: m})
	}
}

// runSimulator runs the mob simulator: a charge (from an essence in the input
// slot) is spent every interval, and the drops of the mob are rolled into the
// output; it waits for room, and for energy.
func runSimulator(s *core.Session, st *MachineState) {
	if st.Charges == 0 {
		slot, in, ok := st.FirstInput()
		if !ok {
			st.Progress = 0
			return
		}
		e := in.Item().(sgitem.Essence)
		_ = st.Inv().SetItem(slot, in.Grow(-1))
		st.Mob, st.Charges = e.Mob, sgitem.SimCharges
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	var drops []item.Stack
	var all []item.Stack
	for _, d := range sgitem.SimDrops[st.Mob] {
		all = append(all, item.NewStack(d.Item, d.Max))
		if s.Rand.IntN(100) < d.Chance {
			drops = append(drops, item.NewStack(d.Item, d.Min+s.Rand.IntN(d.Max-d.Min+1)))
		}
	}
	if !st.RoomFor(all) {
		return
	}
	for _, d := range drops {
		st.Emit(d)
	}
	st.Progress = 0
	if st.Charges--; st.Charges == 0 {
		st.Mob = ""
	}
}
