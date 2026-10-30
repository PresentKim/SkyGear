package catalog

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// gameRecipes registers the crafting recipes of the game's own items, open
// once researched. Drafts.
func gameRecipes() {
	st := func(it world.Item) item.Stack { return item.NewStack(it, 1) }
	n := func(it world.Item, count int) []recipe.Item {
		out := make([]recipe.Item, count)
		for i := range out {
			out[i] = st(it)
		}
		return out
	}
	add := func(out item.Stack, in ...[]recipe.Item) {
		recipe.Register(recipe.NewShapeless(slices.Concat(in...), out, "crafting_table"))
	}
	// shaped adds a recipe that crafts only as arranged (cells row by row,
	// nil is an empty cell), not mirrored either.
	shaped := func(out item.Stack, w, h int, cells ...world.Item) {
		in := make([]recipe.Item, len(cells))
		for i, c := range cells {
			in[i] = item.Stack{}
			if c != nil {
				in[i] = st(c)
			}
		}
		recipe.Register(recipe.NewShaped(in, out, recipe.NewShape(w, h), "crafting_table").WithoutMirror())
	}
	cobble, planks, iron := block.Cobblestone{}, OakPlanks, item.IronIngot{}
	// The hammers: two of the material and two sticks in a diagonal
	// pattern, the stone hammer of cobblestone, the iron hammer of iron.
	shaped(st(sgitem.Hammer{Stone: true}), 3, 3,
		nil, cobble, nil,
		nil, Stick, cobble,
		Stick, nil, nil)
	shaped(st(sgitem.Hammer{}), 3, 3,
		nil, iron, nil,
		nil, Stick, iron,
		Stick, nil, nil)
	// The wooden bucket: three planks in a V, like the iron bucket.
	shaped(st(sgitem.WoodenBucket{}), 3, 2,
		planks, nil, planks,
		nil, planks, nil)
	// The sieve: a row of planks on two stick legs. The mesh is made and
	// fitted separately (sieve.go).
	shaped(machine.Item(machine.KindSieve), 3, 2,
		planks, planks, planks,
		Stick, nil, Stick)
	// The combustion generator and the first energy conduit.
	add(machine.Item(machine.KindGenerator), n(iron, 6), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 2))
	// The conduits: sand and gravel in a checkerboard make the binder mix, a
	// furnace bakes it into the binder; the basic conduits are binders above
	// and below a row of redstone (energy), an iron nugget, a chest and an
	// iron nugget (item) or glass (fluid).
	sand, gravel, binder := block.Sand{}, block.Gravel{}, sgitem.ConduitBinder{}
	shaped(item.NewStack(sgitem.ConduitBinderMix{}, 6), 3, 3,
		sand, gravel, sand,
		gravel, sand, gravel,
		sand, gravel, sand)
	redstone, nugget := item.RedstoneWire{}, item.IronNugget{}
	shaped(item.NewStack(conduit.EnergyConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		redstone, redstone, redstone,
		binder, binder, binder)
	shaped(item.NewStack(conduit.ItemConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		nugget, block.NewChest(), nugget,
		binder, binder, binder)
	shaped(item.NewStack(conduit.FluidConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		block.Glass{}, block.Glass{}, block.Glass{},
		binder, binder, binder)
	add(machine.Item(machine.KindCrusher), n(cobble, 6), n(iron, 3))
	// The alloy smelter and the energy cell.
	add(machine.Item(machine.KindAlloy), n(iron, 6), n(block.NewFurnace(cube.North), 1), n(block.NewHopper(), 1))
	add(machine.Item(machine.KindEnergyCell), n(sgitem.Ingot{Metal: sgitem.MetalLead}, 3), n(item.RedstoneWire{}, 4), n(iron, 2))
	add(machine.Item(machine.KindTank), n(item.IronIngot{}, 3), n(block.Glass{}, 6))
	add(machine.Item(machine.KindCrucible), n(sgitem.Ingot{Metal: sgitem.MetalBronze}, 4), n(block.NewFurnace(cube.North), 1), n(block.NewHopper(), 1))
	add(machine.Item(machine.KindLavaGenerator), n(sgitem.Ingot{Metal: sgitem.MetalSteel}, 3), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 3), n(block.Glass{}, 2))
	add(machine.Item(machine.KindWasher), n(sgitem.Ingot{Metal: sgitem.MetalBronze}, 4), n(block.Glass{}, 4), n(block.NewHopper(), 1))
	add(machine.Item(machine.KindGate), n(sgitem.Ingot{Metal: sgitem.MetalSteel}, 4), n(block.Obsidian{}, 3), n(item.EnderPearl{}, 2))
	add(machine.Item(machine.KindStorageController), n(sgitem.Ingot{Metal: sgitem.MetalBronze}, 4), n(storage.Drawer{}, 1), n(block.NewHopper(), 1), n(item.RedstoneWire{}, 2))
	add(item.NewStack(machine.Machine{Kind: machine.KindStorageLink}, 4), n(iron, 2), n(item.RedstoneWire{}, 1))
	add(machine.Item(machine.KindTinker), n(item.IronIngot{}, 6), n(sgitem.Ingot{Metal: sgitem.MetalBronze}, 2), n(block.CraftingTable{}, 1))
	add(machine.Item(machine.KindSimulator), n(sgitem.Ingot{Metal: sgitem.MetalSteel}, 3), n(block.Glass{}, 3), n(item.RedstoneWire{}, 3))
	// The dragon egg, the last project's prize (project.go).
	add(item.NewStack(block.DragonEgg{}, 1), n(item.EnderEye{}, 1), n(block.Obsidian{}, 8))
	// The automation equipment.
	add(machine.Item(machine.KindFurnace), n(iron, 5), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 2))
	add(machine.Item(machine.KindVacuum), n(iron, 4), n(block.NewHopper(), 1), n(item.RedstoneWire{}, 2))
	add(machine.Item(machine.KindFeller), n(iron, 4), n(item.Axe{Tier: item.ToolTierIron}, 1), n(block.NewHopper(), 1))
	add(machine.Item(machine.KindCrafter), n(block.CraftingTable{}, 1), n(block.NewHopper(), 2), n(iron, 3), n(item.RedstoneWire{}, 3))
	add(machine.Item(machine.KindLab), n(iron, 3), n(block.Glass{}, 3), n(item.RedstoneWire{}, 3))
	// Better conduits: a conduit of the tier below with an electrum ingot (2) or,
	// for the third tier, a steel ingot and a diamond (3).
	for _, tier := range []int{2, 3} {
		for _, kind := range []string{"energy", "item", "fluid"} {
			below := conduit.Of(kind, tier-1)
			if tier == 2 {
				add(st(conduit.Of(kind, tier)), []recipe.Item{st(below)}, n(sgitem.Ingot{Metal: sgitem.MetalElectrum}, 1))
			} else {
				add(st(conduit.Of(kind, tier)), []recipe.Item{st(below)}, n(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1), n(item.Diamond{}, 1))
			}
		}
	}
	// The sieve is a block with many states (how full it is); the recipe takes it
	// whatever its state, as the client matches recipe blocks by the meta it is
	// told (any), not by the one state an empty sieve has.
	anySieve := []recipe.Item{st(machine.Machine{Kind: machine.KindSieve}).WithValue("variants", true)}
	add(machine.Item(machine.KindAutoSieve), n(iron, 4), anySieve, n(block.NewHopper(), 1))
	// The machine tiers' circuits and kits.
	steel, electrum := sgitem.Ingot{Metal: sgitem.MetalSteel}, sgitem.Ingot{Metal: sgitem.MetalElectrum}
	infused, reinforced, atomic := sgitem.TierAlloy{Kind: "infused"}, sgitem.TierAlloy{Kind: "reinforced"}, sgitem.TierAlloy{Kind: "atomic"}
	c1, c2, c3, c4 := sgitem.Circuit{Level: 1}, sgitem.Circuit{Level: 2}, sgitem.Circuit{Level: 3}, sgitem.Circuit{Level: 4}
	shaped(st(c1), 3, 1, redstone, infused, redstone)
	shaped(st(c2), 3, 1, reinforced, c1, reinforced)
	shaped(st(c3), 3, 1, atomic, c2, atomic)
	shaped(st(c4), 3, 3,
		atomic, item.NetheriteScrap{}, atomic,
		sgitem.ResearchEssence{Kind: sgitem.EssenceHunting}, c3, sgitem.ResearchEssence{Kind: sgitem.EssenceMagic},
		atomic, nil, atomic)
	shaped(st(sgitem.TierKit{Level: 1}), 3, 3,
		steel, c2, steel,
		electrum, machine.Machine{Kind: machine.KindEnergyCell}, electrum,
		steel, c2, steel)
	shaped(st(sgitem.TierKit{Level: 2}), 3, 3,
		item.Diamond{}, c3, item.Diamond{},
		reinforced, sgitem.TierKit{Level: 1}, reinforced,
		item.Diamond{}, c3, item.Diamond{})
	shaped(st(sgitem.TierKit{Level: 3}), 3, 3,
		item.NetheriteIngot{}, c4, atomic,
		atomic, sgitem.TierKit{Level: 2}, atomic,
		sgitem.ResearchEssence{Kind: sgitem.EssenceAlchemy}, c4, sgitem.ResearchEssence{Kind: sgitem.EssenceFarming})
	shaped(st(sgitem.ConfigCopier{}), 3, 3,
		steel, c2, steel,
		infused, item.LapisLazuli{}, infused,
		nil, item.Stick{}, nil)
	// The magic sieve: an auto sieve (in any state) with eyes of ender, diamonds
	// and amethyst.
	add(machine.Item(machine.KindMagicSieve), []recipe.Item{st(machine.Machine{Kind: machine.KindAutoSieve}).WithValue("variants", true)}, n(item.EnderEye{}, 2), n(item.Diamond{}, 2), n(item.AmethystShard{}, 4))
	add(machine.Item(machine.KindMiner), n(iron, 6), n(block.NewHopper(), 1))
	add(machine.Item(machine.KindHarvester), n(iron, 5), n(block.NewHopper(), 1), n(block.NewChest(), 1))
	// The drawer: a ring of planks round a chest.
	shaped(st(storage.Drawer{}), 3, 3,
		planks, planks, planks,
		planks, block.NewChest(), planks,
		planks, planks, planks)
	add(st(farm.Barrel{Wood: "oak"}), n(planks, 6), n(Stick, 2))
	// The stone barrel; a torch under it is the least heat for it.
	add(st(farm.Barrel{Wood: farm.StoneWood}), n(block.Stone{}, 6), n(Stick, 2))
	// Hoppers: the wood hopper from planks and a chest; each hopper and its
	// reverse turn into one another when crafted alone.
	woodHopper, woodReverse := hopper.NewFlowHopper("wood_hopper", cube.FaceDown), hopper.NewFlowHopper("wood_hopper_reverse", cube.FaceUp)
	ironReverse := hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp)
	add(st(woodHopper), n(planks, 5), n(block.NewChest(), 1))
	add(st(woodReverse), n(woodHopper, 1))
	add(st(woodHopper), n(woodReverse, 1))
	add(st(ironReverse), n(block.NewHopper(), 1))
	add(st(block.NewHopper()), n(ironReverse, 1))
	add(st(sgitem.DrawerKey{}), n(item.IronNugget{}, 4), n(Stick, 1))
	add(st(sgitem.DrawerUpgrade{Kind: sgitem.UpgradeCapacity}), n(iron, 4), n(block.NewChest(), 1))
	add(st(sgitem.DrawerUpgrade{Kind: sgitem.UpgradeVoid}), n(item.Flint{}, 4), n(iron, 1))
	logRecipes(woodHopper)
}

