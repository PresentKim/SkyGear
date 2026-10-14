package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
)

// The auto harvester (second era, draft) harvests the grown crops around it
// on its own level, harvestRadius blocks each way: wheat, carrots, potatoes
// and beetroots are picked and left as young plants, and melons and pumpkins
// grown from stems are cut; blooming magic plants are picked and go back to
// buds. The products go to its output slots, from where
// it pushes them into neighbouring containers like the auto miner.

// harvestRadius is how far the harvester reaches in x and z.
const harvestRadius = 4

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
func (s *session) runHarvester(st *machineState, pos cube.Pos, tx *world.Tx) {
	if st.progress < st.kind.interval {
		if st.consume() {
			st.progress++
		}
		return
	}
	st.progress = 0
	for x := -harvestRadius; x <= harvestRadius; x++ {
		for z := -harvestRadius; z <= harvestRadius; z++ {
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
			if !fits(st.inv(), st.kind.output, drops) {
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
					put(st.inv(), st.kind.output, it)
				}
			}
			return
		}
	}
}

// harvestMagicPlant picks a blooming magic plant: the yield goes to the
// harvester's output and the plant goes back to a bud. A plant whose yield
// does not fit is left for the next cycle.
func (s *session) harvestMagicPlant(st *machineState, tx *world.Tx, at cube.Pos, mp MagicPlant) bool {
	drops, ok := s.plantDrops(mp)
	if !ok || !fits(st.inv(), st.kind.output, drops) {
		return false
	}
	for _, it := range drops {
		if !it.Empty() {
			put(st.inv(), st.kind.output, it)
		}
	}
	s.resetPlant(tx, at, mp)
	return true
}

// replantSeed takes the seed a harvested crop is replanted with: one from
// its drops, else one from the harvester's output. Without a seed the crop
// is not replanted (after becomes air). Stems (melon, pumpkin) need none.
func replantSeed(st *machineState, after world.Block, drops []item.Stack) (world.Block, []item.Stack) {
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
	for _, slot := range st.kind.output {
		if it, _ := st.inv().Item(slot); !it.Empty() && it.Comparable(seed) {
			_ = st.inv().SetItem(slot, it.Grow(-1))
			return after, drops
		}
	}
	return block.Air{}, drops
}
