package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The auto harvester harvests the grown crops around it on its own level,
// harvestRadius blocks each way: crops are left as young plants, stem fruits
// are cut and blooming magic plants go back to buds. The products go to its
// output slots.

// harvestable reports whether b at pos is ready to harvest, and returns the
// block to leave in its place (a replanted crop or air).
func harvestable(b world.Block) (world.Block, bool) {
	switch c := b.(type) {
	case block.WheatSeeds:
		if c.Growth >= 7 {
			c.Growth = 0
			return c, true
		}
	case block.Carrot:
		if c.Growth >= 7 {
			c.Growth = 0
			return c, true
		}
	case block.Potato:
		if c.Growth >= 7 {
			c.Growth = 0
			return c, true
		}
	case block.BeetrootSeeds:
		if c.Growth >= 7 {
			c.Growth = 0
			return c, true
		}
	case block.Melon, block.Pumpkin:
		return block.Air{}, true
	}
	return nil, false
}

// runHarvester harvests one grown crop per interval.
// The cycle runs on whether or not a crop is ready, so the gauge keeps
// turning.
func (s *session) runHarvester(st *machine.State, pos cube.Pos, tx *world.Tx) {
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	st.Progress = 0
	radius := st.Reach(machine.KindHarvester)
	for x := -radius; x <= radius; x++ {
		for z := -radius; z <= radius; z++ {
			at := pos.Add(cube.Pos{x, 0, z})
			b := tx.Block(at)
			if kind, _, ok := cropOf(b); ok {
				if _, known := s.state.Crops[cropKey(at)]; !known {
					// Planted before crops were scheduled: start its timer.
					s.plantCrop(at, kind)
				}
			}
			if mp, isPlant := b.(MagicPlant); isPlant && mp.Bloom {
				if s.harvestMagicPlant(st, tx, at, mp) {
					st.Done++
					return
				}
				continue
			}
			after, ok := harvestable(b)
			if !ok {
				continue
			}
			var drops []item.Stack
			if br, ok := b.(block.Breakable); ok && br.BreakInfo().Drops != nil {
				drops = br.BreakInfo().Drops(item.ToolNone{}, nil)
			}
			if !st.RoomFor(drops) {
				return
			}
			after, drops = replantSeed(st, after, drops)
			tx.SetBlock(at, after, nil)
			if kind, _, ok := cropOf(after); ok {
				s.plantCrop(at, kind)
			}
			tx.AddParticle(at.Vec3Centre(), particle.BlockBreak{Block: b})
			for _, it := range drops {
				if !it.Empty() {
					st.Emit(it)
				}
			}
			st.Done++
			return
		}
	}
}

// harvestMagicPlant picks a blooming magic plant: the yield goes to the
// harvester's output and the plant goes back to a bud. A plant whose yield
// does not fit is left for the next cycle.
func (s *session) harvestMagicPlant(st *machine.State, tx *world.Tx, at cube.Pos, mp MagicPlant) bool {
	drops, ok := s.plantDrops(mp)
	if !ok || !st.RoomFor(drops) {
		return false
	}
	for _, it := range drops {
		if !it.Empty() {
			st.Emit(it)
		}
	}
	s.resetPlant(tx, at, mp)
	return true
}

// replantSeed takes the seed a harvested crop is replanted with: one from
// its drops, else one from the harvester's output. Without a seed the crop
// is not replanted (after becomes air). Stems (melon, pumpkin) need none.
func replantSeed(st *machine.State, after world.Block, drops []item.Stack) (world.Block, []item.Stack) {
	if _, _, crop := cropOf(after); !crop {
		return after, drops
	}
	seedItem, ok := after.(world.Item)
	if !ok {
		return after, drops
	}
	seed := item.NewStack(seedItem, 1)
	for i, it := range drops {
		if it.Comparable(seed) && it.Count() > 0 {
			drops[i] = it.Grow(-1)
			return after, drops
		}
	}
	for _, slot := range st.Def.Output {
		if it, _ := st.Inv().Item(slot); !it.Empty() && it.Comparable(seed) {
			_ = st.Inv().SetItem(slot, it.Grow(-1))
			return after, drops
		}
	}
	return block.Air{}, drops
}
