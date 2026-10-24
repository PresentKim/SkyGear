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

// The drop tables follow ATM10: To the Sky (docs/design/16-ore-balance.md):
// Ex Deorum's tables (the 1.21.1 port of Ex Nihilo) with the pack's
// changes, for the items Bedrock has. Every mesh of the game has its own
// column; a row lists the chance per mille of its drop for each mesh from
// string to netherite, 0 for none. Rows the pack does not have are marked
// as ours.

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

// twice is a drop rolled twice (Ex Deorum's binomial with n = 2): each of the
// stack's items is its own roll.
func twice(it world.Item) item.Stack { return item.NewStack(it, 2) }

// rolls is a drop rolled n times (Ex Deorum's binomial): each of the stack's
// items is its own roll.
func rolls(it world.Item, n int) item.Stack { return item.NewStack(it, n) }

// The crushed blocks a sieve takes.
var (
	deepslateC  = CrushedBlock{Kind: CrushedDeepslate}
	netherrackC = CrushedBlock{Kind: CrushedNetherrack}
	blackstoneC = CrushedBlock{Kind: CrushedBlackstone}
	endStoneC   = CrushedBlock{Kind: CrushedEndStone}
)

var sieveRows = []sieveRow{
	// Dirt: ATM10: To the Sky (Ex Deorum; the pack's additions are flowers
	// and mod seeds Bedrock or the server lacks, as are sweet berries). The
	// golden apple's 2.5 % on the netherite mesh is 3 per mille.
	{block.Dirt{}, rolls(Pebble{Kind: PebbleStone}, 7), [6]int{600, 600, 0, 0, 0, 0}},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleStone}, 8), [6]int{0, 0, 650, 700, 700, 0}},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleStone}, 5), from(5, 400)},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleAndesite}, 7), from(1, 400, 0, 0, 0, 0)},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleGranite}, 7), from(1, 400, 0, 0, 0, 0)},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleDiorite}, 7), from(1, 400, 0, 0, 0, 0)},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleDeepslate}, 3), [6]int{0, 0, 450, 550, 600, 0}},
	{block.Dirt{}, rolls(Pebble{Kind: PebbleDeepslate}, 4), from(5, 650)},
	{block.Dirt{}, stack(item.Flint{}), [6]int{250, 300, 300, 200, 0, 0}},
	{block.Dirt{}, rolls(item.Flint{}, 3), from(4, 300, 400)},
	{block.Dirt{}, stack(block.WheatSeeds{}), [6]int{125, 150, 175, 200, 0, 0}},
	{block.Dirt{}, stack(block.MelonSeeds{}), [6]int{100, 120, 150, 165, 0, 0}},
	{block.Dirt{}, stack(block.PumpkinSeeds{}), [6]int{100, 120, 150, 165, 0, 0}},
	{block.Dirt{}, stack(block.BeetrootSeeds{}), [6]int{100, 0, 0, 0, 0, 0}},
	{block.Dirt{}, stack(block.Potato{}), [6]int{100, 130, 150, 175, 250, 300}},
	{block.Dirt{}, stack(block.Carrot{}), [6]int{100, 130, 150, 175, 250, 300}},
	{block.Dirt{}, stack(GrassSeeds{}), [6]int{100, 150, 175, 250, 150, 200}},
	{block.Dirt{}, stack(MyceliumSpores{}), [6]int{30, 50, 100, 130, 100, 200}},
	{block.Dirt{}, stack(block.SugarCane{}), [6]int{100, 150, 150, 0, 0, 0}},
	{block.Dirt{}, stack(item.PoisonousPotato{}), [6]int{50, 30, 0, 0, 0, 0}},
	{block.Dirt{}, stack(block.Bamboo{}), [6]int{40, 40, 60, 50, 60, 60}},
	{block.Dirt{}, stack(block.PinkPetals{}), from(1, 30, 0, 0, 0, 0)},
	{block.Dirt{}, stack(item.IronNugget{}), from(2, 50, 50)},
	{block.Dirt{}, stack(item.GoldenCarrot{}), [6]int{0, 0, 0, 20, 0, 10}},
	{block.Dirt{}, stack(item.GoldenApple{}), from(5, 3)},

	// Gravel: ATM10: To the Sky (Ex Deorum 1.21.1 SieveRecipes.java with the
	// pack's KubeJS: gold nuggets and gravel pebbles removed, diamond and
	// emerald from the iron mesh on, twice on the netherite mesh) —
	// docs/design/16-ore-balance.md. Every ore is a chunk from the string
	// mesh on; research decides when a metal is of use.
	{block.Gravel{}, stack(item.Flint{}), [6]int{200, 250, 150, 130, 50, 0}},
	{block.Gravel{}, stack(item.Coal{}), [6]int{100, 125, 150, 200, 60, 60}},
	{block.Gravel{}, stack(item.LapisLazuli{}), [6]int{30, 50, 80, 100, 110, 110}},
	{block.Gravel{}, stack(item.AmethystShard{}), [6]int{10, 15, 40, 80, 60, 80}},
	{block.Gravel{}, stack(item.Diamond{}), [6]int{0, 0, 10, 40, 60, 0}},
	{block.Gravel{}, twice(item.Diamond{}), from(5, 60)},
	{block.Gravel{}, stack(item.Emerald{}), [6]int{0, 0, 10, 30, 40, 0}},
	{block.Gravel{}, twice(item.Emerald{}), from(5, 50)},
	{block.Gravel{}, stack(item.RawGold{}), from(3, 20)},
	{block.Gravel{}, stack(OreChunk{Ore: OreIron}), [6]int{100, 120, 140, 140, 150, 170}},
	{block.Gravel{}, stack(OreChunk{Ore: OreCopper}), [6]int{80, 100, 120, 70, 70, 100}},
	{block.Gravel{}, stack(OreChunk{Ore: OreGold}), [6]int{30, 40, 60, 100, 80, 90}},
	{block.Gravel{}, stack(OreChunk{Ore: MetalTin}), [6]int{60, 70, 90, 70, 110, 120}},
	{block.Gravel{}, stack(OreChunk{Ore: MetalLead}), [6]int{40, 50, 60, 70, 90, 110}},
	{block.Gravel{}, stack(OreChunk{Ore: MetalSilver}), [6]int{40, 50, 55, 120, 80, 110}},
	{block.Gravel{}, stack(OreChunk{Ore: MetalNickel}), [6]int{40, 55, 70, 70, 90, 100}},

	// Sand: ATM10: To the Sky (Ex Deorum without the gold nuggets, the
	// pack's cactus and kelp on the diamond mesh; pottery sherds and armour
	// trims left out).
	{block.Sand{}, stack(block.Cactus{}), [6]int{130, 0, 130, 100, 120, 150}},
	{block.Sand{}, stack(item.Flint{}), [6]int{200, 0, 230, 180, 230, 0}},
	{block.Sand{}, rolls(item.Flint{}, 2), [6]int{0, 200, 0, 0, 0, 230}},
	{block.Sand{}, stack(block.DeadBush{}), [6]int{80, 30, 80, 60, 0, 0}},
	{block.Sand{}, stack(item.IronNugget{}), [6]int{130, 160, 180, 160, 220, 230}},
	{block.Sand{}, stack(block.Kelp{}), [6]int{100, 0, 70, 50, 75, 100}},
	{block.Sand{}, stack(block.SeaPickle{}), [6]int{50, 0, 30, 30, 0, 70}},
	{block.Sand{}, stack(item.PrismarineShard{}), from(2, 60, 80, 90, 120)},
	{block.Sand{}, stack(item.PrismarineCrystals{}), from(2, 60, 80, 90, 120)},
	{block.Sand{}, stack(item.RawGold{}), from(3, 40)},

	// Crushed deepslate (the pack takes its deepslate pebbles out).
	{deepslateC, stack(OreChunk{Ore: OreIron}), [6]int{120, 130, 150, 150, 180, 200}},
	{deepslateC, stack(OreChunk{Ore: OreCopper}), [6]int{120, 110, 100, 90, 90, 100}},
	{deepslateC, stack(OreChunk{Ore: OreGold}), [6]int{80, 80, 90, 150, 130, 150}},
	{deepslateC, stack(OreChunk{Ore: MetalTin}), [6]int{50, 70, 110, 120, 130, 160}},
	{deepslateC, stack(OreChunk{Ore: MetalLead}), [6]int{50, 80, 100, 90, 110, 120}},
	{deepslateC, stack(OreChunk{Ore: MetalSilver}), [6]int{50, 80, 100, 150, 110, 120}},
	{deepslateC, stack(OreChunk{Ore: MetalNickel}), [6]int{40, 60, 90, 110, 110, 150}},
	{deepslateC, stack(item.AmethystShard{}), [6]int{50, 60, 60, 80, 70, 100}},
	{deepslateC, stack(item.Diamond{}), [6]int{40, 50, 60, 80, 80, 100}},
	{deepslateC, stack(item.LapisLazuli{}), [6]int{40, 50, 80, 70, 120, 140}},
	{deepslateC, stack(item.Emerald{}), [6]int{30, 40, 50, 70, 80, 100}},
	{deepslateC, stack(item.RawGold{}), from(3, 50)},
	{deepslateC, rolls(Pebble{Kind: PebbleTuff}, 4), [6]int{0, 400, 0, 0, 0, 0}},
	{deepslateC, rolls(Pebble{Kind: PebbleCalcite}, 4), [6]int{0, 400, 0, 0, 0, 0}},
	{deepslateC, rolls(Pebble{Kind: PebbleBasalt}, 4), [6]int{0, 400, 0, 0, 0, 0}},

	// Crushed blackstone.
	{blackstoneC, rolls(Pebble{Kind: PebbleBlackstone}, 4), [6]int{600, 650, 0, 0, 0, 0}},
	{blackstoneC, rolls(Pebble{Kind: PebbleBlackstone}, 5), from(2, 650, 700, 700, 750)},
	{blackstoneC, rolls(Pebble{Kind: PebbleBasalt}, 3), [6]int{500, 550, 0, 0, 0, 0}},
	{blackstoneC, rolls(Pebble{Kind: PebbleBasalt}, 4), [6]int{0, 0, 550, 500, 0, 0}},
	{blackstoneC, stack(block.AncientDebris{}), [6]int{20, 30, 40, 50, 60, 100}},
	{blackstoneC, stack(item.MagmaCream{}), [6]int{80, 90, 90, 100, 110, 120}},
	{blackstoneC, stack(item.Gunpowder{}), [6]int{70, 90, 90, 100, 110, 110}},
	{blackstoneC, stack(item.Dye{Colour: item.ColourBlack()}), [6]int{70, 80, 80, 60, 0, 0}},

	// Crushed netherrack (nylium spores left out: Bedrock has none).
	{netherrackC, rolls(Pebble{Kind: PebbleBlackstone}, 3), [6]int{400, 0, 0, 0, 0, 0}},
	{netherrackC, rolls(Pebble{Kind: PebbleBlackstone}, 4), [6]int{0, 500, 600, 600, 600, 0}},
	{netherrackC, rolls(Pebble{Kind: PebbleBlackstone}, 5), from(5, 650)},
	{netherrackC, rolls(Pebble{Kind: PebbleBasalt}, 3), [6]int{300, 0, 0, 0, 0, 0}},
	{netherrackC, rolls(Pebble{Kind: PebbleBasalt}, 4), [6]int{0, 400, 450, 450, 0, 0}},
	{netherrackC, stack(item.BlazePowder{}), [6]int{80, 90, 100, 110, 140, 150}},
	{netherrackC, stack(item.NetherQuartz{}), [6]int{80, 90, 110, 130, 130, 150}},
	{netherrackC, stack(item.MagmaCream{}), [6]int{50, 60, 70, 80, 100, 100}},
	{netherrackC, stack(item.Gunpowder{}), [6]int{80, 90, 100, 110, 130, 130}},
	{netherrackC, stack(item.RawGold{}), from(3, 30)},

	// Crushed end stone (chorus and sculk left out: the server has none).
	{endStoneC, stack(item.EnderPearl{}), [6]int{70, 80, 100, 120, 150, 170}},
	{endStoneC, stack(item.EnderEye{}), [6]int{20, 30, 40, 70, 90, 90}},
	{endStoneC, stack(item.EchoShard{}), from(5, 30)},

	// Soul sand (nylium spores left out).
	{block.SoulSand{}, stack(item.NetherQuartz{}), [6]int{120, 140, 150, 170, 190, 210}},
	{block.SoulSand{}, stack(item.Gunpowder{}), [6]int{70, 80, 70, 100, 110, 140}},
	{block.SoulSand{}, stack(item.Bone{}), [6]int{80, 100, 80, 110, 0, 0}},
	{block.SoulSand{}, stack(item.GhastTear{}), [6]int{60, 70, 60, 80, 90, 110}},
	{block.SoulSand{}, stack(block.NetherWart{}), [6]int{60, 60, 50, 80, 100, 120}},
	{block.SoulSand{}, stack(item.GlowstoneDust{}), [6]int{60, 70, 60, 90, 110, 130}},

	// Red sand.
	{block.Sand{Red: true}, stack(block.Cactus{}), [6]int{120, 120, 120, 120, 100, 120}},
	{block.Sand{Red: true}, stack(block.DeadBush{}), [6]int{70, 70, 70, 70, 30, 0}},
	{block.Sand{Red: true}, stack(item.RedstoneWire{}), [6]int{80, 90, 110, 70, 140, 170}},
	{block.Sand{Red: true}, stack(item.RawGold{}), [6]int{30, 40, 60, 110, 80, 100}},

	// Dust: ATM10: To the Sky (Ex Deorum without the gold nuggets; the
	// pack's additions are mod items). Monster essences are added on top
	// (essenceExtra).
	{Dust{}, stack(item.Gunpowder{}), [6]int{100, 110, 130, 130, 140, 140}},
	{Dust{}, stack(item.BoneMeal{}), [6]int{100, 110, 120, 110, 100, 130}},
	{Dust{}, stack(item.RedstoneWire{}), [6]int{60, 90, 100, 120, 120, 140}},
	{Dust{}, stack(item.GlowstoneDust{}), [6]int{40, 70, 90, 110, 110, 150}},
	{Dust{}, stack(item.BlazePowder{}), [6]int{30, 40, 50, 60, 60, 100}},
	{Dust{}, stack(item.IronNugget{}), [6]int{0, 0, 60, 0, 0, 80}},
	{Dust{}, stack(item.RawGold{}), from(3, 20)},
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
