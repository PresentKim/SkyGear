// Package oneblock defines the OneBlock phases: which blocks the OneBlock
// turns into and which bonus items it may drop as it is broken.
package oneblock

import (
	"github.com/presentkim/dragonfactory/internal/i18n"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// BlockEntry is a weighted block the OneBlock may become.
type BlockEntry struct {
	Block  world.Block
	Weight int
}

// Bonus is an item that may be given when the OneBlock is broken.
type Bonus struct {
	Item   item.Stack
	Chance float64 // 0..1
}

// Phase is a stage of the OneBlock progression.
type Phase struct {
	Name string
	// Start is the amount of broken blocks required to enter the phase.
	Start  int
	Blocks []BlockEntry
	Bonus  []Bonus
	// Reward is given once when the phase is first reached.
	Reward []item.Stack
}

func (p Phase) pick() world.Block {
	total := 0
	for _, e := range p.Blocks {
		total += e.Weight
	}
	n := rand.IntN(total)
	for _, e := range p.Blocks {
		if n < e.Weight {
			return e.Block
		}
		n -= e.Weight
	}
	return p.Blocks[0].Block
}

// Rolls returns the bonus items dropped for one break.
func (p Phase) Rolls() []item.Stack {
	var out []item.Stack
	for _, b := range p.Bonus {
		if rand.Float64() < b.Chance {
			out = append(out, b.Item)
		}
	}
	return out
}

// PhaseAt returns the index and phase for a broken count. After the last
// phase the final phase repeats forever.
func PhaseAt(broken int) (int, Phase) {
	idx := 0
	for i, p := range Phases {
		if broken >= p.Start {
			idx = i
		}
	}
	return idx, Phases[idx]
}

// Next returns the block the OneBlock becomes after `broken` breaks.
func Next(broken int) world.Block {
	if broken == 0 {
		return block.Grass{}
	}
	_, p := PhaseAt(broken)
	return p.pick()
}

func s(it world.Item, n int) item.Stack { return item.NewStack(it, n) }

var (
	waterBucket = item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}
	oakLeaves   = block.Leaves{Type: block.OakLeaves(), Persistent: true}
)

// Phases is the farming focused OneBlock progression. Gravity affected and
// non-solid blocks are deliberately avoided so the OneBlock never falls into
// the void or pops off.
var Phases = []Phase{
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.1"),
		Start: 0,
		Blocks: []BlockEntry{
			{block.Grass{}, 40}, {block.Dirt{}, 25},
			{block.Log{Wood: block.OakWood()}, 25}, {oakLeaves, 10},
		},
		Bonus: []Bonus{
			{s(block.WheatSeeds{}, 2), 0.25},
			{s(block.Sapling{Type: block.OakSapling()}, 1), 0.08},
			{s(item.Apple{}, 1), 0.05},
			{s(item.BoneMeal{}, 2), 0.05},
		},
		Reward: []item.Stack{s(block.WheatSeeds{}, 4), s(block.Sapling{Type: block.OakSapling()}, 1)},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.2"),
		Start: 100,
		Blocks: []BlockEntry{
			{block.Grass{}, 25}, {block.Dirt{}, 20}, {block.Log{Wood: block.OakWood()}, 15},
			{block.HayBale{}, 10}, {block.Pumpkin{}, 10}, {block.Melon{}, 10}, {block.Clay{}, 5},
		},
		Bonus: []Bonus{
			{s(waterBucket, 1), 0.03},
			{s(block.Carrot{}, 1), 0.10},
			{s(block.Potato{}, 1), 0.10},
			{s(block.BeetrootSeeds{}, 2), 0.10},
			{s(block.PumpkinSeeds{}, 1), 0.06},
			{s(block.MelonSeeds{}, 1), 0.06},
			{s(block.SugarCane{}, 1), 0.06},
			{s(item.BoneMeal{}, 2), 0.08},
		},
		Reward: []item.Stack{s(waterBucket, 1), s(item.Hoe{Tier: item.ToolTierWood}, 1)},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.3"),
		Start: 300,
		Blocks: []BlockEntry{
			{block.Log{Wood: block.OakWood()}, 20}, {block.Log{Wood: block.BirchWood()}, 15},
			{block.Log{Wood: block.SpruceWood()}, 15}, {block.Log{Wood: block.JungleWood()}, 15},
			{oakLeaves, 10}, {block.Grass{}, 15}, {block.Podzol{}, 10},
		},
		Bonus: []Bonus{
			{s(block.Sapling{Type: block.BirchSapling()}, 1), 0.06},
			{s(block.Sapling{Type: block.SpruceSapling()}, 1), 0.06},
			{s(block.Sapling{Type: block.JungleSapling()}, 1), 0.06},
			{s(block.CocoaBean{}, 1), 0.08},
			{s(item.Apple{}, 1), 0.08},
		},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.4"),
		Start: 650,
		Blocks: []BlockEntry{
			{block.Stone{}, 40}, {block.Cobblestone{}, 20}, {block.CoalOre{}, 15},
			{block.IronOre{}, 10}, {block.CopperOre{}, 10}, {block.GoldOre{}, 5},
		},
		Bonus: []Bonus{
			{s(item.BoneMeal{}, 3), 0.08},
			{s(item.RawIron{}, 1), 0.04},
		},
		Reward: []item.Stack{s(item.Pickaxe{Tier: item.ToolTierStone}, 1)},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.5"),
		Start: 1100,
		Blocks: []BlockEntry{
			{block.Mud{}, 25}, {block.Clay{}, 20}, {block.Grass{}, 20}, {block.Dirt{}, 15},
			{block.Log{Wood: block.MangroveWood()}, 20},
		},
		Bonus: []Bonus{
			{s(waterBucket, 1), 0.04},
			{s(block.SugarCane{}, 2), 0.10},
			{s(block.LilyPad{}, 1), 0.06},
			{s(block.Pumpkin{}, 1), 0.04},
		},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.6"),
		Start: 1700,
		Blocks: []BlockEntry{
			{block.Netherrack{}, 35}, {block.SoulSand{}, 20}, {block.NetherWartBlock{}, 15},
			{block.NetherQuartzOre{}, 10}, {block.Glowstone{}, 10}, {block.Shroomlight{}, 10},
		},
		Bonus: []Bonus{
			{s(block.NetherWart{}, 1), 0.12},
			{s(item.BlazePowder{}, 1), 0.02},
		},
		Reward: []item.Stack{s(block.NetherWart{}, 4)},
	},
	{
		Name:  i18n.T(i18n.Default, "oneblock.phase.init.7"),
		Start: 2500,
		Blocks: []BlockEntry{
			{block.HayBale{}, 20}, {block.Pumpkin{}, 15}, {block.Melon{}, 15}, {block.Grass{}, 15},
			{block.GoldOre{}, 10}, {block.EmeraldOre{}, 10}, {block.DiamondOre{}, 5}, {block.IronOre{}, 10},
		},
		Bonus: []Bonus{
			{s(item.GoldenCarrot{}, 1), 0.04},
			{s(item.GlisteringMelonSlice{}, 1), 0.04},
			{s(item.BoneMeal{}, 4), 0.10},
		},
	},
}
