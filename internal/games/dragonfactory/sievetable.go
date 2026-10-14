package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
)

// SieveDrop is a possible product of sifting a block.
type SieveDrop struct {
	Item item.Stack
	// Chance is the percent chance per sifted block; every drop of the
	// block rolls on its own, so one block can give several.
	Chance int
}

// Ore chunks are the pieces of ore a sieve finds. Four of one kind make an
// ore block, which a furnace smelts (the recipes open with the researches of
// the metal). Better meshes will find more kinds and more of them.
const (
	OreIron   = "iron"
	OreGold   = "gold"
	OreCopper = "copper"
)

// sieveTables are the drops of every kind of block sifted, by mesh tier. The
// numbers follow the spirit of Ex Nihilo (generous, since the sieve is the
// only source of nearly everything in the sky) and are drafts.
var sieveTables = map[string]map[string][]SieveDrop{
	"string": {
		itemKey(block.Gravel{}): {
			{item.NewStack(item.Flint{}, 1), 40},
			{item.NewStack(item.Coal{}, 1), 20},
			{item.NewStack(OreChunk{Ore: OreIron}, 1), 30},
			{item.NewStack(OreChunk{Ore: OreCopper}, 1), 20},
			{item.NewStack(OreChunk{Ore: OreGold}, 1), 10},
		},
		itemKey(block.Sand{}): {
			{item.NewStack(item.ClayBall{}, 1), 35},
			{item.NewStack(block.Kelp{}, 1), 12},
			{item.NewStack(OreChunk{Ore: OreGold}, 1), 15},
			{item.NewStack(OreChunk{Ore: OreCopper}, 1), 10},
		},
		itemKey(block.Dirt{}): {
			{item.NewStack(block.WheatSeeds{}, 1), 50},
			{item.NewStack(block.Sapling{Type: block.OakSapling()}, 1), 20},
			{item.NewStack(block.BeetrootSeeds{}, 1), 12},
			{item.NewStack(block.PumpkinSeeds{}, 1), 12},
			{item.NewStack(block.MelonSeeds{}, 1), 12},
			{item.NewStack(block.Carrot{}, 1), 8},
			{item.NewStack(block.Potato{}, 1), 8},
		},
		itemKey(Dust{}): {
			{item.NewStack(item.BoneMeal{}, 1), 30},
			{item.NewStack(item.RedstoneWire{}, 1), 20},
			{item.NewStack(item.Gunpowder{}, 1), 12},
		},
	},
}
