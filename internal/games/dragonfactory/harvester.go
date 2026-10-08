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
// grown from stems are cut. The products go to its output slots, from where
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
func (s *session) runHarvester(st *machineState, pos cube.Pos, tx *world.Tx) {
	if st.progress < st.kind.interval {
		st.progress++
		return
	}
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
			tx.SetBlock(at, after, nil)
			if kind, _, ok := cropOf(after); ok {
				s.plantCrop(at, kind)
			}
			tx.AddParticle(at.Vec3Centre(), particle.BlockBreak{Block: b})
			for _, it := range drops {
				put(st.inv(), st.kind.output, it)
			}
			st.progress = 0
			return
		}
	}
}