// logRecipes are the shortcuts from logs (any log, stripped or not) to what
// is otherwise made of their planks, at the planks' cost or a little less.
// A result's research opens them with its other recipes. Their shapes keep
// clear of the vanilla recipes of logs (one log: planks; four in a square:
// wood).
func logRecipes(woodHopper world.Item) {
	log, iron := recipe.NewItemTag("minecraft:logs", 1), item.NewStack(item.IronIngot{}, 1)
	none := item.Stack{}
	shaped := func(out item.Stack, w, h int, cells ...recipe.Item) {
		recipe.Register(recipe.NewShaped(cells, out, recipe.NewShape(w, h), "crafting_table").WithoutMirror())
	}
	// Two logs on top of each other: sixteen sticks, as their planks give.
	shaped(item.NewStack(Stick, 16), 1, 2, log, log)
	// Eight logs around an empty middle: four chests.
	shaped(item.NewStack(block.NewChest(), 4), 3, 3,
		log, log, log,
		log, none, log,
		log, log, log)
	// Three logs in a V: a wood hopper (five planks and a chest's eight).
	shaped(item.NewStack(woodHopper, 1), 3, 2,
		log, none, log,
		none, log, none)
	// The iron hopper with two logs where its chest goes.
	shaped(item.NewStack(block.NewHopper(), 1), 3, 3,
		iron, none, iron,
		iron, log, iron,
		none, log, none)
}

// dyeUnlocks are the results the dyeing research opens: the 16 dyes (made
// from petals) and the blocks that take a colour.
func dyeUnlocks() []string {
	var out []string
	for _, c := range sgitem.PetalColours {
		for _, suffix := range []string{"dye", "stained_glass", "stained_glass_pane", "terracotta", "concrete_powder", "candle"} {
			out = append(out, c+"_"+suffix)
		}
	}
	return out
}

func init() { gameRecipes() }
