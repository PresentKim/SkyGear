package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Essences and the mob simulator (docs/design/07-progression-expansion.md §6):
// the island has no mobs, so what they drop comes from the simulator instead. An
// essence of a mob is found by sifting dust with a better mesh; one essence
// charges the simulator for simCharges runs, each run (an interval of energy)
// gives the drops of the mob, rolled.

// Mob kinds of the essences.
const (
	MobZombie   = "zombie"
	MobSkeleton = "skeleton"
	MobCreeper  = "creeper"
	MobSpider   = "spider"
	MobSlime    = "slime"
	MobEnder    = "enderman"
	MobBlaze    = "blaze"
)

// mobKinds are the mobs in menu order.
var mobKinds = []string{MobZombie, MobSkeleton, MobCreeper, MobSpider, MobSlime, MobEnder, MobBlaze}

// simCharges is how many runs one essence gives.
const simCharges = 8

// SimDrop is a possible drop of a run: Min to Max of Item with a Chance percent.
type SimDrop struct {
	Item     world.Item
	Min, Max int
	Chance   int
}

// simDrops are the drops of each mob, drafts.
var simDrops = map[string][]SimDrop{
	MobZombie:   {{item.RottenFlesh{}, 1, 3, 100}},
	MobSkeleton: {{item.Bone{}, 1, 2, 100}, {item.Arrow{}, 1, 2, 60}},
	MobCreeper:  {{item.Gunpowder{}, 1, 2, 100}},
	MobSpider:   {{block.String{}, 1, 2, 100}, {item.SpiderEye{}, 1, 1, 30}},
	MobSlime:    {{item.Slimeball{}, 1, 2, 100}},
	MobEnder:    {{item.EnderPearl{}, 1, 1, 40}},
	MobBlaze:    {{item.BlazeRod{}, 1, 1, 40}},
}

// Essence is the essence of a mob.
type Essence struct{ Mob string }

func (e Essence) EncodeItem() (string, int16) { return "dragonfactory:essence_" + e.Mob, 0 }
func (e Essence) Name() string {
	return i18n.T(i18n.Default, "mob."+e.Mob) + i18n.T(i18n.Default, "name.Essence.suffix")
}
func (Essence) Category() category.Category { return category.Items() }

// maxStackSize keeps essences in stacks of 16.
func (Essence) MaxCount() int { return 16 }

// essenceDrop makes a sieve drop of an essence.
func essenceDrop(mob string, chance int) SieveDrop {
	return SieveDrop{item.NewStack(Essence{Mob: mob}, 1), chance}
}

// essenceExtra is what the meshes find in dust: the essences of weak mobs from
// the iron mesh on, of the rarer ones from the better meshes.
var essenceExtra = map[string]map[string][]SieveDrop{
	"iron":      {itemKey(Dust{}): {essenceDrop(MobZombie, 4), essenceDrop(MobSkeleton, 4), essenceDrop(MobSpider, 3), essenceDrop(MobCreeper, 3)}},
	"gold":      {itemKey(Dust{}): {essenceDrop(MobZombie, 5), essenceDrop(MobSkeleton, 5), essenceDrop(MobSpider, 4), essenceDrop(MobCreeper, 4), essenceDrop(MobSlime, 3)}},
	"diamond":   {itemKey(Dust{}): {essenceDrop(MobZombie, 5), essenceDrop(MobSkeleton, 5), essenceDrop(MobSpider, 5), essenceDrop(MobCreeper, 5), essenceDrop(MobSlime, 5), essenceDrop(MobEnder, 3)}},
	"netherite": {itemKey(Dust{}): {essenceDrop(MobSkeleton, 6), essenceDrop(MobCreeper, 6), essenceDrop(MobSlime, 6), essenceDrop(MobEnder, 5), essenceDrop(MobBlaze, 4)}},
}

// acceptsEssence reports whether the simulator takes it.
func acceptsEssence(it item.Stack) bool {
	_, ok := it.Item().(Essence)
	return ok
}

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
		e := in.Item().(Essence)
		_ = st.inv().SetItem(slot, in.Grow(-1))
		st.mob, st.charges = e.Mob, simCharges
	}
	if st.progress < st.kind.interval {
		if st.consume() {
			st.progress++
		}
		return
	}
	var drops []item.Stack
	var all []item.Stack
	for _, d := range simDrops[st.mob] {
		all = append(all, item.NewStack(d.Item, d.Max))
		if s.rand.IntN(100) < d.Chance {
			drops = append(drops, item.NewStack(d.Item, d.Min+s.rand.IntN(d.Max-d.Min+1)))
		}
	}
	if !fits(st.inv(), st.kind.output, all) {
		return
	}
	for _, d := range drops {
		put(st.inv(), st.kind.output, d)
	}
	st.progress = 0
	if st.charges--; st.charges == 0 {
		st.mob = ""
	}
}

func init() {
	for _, m := range mobKinds {
		world.RegisterItem(Essence{Mob: m})
	}
}
