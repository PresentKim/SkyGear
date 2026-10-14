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

// stringTable is the drop table of the string mesh. The numbers follow the
// spirit of Ex Nihilo (generous, since the sieve is the only source of
// nearly everything in the sky) and are drafts.
var stringTable = map[string][]SieveDrop{
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
}

// meshBoost is how much a better mesh multiplies the chances of the string
// table (percent, up to 95), and meshExtra what it finds on top.
var meshBoost = map[string]int{"flint": 130, "iron": 160, "gold": 200, "diamond": 250, "netherite": 300}

var meshExtra = map[string]map[string][]SieveDrop{
	"flint": {
		itemKey(block.Gravel{}): {{item.NewStack(item.LapisLazuli{}, 1), 10}},
	},
	"iron": {
		itemKey(block.Gravel{}): {{item.NewStack(item.LapisLazuli{}, 1), 15}, {item.NewStack(item.Diamond{}, 1), 1}},
		itemKey(block.Sand{}):   {{item.NewStack(item.AmethystShard{}, 1), 3}},
	},
	"gold": {
		itemKey(block.Gravel{}): {{item.NewStack(item.LapisLazuli{}, 1), 20}, {item.NewStack(item.Diamond{}, 1), 3}, {item.NewStack(item.Emerald{}, 1), 2}},
		itemKey(block.Sand{}):   {{item.NewStack(item.AmethystShard{}, 1), 6}, {item.NewStack(item.NetherQuartz{}, 1), 6}},
	},
	"diamond": {
		itemKey(block.Gravel{}): {{item.NewStack(item.LapisLazuli{}, 1), 25}, {item.NewStack(item.Diamond{}, 1), 6}, {item.NewStack(item.Emerald{}, 1), 4}, {item.NewStack(item.NetheriteScrap{}, 1), 1}},
		itemKey(block.Sand{}):   {{item.NewStack(item.AmethystShard{}, 1), 10}, {item.NewStack(item.NetherQuartz{}, 1), 10}},
	},
	"netherite": {
		itemKey(block.Gravel{}): {{item.NewStack(item.LapisLazuli{}, 1), 30}, {item.NewStack(item.Diamond{}, 1), 10}, {item.NewStack(item.Emerald{}, 1), 8}, {item.NewStack(item.NetheriteScrap{}, 1), 3}},
		itemKey(block.Sand{}):   {{item.NewStack(item.AmethystShard{}, 1), 15}, {item.NewStack(item.NetherQuartz{}, 1), 15}},
	},
}

// sieveTables are the drops of every kind of block sifted, by mesh tier: the
// string table, and for the better meshes its chances multiplied plus the
// extras.
var sieveTables = buildSieveTables()

func buildSieveTables() map[string]map[string][]SieveDrop {
	out := map[string]map[string][]SieveDrop{"string": stringTable}
	for tier, boost := range meshBoost {
		t := map[string][]SieveDrop{}
		for key, drops := range stringTable {
			for _, d := range drops {
				d.Chance = min(95, d.Chance*boost/100)
				t[key] = append(t[key], d)
			}
		}
		for key, drops := range meshExtra[tier] {
			t[key] = append(t[key], drops...)
		}
		out[tier] = t
	}
	return out
}
