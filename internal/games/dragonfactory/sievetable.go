package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// SieveDrop is a possible product of sifting a block.
type SieveDrop struct {
	Item item.Stack
	// Permille is the chance per sifted block in per mille (1 = 0.1 %);
	// every drop of the block rolls on its own, so one block can give
	// several.
	Permille int
}

// Ore chunks are the pieces of ore a sieve finds. Four of one kind make an
// ore block, which a furnace smelts (the recipes open with the researches of
// the metal).
const (
	OreIron   = "iron"
	OreGold   = "gold"
	OreCopper = "copper"
)

// The drop tables follow the Ex Nihilo of SkyFactory 3 (Ex Nihilo Creatio,
// recipes/defaults/ExNihilo.kt): common things (seeds, flint, bone meal)
// come in tens of percent, ores in single percents that grow with the mesh,
// and the gems in fractions of a percent. Every mesh of the game has its own
// column; a row lists the chance per mille of its drop for each mesh from
// string to netherite, 0 for none. Chances are drafts.
//
// Each mesh finds the common things like the lowest mesh does and the ores
// from the mesh the row says on, so a better mesh is never worse. Our own
// additions to the Ex Nihilo lists (kelp, amethyst, quartz, the metals of
// metals.go) are marked in the rows.

// sieveRow is one drop of one sifted block with its chance per mesh tier
// (meshTiers order: string, flint, iron, gold, diamond, netherite).
type sieveRow struct {
	in   world.Item
	drop item.Stack
	per  [6]int
}

// same is a chance the same on every mesh.
func same(n int) [6]int { return [6]int{n, n, n, n, n, n} }

// from is a chance that starts at a mesh tier (index into meshTiers) with
// the chances of the meshes from there on.
func from(tier int, chances ...int) [6]int {
	var out [6]int
	copy(out[tier:], chances)
	return out
}

func stack(it world.Item) item.Stack { return item.NewStack(it, 1) }

var sieveRows = []sieveRow{
	// Dirt: seeds on every mesh (Ex Nihilo: wheat 70 %, melon and pumpkin
	// 35 %, other seeds 5 %).
	{block.Dirt{}, stack(block.WheatSeeds{}), same(700)},
	{block.Dirt{}, stack(block.MelonSeeds{}), same(350)},
	{block.Dirt{}, stack(block.PumpkinSeeds{}), same(350)},
	{block.Dirt{}, stack(block.BeetrootSeeds{}), same(350)},
	{block.Dirt{}, stack(block.Carrot{}), same(50)},
	{block.Dirt{}, stack(block.Potato{}), same(50)},
	{block.Dirt{}, stack(block.Sapling{Type: block.OakSapling()}), same(50)},

	// Gravel: flint 25 %, coal and lapis from the flint mesh (Ex Nihilo:
	// coal 12.5 %, lapis 5 %), diamond and emerald from the iron mesh
	// (0.8 %, double with the diamond mesh).
	{block.Gravel{}, stack(item.Flint{}), same(250)},
	{block.Gravel{}, stack(item.Coal{}), [6]int{60, 125, 125, 125, 125, 125}},
	{block.Gravel{}, stack(item.LapisLazuli{}), from(1, 50, 70, 90, 110, 130)},
	{block.Gravel{}, stack(item.Diamond{}), from(2, 8, 12, 16, 24)},
	{block.Gravel{}, stack(item.Emerald{}), from(2, 8, 12, 16, 24)},
	{block.Gravel{}, stack(item.NetheriteScrap{}), from(4, 10, 25)},
	// Ex Nihilo ores in gravel: iron 10/15/25 %, gold 5/7.5/15 %, copper
	// 5/7.5 % on the flint, iron and diamond meshes. The string mesh finds
	// iron too so the first ore comes without a better mesh; the metals
	// first found by a mesh are what the researches ask for.
	{block.Gravel{}, stack(OreChunk{Ore: OreIron}), [6]int{100, 100, 150, 200, 250, 300}},
	{block.Gravel{}, stack(OreChunk{Ore: OreGold}), from(2, 75, 110, 150, 200)},
	{block.Gravel{}, stack(OreChunk{Ore: OreCopper}), from(1, 80, 100, 120, 150, 180)},
	{block.Gravel{}, stack(OreChunk{Ore: MetalLead}), from(2, 60, 80, 100, 120)},
	{block.Gravel{}, stack(OreChunk{Ore: MetalNickel}), from(3, 50, 80, 100)},

	// Sand: kelp is ours. Ex Nihilo has tin (5 / 7.5 %) and silver (15 %
	// with the diamond mesh) in sand.
	{block.Sand{}, stack(block.Kelp{}), same(200)},
	{block.Sand{}, stack(item.AmethystShard{}), from(2, 20, 40, 60, 80)},
	{block.Sand{}, stack(item.NetherQuartz{}), from(3, 40, 60, 80)},
	{block.Sand{}, stack(OreChunk{Ore: MetalTin}), from(1, 50, 75, 100, 150, 200)},
	{block.Sand{}, stack(OreChunk{Ore: MetalLead}), from(2, 40, 60, 80, 100)},
	{block.Sand{}, stack(OreChunk{Ore: MetalSilver}), from(3, 100, 150, 200)},
	{block.Sand{}, stack(OreChunk{Ore: MetalNickel}), from(4, 80, 120)},

	// Dust: bone meal 20 % and gunpowder 7 % like Ex Nihilo; redstone 12.5 %
	// with the iron mesh, 25 % with the diamond mesh (ours: from the string
	// mesh on, the first machines need it).
	{Dust{}, stack(item.BoneMeal{}), same(200)},
	{Dust{}, stack(item.Gunpowder{}), same(70)},
	{Dust{}, stack(item.RedstoneWire{}), [6]int{100, 120, 125, 180, 250, 300}},
}

// meshIndex is the index of a mesh tier in meshTiers.
func meshIndex(tier string) int {
	for i, t := range meshTiers {
		if t.tier == tier {
			return i
		}
	}
	return -1
}

// essenceDrop makes a sieve drop of an essence; chance is a percent.
func essenceDrop(mob string, chance int) SieveDrop {
	return SieveDrop{item.NewStack(Essence{Mob: mob}, 1), chance * 10}
}

// sieveTables are the drops of every kind of block sifted, by mesh tier.
var sieveTables = buildSieveTables()

func buildSieveTables() map[string]map[string][]SieveDrop {
	out := map[string]map[string][]SieveDrop{}
	for i, t := range meshTiers {
		table := map[string][]SieveDrop{}
		for _, r := range sieveRows {
			if r.per[i] > 0 {
				key := itemKey(r.in)
				table[key] = append(table[key], SieveDrop{r.drop, r.per[i]})
			}
		}
		for key, drops := range essenceExtra[t.tier] {
			table[key] = append(table[key], drops...)
		}
		out[t.tier] = table
	}
	return out
}
