package dragonfactory

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Research (docs/design/02-dragonfactory.md §9): every crafting recipe is
// locked but oak planks, the crafting table and the research table. A
// research opens the recipes making its results for everyone on the
// island at once (recipes.go). Researches form a tree: each needs the ones
// before it. A research is paid with items, delivered from the inventory
// at the research table (in parts if need be). Trees and costs are drafts.

// Research is one node of the research tree.
type Research struct {
	ID string
	// Needs are the researches that must be done first.
	Needs []string
	// Cost are the items to deliver.
	Cost []Requirement
	// Unlocks are the item names (without "minecraft:") whose recipes the
	// research opens; every variant of a name opens together.
	Unlocks []string
	// Icon is the item shown for a research that opens no recipe (a project,
	// project.go).
	Icon string
}

var (
	oakPlanks = block.Planks{Wood: block.OakWood()}
	stick     = item.Stick{}
)

// Title is the name of the research in lang (the catalog, i18n).
func (r *Research) Title(lang string) string { return i18n.T(lang, "research."+r.ID+".name") }

// langOf is the catalog language of p (the default for a nil p).
func langOf(p *player.Player) string {
	if p == nil {
		return i18n.Default
	}
	return i18n.For(p)
}

// researchTree is every research in menu order; a research's needs come
// before it.
// researchTree is the research tree: the researches below with the research
// essences of researchEssenceCosts added (researchessence.go).
var researchTree = withEssenceCosts(baseResearchTree)

// baseResearchTree are the researches with their material costs.
var baseResearchTree = []Research{
	{ID: "stick", Cost: []Requirement{{oakPlanks, 4}}, Unlocks: []string{"stick"}},
	{ID: "pebbles", Needs: []string{"stick"}, Cost: []Requirement{{Pebble{Kind: PebbleStone}, 4}},
		Unlocks: []string{"cobblestone"}},
	{ID: "wood_tools", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 8}, {stick, 4}},
		Unlocks: []string{"wooden_pickaxe", "wooden_axe", "wooden_shovel", "wooden_hoe", "wooden_sword"}},
	{ID: "chest", Needs: []string{"stick"}, Cost: []Requirement{{oakPlanks, 32}}, Unlocks: []string{"chest"}},
	{ID: "wood_hopper", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 32}},
		Unlocks: []string{"dragonfactory:wood_hopper", "dragonfactory:wood_hopper_reverse"}},
	{ID: "slab", Needs: []string{"wood_tools"}, Cost: []Requirement{{oakPlanks, 16}}, Unlocks: []string{"oak_slab"}},
	{ID: "compost", Needs: []string{"slab"}, Cost: []Requirement{{oakPlanks, 32}}, Unlocks: []string{"composter"}},
	{ID: "barrel", Needs: []string{"compost"}, Cost: []Requirement{{oakPlanks, 32}, {stick, 4}}, Unlocks: []string{"dragonfactory:barrel_oak"}},
	{ID: "furnace", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 16}}, Unlocks: []string{"furnace"}},
	{ID: "stone_tools", Needs: []string{"wood_tools", "pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 16}, {stick, 8}},
		Unlocks: []string{"stone_pickaxe", "stone_axe", "stone_shovel", "stone_hoe", "stone_sword"}},
	{ID: "string", Needs: []string{"stone_tools"}, Cost: []Requirement{{BarkFiber{}, 8}}, Unlocks: []string{"string"}},
	{ID: "hammer", Needs: []string{"pebbles"}, Cost: []Requirement{{block.Cobblestone{}, 16}, {stick, 4}}, Unlocks: []string{"dragonfactory:hammer"}},
	{ID: "sieve", Needs: []string{"string", "hammer"}, Cost: []Requirement{{block.String{}, 8}, {oakPlanks, 16}}, Unlocks: []string{"dragonfactory:sieve", "dragonfactory:mesh_string", "dragonfactory:mesh_flint"}},
	{ID: "torch", Needs: []string{"furnace"}, Cost: []Requirement{{item.Charcoal{}, 8}, {stick, 8}}, Unlocks: []string{"torch"}},
	{ID: "woodwork", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 64}, {stick, 16}},
		Unlocks: []string{"oak_stairs", "oak_fence", "fence_gate", "wooden_door", "trapdoor", "ladder", "bowl"}},
	{ID: "masonry", Needs: []string{"furnace"}, Cost: []Requirement{{block.Stone{}, 32}, {block.Cobblestone{}, 32}},
		Unlocks: []string{"cobblestone_slab", "stone_stairs", "cobblestone_wall", "normal_stone_slab", "normal_stone_stairs", "stone_bricks"}},
	{ID: "glasswork", Needs: []string{"furnace"}, Cost: []Requirement{{block.Glass{}, 16}}, Unlocks: []string{"glass_pane", "glass_bottle"}},
	{ID: "iron", Needs: []string{"furnace", "sieve"}, Cost: []Requirement{{OreChunk{Ore: OreIron}, 16}}, Unlocks: []string{"iron_ore", "iron_ingot", "iron_block", "iron_nugget", "dragonfactory:mesh_iron"}},
	{ID: "bucket", Needs: []string{"iron", "barrel"}, Cost: []Requirement{{item.IronIngot{}, 6}}, Unlocks: []string{"bucket"}},
	{ID: "shears", Needs: []string{"iron", "wood_tools"}, Cost: []Requirement{{item.IronIngot{}, 4}}, Unlocks: []string{"shears"}},
	{ID: "stone_barrel", Needs: []string{"barrel", "furnace", "bucket", "torch"}, Cost: []Requirement{{block.Stone{}, 32}}, Unlocks: []string{"dragonfactory:barrel_stone"}},
	{ID: "hopper", Needs: []string{"iron", "chest"}, Cost: []Requirement{{item.IronIngot{}, 10}, {block.NewChest(), 4}}, Unlocks: []string{"hopper", "dragonfactory:iron_hopper_reverse"}},
	{ID: "generator", Needs: []string{"iron", "furnace"}, Cost: []Requirement{{item.IronIngot{}, 16}}, Unlocks: []string{"dragonfactory:generator"}},
	{ID: "laboratory", Needs: []string{"generator", "glasswork"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.Glass{}, 8}, {item.RedstoneWire{}, 8}},
		Unlocks: []string{"dragonfactory:laboratory"}},
	{ID: "energy_conduit", Needs: []string{"generator"}, Cost: []Requirement{{item.IronIngot{}, 16}}, Unlocks: []string{"dragonfactory:conduit_energy_1", "dragonfactory:conduit_binder_mix"}},
	{ID: "item_conduit", Needs: []string{"hopper"}, Cost: []Requirement{{item.IronIngot{}, 16}}, Unlocks: []string{"dragonfactory:conduit_item_1", "dragonfactory:conduit_binder_mix"}},
	{ID: "fluid_conduit", Needs: []string{"item_conduit", "glasswork"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.Glass{}, 8}}, Unlocks: []string{"dragonfactory:conduit_fluid_1", "dragonfactory:conduit_binder_mix"}},
	{ID: "crusher", Needs: []string{"hammer", "generator"}, Cost: []Requirement{{block.Cobblestone{}, 64}, {item.IronIngot{}, 8}}, Unlocks: []string{"dragonfactory:crusher"}},
	{ID: "auto_sieve", Needs: []string{"sieve", "hopper", "generator"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.String{}, 8}}, Unlocks: []string{"dragonfactory:auto_sieve"}},
	{ID: "iron_tools", Needs: []string{"iron", "stone_tools"}, Cost: []Requirement{{item.IronIngot{}, 32}},
		Unlocks: []string{"iron_pickaxe", "iron_axe", "iron_shovel", "iron_hoe", "iron_sword"}},
	{ID: "auto_miner", Needs: []string{"iron_tools", "hopper", "generator"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.NewHopper(), 2}}, Unlocks: []string{"dragonfactory:auto_miner"}},
	{ID: "drawer", Needs: []string{"chest"}, Cost: []Requirement{{oakPlanks, 32}, {block.NewChest(), 2}}, Unlocks: []string{"dragonfactory:drawer", "dragonfactory:drawer_key"}},
	{ID: "drawer_upgrades", Needs: []string{"drawer", "iron"}, Cost: []Requirement{{item.IronIngot{}, 16}},
		Unlocks: []string{"dragonfactory:drawer_upgrade_capacity", "dragonfactory:drawer_upgrade_void"}},
	{ID: "farming", Needs: []string{"wood_tools"}, Cost: []Requirement{{item.Wheat{}, 16}}, Unlocks: []string{"bread", "hay_block"}},
	{ID: "harvester", Needs: []string{"farming", "hopper", "generator"}, Cost: []Requirement{{item.IronIngot{}, 16}}, Unlocks: []string{"dragonfactory:harvester"}},
	{ID: "clay", Needs: []string{"furnace", "barrel"}, Cost: []Requirement{{item.ClayBall{}, 16}}, Unlocks: []string{"clay", "brick_block"}},
	{ID: "gold", Needs: []string{"iron"}, Cost: []Requirement{{OreChunk{Ore: OreGold}, 16}}, Unlocks: []string{"gold_ore", "gold_ingot", "gold_block", "gold_nugget", "dragonfactory:mesh_gold"}},
	{ID: "mesh_diamond", Needs: []string{"gold"}, Cost: []Requirement{{item.Diamond{}, 4}}, Unlocks: []string{"dragonfactory:mesh_diamond"}},
	{ID: "mesh_netherite", Needs: []string{"mesh_diamond"}, Cost: []Requirement{{item.NetheriteScrap{}, 4}}, Unlocks: []string{"dragonfactory:mesh_netherite", "netherite_ingot"}},
	{ID: "copper", Needs: []string{"furnace", "sieve"}, Cost: []Requirement{{OreChunk{Ore: OreCopper}, 16}}, Unlocks: []string{"copper_ore", "copper_ingot", "copper_block", "copper_nugget"}},
	{ID: "tin", Needs: []string{"copper"}, Cost: []Requirement{{OreChunk{Ore: MetalTin}, 16}}, Unlocks: []string{"dragonfactory:metal_dust_tin"}},
	{ID: "lead", Needs: []string{"tin", "iron"}, Cost: []Requirement{{OreChunk{Ore: MetalLead}, 16}}, Unlocks: []string{"dragonfactory:metal_dust_lead"}},
	{ID: "silver", Needs: []string{"gold"}, Cost: []Requirement{{OreChunk{Ore: MetalSilver}, 16}}, Unlocks: []string{"dragonfactory:metal_dust_silver"}},
	{ID: "nickel", Needs: []string{"gold"}, Cost: []Requirement{{OreChunk{Ore: MetalNickel}, 16}}, Unlocks: []string{"dragonfactory:metal_dust_nickel"}},
	{ID: "alloy_smelter", Needs: []string{"generator", "tin", "crusher"}, Cost: []Requirement{{item.IronIngot{}, 16}, {Ingot{Metal: MetalTin}, 8}}, Unlocks: []string{"dragonfactory:alloy_smelter"}},
	{ID: "energy_cell", Needs: []string{"generator", "lead"}, Cost: []Requirement{{Ingot{Metal: MetalLead}, 8}, {item.IronIngot{}, 8}}, Unlocks: []string{"dragonfactory:energy_cell"}},
	{ID: "conduit_upgrade_2", Needs: []string{"alloy_smelter", "silver", "energy_conduit", "item_conduit"}, Cost: []Requirement{{Ingot{Metal: MetalElectrum}, 8}}, Unlocks: []string{"dragonfactory:conduit_energy_2", "dragonfactory:conduit_item_2", "dragonfactory:conduit_fluid_2"}},
	{ID: "conduit_upgrade_3", Needs: []string{"conduit_upgrade_2"}, Cost: []Requirement{{Ingot{Metal: MetalSteel}, 8}, {item.Diamond{}, 4}}, Unlocks: []string{"dragonfactory:conduit_energy_3", "dragonfactory:conduit_item_3", "dragonfactory:conduit_fluid_3"}},
	{ID: "mob_simulator", Needs: []string{"alloy_smelter", "crusher"}, Cost: []Requirement{{Ingot{Metal: MetalBronze}, 8}, {Essence{Mob: MobZombie}, 4}, {Essence{Mob: MobSkeleton}, 4}}, Unlocks: []string{"dragonfactory:mob_simulator"}},
	{ID: "auto_composter", Needs: []string{"compost", "generator"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.Dirt{}, 16}}, Unlocks: []string{"dragonfactory:auto_composter"}},
	{ID: "fluid_tank", Needs: []string{"fluid_conduit", "bucket"}, Cost: []Requirement{{item.IronIngot{}, 16}, {block.Glass{}, 8}}, Unlocks: []string{"dragonfactory:fluid_tank"}},
	{ID: "crucible", Needs: []string{"stone_barrel", "generator", "fluid_tank", "tin"}, Cost: []Requirement{{item.IronIngot{}, 16}, {Ingot{Metal: MetalBronze}, 8}}, Unlocks: []string{"dragonfactory:crucible"}},
	{ID: "lava_generator", Needs: []string{"crucible", "alloy_smelter"}, Cost: []Requirement{{Ingot{Metal: MetalSteel}, 8}}, Unlocks: []string{"dragonfactory:lava_generator"}},
	{ID: "ore_washer", Needs: []string{"fluid_tank", "alloy_smelter"}, Cost: []Requirement{{item.IronIngot{}, 12}, {Ingot{Metal: MetalBronze}, 4}}, Unlocks: []string{"dragonfactory:ore_washer"}},
	{ID: "dimension_gate", Needs: []string{"mob_simulator", "lava_generator"}, Cost: []Requirement{{Ingot{Metal: MetalSteel}, 8}, {Essence{Mob: MobZombie}, 8}, {Essence{Mob: MobSkeleton}, 8}}, Unlocks: []string{"dragonfactory:dimension_gate"}},
	{ID: "diamond_tools", Needs: []string{"iron_tools", "mesh_diamond"}, Cost: []Requirement{{item.Diamond{}, 8}},
		Unlocks: []string{"diamond_pickaxe", "diamond_axe", "diamond_shovel", "diamond_hoe", "diamond_sword"}},
	{ID: "storage_controller", Needs: []string{"drawer_upgrades", "item_conduit", "tin"}, Cost: []Requirement{{item.IronIngot{}, 16}, {Ingot{Metal: MetalBronze}, 8}}, Unlocks: []string{"dragonfactory:storage_controller", "dragonfactory:storage_link"}},
	{ID: "tinker_table", Needs: []string{"iron_tools", "tin"}, Cost: []Requirement{{item.IronIngot{}, 24}, {item.Flint{}, 8}}, Unlocks: []string{"dragonfactory:tinker_table"}},
	{ID: "copper_tools", Needs: []string{"copper", "iron_tools"}, Cost: []Requirement{{item.CopperIngot{}, 12}},
		Unlocks: []string{"copper_pickaxe", "copper_axe", "copper_shovel", "copper_hoe", "copper_sword"}},
	{ID: "copper_armor", Needs: []string{"copper_tools"}, Cost: []Requirement{{item.CopperIngot{}, 12}},
		Unlocks: []string{"copper_helmet", "copper_chestplate", "copper_leggings", "copper_boots"}},
	{ID: "iron_armor", Needs: []string{"iron_tools"}, Cost: []Requirement{{item.IronIngot{}, 16}},
		Unlocks: []string{"iron_helmet", "iron_chestplate", "iron_leggings", "iron_boots"}},
	{ID: "gold_tools", Needs: []string{"gold", "iron_tools"}, Cost: []Requirement{{item.GoldIngot{}, 10}},
		Unlocks: []string{"golden_pickaxe", "golden_axe", "golden_shovel", "golden_hoe", "golden_sword"}},
	{ID: "gold_armor", Needs: []string{"gold_tools"}, Cost: []Requirement{{item.GoldIngot{}, 12}},
		Unlocks: []string{"golden_helmet", "golden_chestplate", "golden_leggings", "golden_boots"}},
	{ID: "diamond_armor", Needs: []string{"diamond_tools", "iron_armor"}, Cost: []Requirement{{item.Diamond{}, 12}},
		Unlocks: []string{"diamond_helmet", "diamond_chestplate", "diamond_leggings", "diamond_boots"}},
	{ID: "archery", Needs: []string{"stone_tools", "string"}, Cost: []Requirement{{block.String{}, 6}, {stick, 6}},
		Unlocks: []string{"bow"}},
	{ID: "ironwork", Needs: []string{"iron_tools", "torch"}, Cost: []Requirement{{item.IronIngot{}, 12}},
		Unlocks: []string{"iron_bars", "iron_chain", "lantern", "flint_and_steel"}},
	{ID: "workbenches", Needs: []string{"iron", "woodwork", "masonry"}, Cost: []Requirement{{item.IronIngot{}, 8}, {oakPlanks, 16}, {item.Flint{}, 4}},
		Unlocks: []string{"stonecutter_block", "grindstone", "fletching_table", "loom", "smithing_table"}},
	{ID: "anvil", Needs: []string{"ironwork", "masonry"}, Cost: []Requirement{{item.IronIngot{}, 24}},
		Unlocks: []string{"anvil"}},
	{ID: "smelters", Needs: []string{"iron_tools", "furnace"}, Cost: []Requirement{{item.IronIngot{}, 12}, {block.Stone{}, 8}},
		Unlocks: []string{"blast_furnace", "smoker", "campfire"}},
	{ID: "redstone_work", Needs: []string{"iron", "generator"}, Cost: []Requirement{{item.RedstoneWire{}, 16}},
		Unlocks: []string{"redstone", "redstone_torch", "redstone_block", "lever", "noteblock"}},
	{ID: "compression", Needs: []string{"mesh_diamond", "furnace"}, Cost: []Requirement{{item.Coal{}, 32}, {item.LapisLazuli{}, 8}},
		Unlocks: []string{"coal", "coal_block", "lapis_lazuli", "lapis_block", "emerald", "emerald_block", "diamond", "diamond_block"}},
	{ID: "quartz_work", Needs: []string{"masonry", "sieve"}, Cost: []Requirement{{item.NetherQuartz{}, 8}},
		Unlocks: []string{"quartz_block", "quartz_bricks", "quartz_pillar", "chiseled_quartz_block", "quartz_slab", "quartz_stairs"}},
	{ID: "sandstone", Needs: []string{"masonry"}, Cost: []Requirement{{block.Sand{}, 32}},
		Unlocks: []string{"sandstone", "cut_sandstone", "chiseled_sandstone", "sandstone_slab", "sandstone_stairs"}},
	{ID: "crystals", Needs: []string{"glasswork", "sieve"}, Cost: []Requirement{{item.AmethystShard{}, 8}},
		Unlocks: []string{"amethyst_block", "tinted_glass"}},
	{ID: "navigation", Needs: []string{"copper", "gold", "redstone_work"}, Cost: []Requirement{{item.CopperIngot{}, 4}, {item.GoldIngot{}, 4}, {item.RedstoneWire{}, 4}},
		Unlocks: []string{"compass", "clock", "spyglass"}},
	{ID: "bone_craft", Needs: []string{"mob_simulator"}, Cost: []Requirement{{item.Bone{}, 8}},
		Unlocks: []string{"bone_meal", "bone_block"}},
	{ID: "slime_block", Needs: []string{"mob_simulator"}, Cost: []Requirement{{item.Slimeball{}, 9}},
		Unlocks: []string{"slime", "slime_ball"}},
	{ID: "blaze_craft", Needs: []string{"mob_simulator", "mesh_netherite"}, Cost: []Requirement{{item.BlazeRod{}, 4}},
		Unlocks: []string{"blaze_powder", "magma_cream", "magma", "ender_eye", "fire_charge"}},
	{ID: "ender_chest", Needs: []string{"blaze_craft", "diamond_tools"}, Cost: []Requirement{{item.EnderPearl{}, 4}, {block.Obsidian{}, 4}},
		Unlocks: []string{"ender_chest"}},
	{ID: "electric_furnace", Needs: []string{"generator", "furnace"}, Cost: []Requirement{{item.IronIngot{}, 12}},
		Unlocks: []string{"dragonfactory:electric_furnace"}},
	{ID: "item_vacuum", Needs: []string{"generator", "hopper"}, Cost: []Requirement{{item.IronIngot{}, 12}},
		Unlocks: []string{"dragonfactory:item_vacuum"}},
	{ID: "tree_feller", Needs: []string{"generator", "iron_tools"}, Cost: []Requirement{{item.IronIngot{}, 12}},
		Unlocks: []string{"dragonfactory:tree_feller"}},
	{ID: "auto_crafter", Needs: []string{"generator", "hopper", "redstone_work"}, Cost: []Requirement{{item.IronIngot{}, 16}, {item.RedstoneWire{}, 8}},
		Unlocks: []string{"dragonfactory:auto_crafter"}},
	{ID: "magic_botany", Needs: []string{"dimension_gate"}, Cost: []Requirement{{PlantSeed{Seed: "flower_red"}, 1}, {PlantSeed{Seed: "flower_yellow"}, 1}, {PlantSeed{Seed: "flower_blue"}, 1}}, Unlocks: []string{"dragonfactory:seed_ore_iron", "dragonfactory:seed_ore_gold", "dragonfactory:seed_ore_copper", "dragonfactory:seed_ore_tin", "dragonfactory:seed_ore_lead", "dragonfactory:seed_ore_silver", "dragonfactory:seed_ore_nickel"}},
	{ID: "magic_gems", Needs: []string{"magic_botany"}, Cost: []Requirement{{PlantSeed{Seed: "ore_gold"}, 2}}, Unlocks: []string{"dragonfactory:seed_gem_lapis", "dragonfactory:seed_gem_redstone", "dragonfactory:seed_gem_quartz", "dragonfactory:seed_gem_coal", "dragonfactory:seed_gem_diamond", "dragonfactory:seed_gem_emerald"}},
	{ID: "magic_essence", Needs: []string{"magic_gems", "blaze_craft"}, Cost: []Requirement{{PlantSeed{Seed: "gem_redstone"}, 2}}, Unlocks: []string{"dragonfactory:seed_ess_zombie", "dragonfactory:seed_ess_skeleton", "dragonfactory:seed_ess_creeper", "dragonfactory:seed_ess_spider", "dragonfactory:seed_ess_slime", "dragonfactory:seed_ess_enderman", "dragonfactory:seed_ess_blaze"}},
	{ID: "magic_sieve", Needs: []string{"auto_sieve", "mob_simulator", "magic_botany"}, Cost: []Requirement{{item.Diamond{}, 8}, {item.EnderPearl{}, 8}},
		Unlocks: []string{"dragonfactory:magic_sieve"}},
	{ID: "machine_tier_1", Needs: []string{"auto_crafter", "energy_cell", "conduit_upgrade_2"}, Cost: []Requirement{{Ingot{Metal: MetalSteel}, 16}, {Ingot{Metal: MetalElectrum}, 16}, {item.Diamond{}, 4}},
		Unlocks: []string{"dragonfactory:circuit_1", "dragonfactory:circuit_2", "dragonfactory:kit_1"}},
	{ID: "machine_tier_2", Needs: []string{"machine_tier_1", "dimension_gate", "lava_generator"}, Cost: []Requirement{{item.Diamond{}, 16}, {Ingot{Metal: MetalInvar}, 16}},
		Unlocks: []string{"dragonfactory:circuit_3", "dragonfactory:kit_2"}},
	{ID: "machine_tier_3", Needs: []string{"machine_tier_2", "magic_essence", "magic_sieve"}, Cost: []Requirement{{item.NetheriteIngot{}, 2}},
		Unlocks: []string{"dragonfactory:circuit_4", "dragonfactory:kit_3"}},
	{ID: "config_copier", Needs: []string{"machine_tier_1", "item_conduit", "auto_crafter"}, Cost: []Requirement{{Ingot{Metal: MetalSteel}, 8}, {item.LapisLazuli{}, 8}},
		Unlocks: []string{"dragonfactory:config_copier"}},
	{ID: "dyeing", Needs: []string{"magic_botany", "glasswork"}, Cost: []Requirement{{Petal{Colour: "red"}, 4}, {Petal{Colour: "yellow"}, 4}, {Petal{Colour: "blue"}, 4}},
		Unlocks: dyeUnlocks()},
	{ID: "harvest", Needs: []string{"farming"}, Cost: []Requirement{{item.MelonSlice{}, 16}, {block.Pumpkin{}, 4}},
		Unlocks: []string{"melon_block", "pumpkin_pie", "melon_seeds", "pumpkin_seeds"}},
	{ID: "golden_food", Needs: []string{"gold", "harvest"}, Cost: []Requirement{{item.GoldIngot{}, 8}, {block.Carrot{}, 4}},
		Unlocks: []string{"golden_apple", "golden_carrot", "glistering_melon_slice"}},
	{ID: "project_industry", Icon: "dragonfactory:storage_controller", Needs: []string{"conduit_upgrade_3", "storage_controller", "auto_crafter", "lava_generator"},
		Cost: []Requirement{{Ingot{Metal: MetalSteel}, 32}, {Ingot{Metal: MetalElectrum}, 32}, {Ingot{Metal: MetalInvar}, 32}, {Ingot{Metal: MetalBronze}, 32}, {item.Diamond{}, 16}}},
	{ID: "project_hunter", Icon: "diamond_sword", Needs: []string{"dimension_gate", "tinker_table", "diamond_armor", "blaze_craft"},
		Cost: []Requirement{{item.BlazeRod{}, 16}, {item.EnderPearl{}, 16}, {item.Slimeball{}, 16}, {item.SpiderEye{}, 16}, {item.Bone{}, 32}}},
	{ID: "project_botanist", Icon: "dragonfactory:petal_red", Needs: []string{"magic_essence", "dyeing"},
		Cost: []Requirement{{PlantSeed{Seed: "ess_enderman"}, 2}, {PlantSeed{Seed: "ess_blaze"}, 2}, {PlantSeed{Seed: "gem_diamond"}, 2}, {PlantSeed{Seed: "gem_emerald"}, 2}, {Essence{Mob: MobEnder}, 8}, {Essence{Mob: MobBlaze}, 8}}},
	{ID: "project_dragon", Needs: []string{"project_industry", "project_hunter", "project_botanist"},
		Cost:    []Requirement{{item.Diamond{}, 32}, {item.Emerald{}, 32}, {item.NetheriteIngot{}, 4}, {Ingot{Metal: MetalSteel}, 32}},
		Unlocks: []string{"dragon_egg"}},
	{ID: "kelp", Needs: []string{"furnace"}, Cost: []Requirement{{item.DriedKelp{}, 16}}, Unlocks: []string{"dried_kelp_block"}},
}

// baseUnlocks are the results open without research.
var baseUnlocks = []string{"oak_planks", "crafting_table", "dragonfactory:research_table"}

// researchByID returns a research of the tree.
func researchByID(id string) (*Research, bool) {
	for i := range researchTree {
		if researchTree[i].ID == id {
			return &researchTree[i], true
		}
	}
	return nil, false
}

// unlockName is the item name of an unlock as items encode it.
func unlockName(n string) string {
	if strings.Contains(n, ":") {
		return n
	}
	return "minecraft:" + n
}

// researched reports whether the save has done a research.
func (s *session) researched(id string) bool { return s.state.Researched[id] }

// researchOpen reports whether a research can be done: not done yet and
// everything it needs is done.
func (s *session) researchOpen(r *Research) bool {
	if s.researched(r.ID) {
		return false
	}
	for _, n := range r.Needs {
		if !s.researched(n) {
			return false
		}
	}
	return true
}

// researchBlocker returns the one research that is still to do before r
// when r is only one step from open: that research is open itself, and it is
// the only one r needs that is not done.
func (s *session) researchBlocker(r *Research) (*Research, bool) {
	if s.researched(r.ID) {
		return nil, false
	}
	var left *Research
	for _, n := range r.Needs {
		if s.researched(n) {
			continue
		}
		if left != nil {
			return nil, false
		}
		need, ok := researchByID(n)
		if !ok {
			return nil, false
		}
		left = need
	}
	if left == nil || !s.researchOpen(left) {
		return nil, false
	}
	return left, true
}

// researchPaid is how many of an item of a research's cost an older save
// had already delivered in parts (partial delivery is gone: a research is
// paid whole, at once; the count only keeps what older saves gave).
func (s *session) researchPaid(r *Research, it world.Item) int {
	return s.state.ResearchPaid[r.ID][itemKey(it)]
}

// researchNeed is how many of an item of a research's cost are still to be
// spent.
func (s *session) researchNeed(r *Research, c Requirement) int {
	return max(0, c.Count-s.researchPaid(r, c.Item))
}

// researchMissing lists what p lacks to do a research now: the cost items
// with how many more they need.
func (s *session) researchMissing(p *player.Player, r *Research) []Requirement {
	var missing []Requirement
	for _, c := range r.Cost {
		if short := s.researchNeed(r, c) - countPlain(p.Inventory(), c.Item); short > 0 {
			missing = append(missing, Requirement{c.Item, short})
		}
	}
	return missing
}

// doResearch spends the whole cost from p's inventory and completes the
// research. A research is never paid in parts: when p lacks anything it
// takes nothing and returns what is missing.
func (s *session) doResearch(p *player.Player, tx *world.Tx, r *Research) (missing []Requirement, done bool) {
	if !s.researchOpen(r) {
		return nil, false
	}
	if missing = s.researchMissing(p, r); len(missing) > 0 {
		return missing, false
	}
	for _, c := range r.Cost {
		takePlain(p.Inventory(), c.Item, s.researchNeed(r, c))
	}
	pourEssences(tx, p, r.Cost)
	s.completeResearch(tx, r)
	account.Grant(p, achResearch)
	return nil, true
}

// completeResearch marks a research done and opens its recipes for
// everyone on the island.
func (s *session) completeResearch(tx *world.Tx, r *Research) {
	if s.state.Researched == nil {
		s.state.Researched = map[string]bool{}
	}
	s.state.Researched[r.ID] = true
	countAll(tx, "research_done", 1)
	if spent := essencesIn(r.Cost); spent > 0 {
		s.advance(tx, advEssenceResearch)
		countAll(tx, "essence_spent", int64(spent))
	}
	if strings.HasPrefix(r.ID, "project_") && r.ID != "project_dragon" {
		countAll(tx, "project_done", 1)
	}
	if len(s.state.Researched) >= len(researchTree) {
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), achAllResearch)
		}
	}
	delete(s.state.ResearchPaid, r.ID)
	s.storeState()
	s.sendRecipes(tx)
	s.updateBoards(tx)
	for e := range tx.Players() {
		p := e.(*player.Player)
		after := researchText(langOf(p), r.ID).After
		p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.research.completeResearch.1"), r.Title(langOf(p)), s.unlockNames(r))
		if after != "" {
			p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.research.completeResearch.2"), after)
		}
	}
}

// unlockNames lists a research's results in Korean.
func (s *session) unlockNames(r *Research) string {
	var names []string
	for _, n := range r.Unlocks {
		if it, ok := world.ItemByName(unlockName(n), 0); ok {
			names = append(names, itemName(item.NewStack(it, 1)))
		} else {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

// researchIcon is a research's form icon: its first result.
func researchIcon(r *Research) string {
	if len(r.Unlocks) == 0 {
		if it, ok := world.ItemByName(unlockName(r.Icon), 0); ok {
			return iconOf(item.NewStack(it, 1))
		}
		return iconMission
	}
	if it, ok := world.ItemByName(unlockName(r.Unlocks[0]), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}

// showResearch shows the research list on the tab of researches still to do.
func (s *session) showResearch(p *player.Player) {
	count(p, "research_open", 1)
	s.showResearchTab(p, false, 0)
}

// showResearchTab shows the research list as a grid of item slots (the
// representative item of each research) under two tabs, the researches
// still to do and the done ones.
func (s *session) showResearchTab(p *player.Player, done bool, page int) {
	body, slots, actions := s.researchListPage(p, done, page)
	ui.ItemGrid(p, i18n.T(i18n.Of(p), "dragonfactory.research.showResearchTab.1"), body, slots, actions...)
}

// researchTip is the hover text of a research in the list: its name in
// colour and the short description.
func researchTip(lang, colour string, r *Research) string {
	tip := colour + r.Title(lang)
	if short := researchText(lang, r.ID).Short; short != "" {
		tip += "\n§7" + short
	}
	return tip
}

// researchList builds the research list of a tab. A research is hidden until
// everything before it is done, so the tree is found step by step. The
// tooltip of a research to do is its name in yellow, in green when p has
// all it spends and can do it now. p may be nil.
func (s *session) researchList(p *player.Player, done bool) (string, []ui.ItemSlot, []ui.Option) {
	return s.researchListPage(p, done, 0)
}

// researchPageSize is how many researches a page of the list holds: the form
// is slow to open with many buttons, so a long list is split into pages.
const researchPageSize = 33

// researchListPage is one page of the research list of a tab.
func (s *session) researchListPage(p *player.Player, done bool, page int) (string, []ui.ItemSlot, []ui.Option) {
	var slots, previews []ui.ItemSlot
	todoN, doneN := 0, 0
	for i := range researchTree {
		r := &researchTree[i]
		show := func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, r) }
		switch {
		case s.researched(r.ID):
			doneN++
			if done {
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Count: "§a✔", Tip: researchTip(langOf(p), "§7", r), Run: show})
			}
		case s.researchOpen(r):
			todoN++
			if !done {
				colour := "§e"
				if p != nil && len(s.researchMissing(p, r)) == 0 {
					colour = "§a"
				}
				slots = append(slots, ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(langOf(p), colour, r), Run: show})
			}
		default:
			// one step from open: shown in grey after the ones to do
			if _, ok := s.researchBlocker(r); ok && !done {
				previews = append(previews, ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(langOf(p), "§8", r), Run: show})
			}
		}
	}
	slots = append(slots, previews...)
	body := i18n.T(i18n.Of(p), "dragonfactory.research.researchList.1")
	if len(slots) == 0 {
		if done {
			body += i18n.T(i18n.Of(p), "dragonfactory.research.researchList.2")
		} else {
			body += i18n.T(i18n.Of(p), "dragonfactory.research.researchList.3")
		}
	}
	pages := max(1, (len(slots)+researchPageSize-1)/researchPageSize)
	page = min(max(page, 0), pages-1)
	slots = slots[min(page*researchPageSize, len(slots)):min((page+1)*researchPageSize, len(slots))]
	tabs := []ui.Option{
		ui.Tab(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchList.4"), todoN), "", !done, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showResearchTab(p, false, 0)
			}
		}),
		ui.Tab(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchList.5"), doneN), "", done, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showResearchTab(p, true, 0)
			}
		}),
	}
	if pages > 1 {
		turn := func(label string, to int) ui.Option {
			return ui.Tab(label, "", false, func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showResearchTab(p, done, to)
				}
			})
		}
		if page > 0 {
			tabs = append(tabs, turn(fmt.Sprintf(i18n.T(i18n.Of(p), "researchpage.prev"), page, pages), page-1))
		}
		if page < pages-1 {
			tabs = append(tabs, turn(fmt.Sprintf(i18n.T(i18n.Of(p), "researchpage.next"), page+2, pages), page+1))
		}
	}
	tabs = append(tabs, s.backOption())
	return body, slots, tabs
}

// showResearchNode shows one research as an item detail form
// (ui.ItemDetail): on the left its big item, name, the research button (or
// "연구 완료") and what it is for; on the right the materials it spends and
// the recipes it opens as item slots (the count at the bottom right, the
// name and owned count in the tooltip).
func (s *session) showResearchNode(p *player.Player, r *Research) {
	d, rows, opts := s.researchNode(p, r)
	ui.ItemDetail(p, i18n.T(i18n.Of(p), "dragonfactory.research.showResearchNode.1")+r.Title(langOf(p)), d, rows, opts...)
}

// researchNode builds the screen of a research. p may be nil (the owned
// counts are then left out).
func (s *session) researchNode(p *player.Player, r *Research) (ui.Detail, []ui.SlotRow, []ui.Option) {
	txt := researchText(langOf(p), r.ID)
	done := s.researched(r.ID)
	var b strings.Builder
	if txt.About != "" {
		fmt.Fprintf(&b, "§f%s\n", txt.About)
	}
	d := ui.Detail{Icon: researchIcon(r), Name: "§6" + r.Title(langOf(p))}
	var opts []ui.Option
	switch {
	case done:
		d.Status = i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.1")
		if txt.After != "" {
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.2"), txt.After)
		}
	case s.researchOpen(r):
		if txt.Hint != "" {
			fmt.Fprintf(&b, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.3"), txt.Hint)
		}
		opts = append(opts, s.opt(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.4"), iconMission, func(p *player.Player, tx *world.Tx) {
			missing, ok := s.doResearch(p, tx, r)
			switch {
			case ok:
				ui.Flash(p, "%s", i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.5"))
				s.showResearch(p)
				return
			case len(missing) > 0:
				var names []string
				for _, m := range missing {
					names = append(names, costText(m))
				}
				ui.Notify(p, i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.6"), strings.Join(names, ", "))
			}
			s.showResearchNode(p, r)
		}))
	}
	// What is still to do before it: a button that goes to that research.
	for _, n := range r.Needs {
		if need, ok := researchByID(n); ok && !done && !s.researched(n) {
			opts = append(opts, s.opt(fmt.Sprintf(i18n.T(i18n.Of(p), "researchneeds.button"), need.Title(langOf(p))), researchIcon(need), func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, need) }))
		}
	}
	d.Body = b.String()
	again := func(p *player.Player, _ *world.Tx) { s.showResearchNode(p, r) }
	// The materials: the count shown is what the research spends (green
	// once p has enough, or the research is done); the tooltip says what
	// p has.
	var cost []ui.ItemSlot
	for _, c := range r.Cost {
		need := s.researchNeed(r, c)
		st := item.NewStack(c.Item, 1)
		mark, tip := "§f", fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.7"), itemName(st), need)
		switch {
		case done:
			mark, tip = "§a", fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.8"), itemName(st), c.Count)
			need = c.Count
		case p != nil:
			have := countPlain(p.Inventory(), c.Item)
			if have >= need {
				mark = "§a"
			}
			tip += fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.9"), mark, have)
		}
		cost = append(cost, ui.ItemSlot{Icon: iconOf(st), Count: mark + strconv.Itoa(need), Tip: tip, Run: again})
	}
	// The recipes it opens.
	var opens []ui.ItemSlot
	for _, n := range r.Unlocks {
		it, ok := world.ItemByName(unlockName(n), 0)
		if !ok {
			continue
		}
		st := item.NewStack(it, 1)
		opens = append(opens, ui.ItemSlot{Icon: iconOf(st), Tip: "§f" + itemName(st) + i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.10"), Run: again})
	}
	rows := []ui.SlotRow{{Heading: i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.11"), Slots: cost}, {Heading: i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.12"), Slots: opens}}
	back := ui.Back(i18n.T(i18n.Of(p), "dragonfactory.research.researchNode.13"), func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
	return d, rows, append(opts, s.opt(back.Label, back.Icon, back.Run))
}

// ResearchTable is the block where research is done. It is crafted from two
// crafting tables (open from the start).
type ResearchTable struct{}

var researchTableHash = block.NextHash()

func (ResearchTable) EncodeItem() (string, int16) { return "dragonfactory:research_table", 0 }
func (ResearchTable) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:research_table", nil
}
func (ResearchTable) Hash() (uint64, uint64)             { return researchTableHash, 0 }
func (ResearchTable) Model() world.BlockModel            { return model.Solid{} }
func (ResearchTable) Name() string                       { return i18n.T(i18n.Default, "name.ResearchTable") }
func (ResearchTable) Category() category.Category        { return category.Items() }
func (ResearchTable) Properties() customblock.Properties { return cubeProperties("research_table") }
func (t ResearchTable) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2.5, BlastResistance: 2.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(t, 1)} },
	}
}

// Activate opens the research tree.
func (ResearchTable) Activate(_ cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	if !s.debounced(p) {
		s.showResearch(p)
	}
	return true
}

func init() {
	world.RegisterBlock(ResearchTable{})
	world.RegisterItem(ResearchTable{})
	table := item.NewStack(block.CraftingTable{}, 1)
	recipe.Register(recipe.NewShapeless([]recipe.Item{table, table}, item.NewStack(ResearchTable{}, 1), "crafting_table"))
	gameRecipes()
}

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
	cobble, planks, iron := block.Cobblestone{}, oakPlanks, item.IronIngot{}
	// The hammer (and its later tiers, with another stone): two stones and
	// two sticks in a diagonal pattern.
	shaped(st(Hammer{}), 3, 3,
		nil, cobble, nil,
		nil, stick, cobble,
		stick, nil, nil)
	// The sieve: a row of planks on two stick legs. The mesh is made and
	// fitted separately (sieve.go).
	shaped(machineItem(KindSieve), 3, 2,
		planks, planks, planks,
		stick, nil, stick)
	// The combustion generator and the first energy conduit.
	add(machineItem(KindGenerator), n(iron, 6), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 2))
	// The conduits (docs/design/06-power-and-conduits.md §5.1): sand and gravel
	// in a checkerboard make the binder mix, a furnace bakes it into the
	// binder; the basic conduits are binders above and below a row of
	// redstone (energy), an iron nugget, a chest and an iron nugget (item) or
	// glass (fluid).
	// The better tiers wait for the ore design.
	sand, gravel, binder := block.Sand{}, block.Gravel{}, ConduitBinder{}
	shaped(item.NewStack(ConduitBinderMix{}, 6), 3, 3,
		sand, gravel, sand,
		gravel, sand, gravel,
		sand, gravel, sand)
	redstone, nugget := item.RedstoneWire{}, item.IronNugget{}
	shaped(item.NewStack(EnergyConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		redstone, redstone, redstone,
		binder, binder, binder)
	shaped(item.NewStack(ItemConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		nugget, block.NewChest(), nugget,
		binder, binder, binder)
	shaped(item.NewStack(FluidConduit{Tier: 1}, 8), 3, 3,
		binder, binder, binder,
		block.Glass{}, block.Glass{}, block.Glass{},
		binder, binder, binder)
	add(machineItem(KindCrusher), n(cobble, 6), n(iron, 3))
	// The alloy smelter and the energy cell (docs/design/07-progression-expansion.md).
	add(machineItem(KindAlloy), n(iron, 6), n(block.NewFurnace(cube.North), 1), n(block.NewHopper(), 1))
	add(machineItem(KindEnergyCell), n(Ingot{Metal: MetalLead}, 3), n(item.RedstoneWire{}, 4), n(iron, 2))
	add(machineItem(KindAutoComposter), n(iron, 6), n(block.Composter{}, 1), n(block.NewHopper(), 1))
	add(machineItem(KindTank), n(item.IronIngot{}, 3), n(block.Glass{}, 6))
	add(machineItem(KindCrucible), n(Ingot{Metal: MetalBronze}, 4), n(block.NewFurnace(cube.North), 1), n(block.NewHopper(), 1))
	add(machineItem(KindLavaGenerator), n(Ingot{Metal: MetalSteel}, 3), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 3), n(block.Glass{}, 2))
	add(machineItem(KindWasher), n(Ingot{Metal: MetalBronze}, 4), n(block.Glass{}, 4), n(block.NewHopper(), 1))
	add(machineItem(KindGate), n(Ingot{Metal: MetalSteel}, 4), n(block.Obsidian{}, 3), n(item.EnderPearl{}, 2))
	add(machineItem(KindStorageController), n(Ingot{Metal: MetalBronze}, 4), n(Drawer{}, 1), n(block.NewHopper(), 1), n(item.RedstoneWire{}, 2))
	add(item.NewStack(Machine{Kind: KindStorageLink}, 4), n(iron, 2), n(item.RedstoneWire{}, 1))
	add(machineItem(KindTinker), n(item.IronIngot{}, 6), n(Ingot{Metal: MetalBronze}, 2), n(block.CraftingTable{}, 1))
	add(machineItem(KindSimulator), n(Ingot{Metal: MetalSteel}, 3), n(block.Glass{}, 3), n(item.RedstoneWire{}, 3))
	// The dragon egg, the last project's prize (project.go).
	add(item.NewStack(block.DragonEgg{}, 1), n(item.EnderEye{}, 1), n(block.Obsidian{}, 8))
	// The automation equipment (docs/design/09-automation-equipment.md).
	add(machineItem(KindFurnace), n(iron, 5), n(block.NewFurnace(cube.North), 1), n(item.RedstoneWire{}, 2))
	add(machineItem(KindVacuum), n(iron, 4), n(block.NewHopper(), 1), n(item.RedstoneWire{}, 2))
	add(machineItem(KindFeller), n(iron, 4), n(item.Axe{Tier: item.ToolTierIron}, 1), n(block.NewHopper(), 1))
	add(machineItem(KindCrafter), n(block.CraftingTable{}, 1), n(block.NewHopper(), 2), n(iron, 3), n(item.RedstoneWire{}, 3))
	add(machineItem(KindLab), n(iron, 3), n(block.Glass{}, 3), n(item.RedstoneWire{}, 3))
	// Better conduits: a conduit of the tier below with an electrum ingot (2) or,
	// for the third tier, a steel ingot and a diamond (3).
	for _, tier := range []int{2, 3} {
		for _, kind := range []string{"energy", "item", "fluid"} {
			below := conduitOf(kind, tier-1)
			if tier == 2 {
				add(st(conduitOf(kind, tier)), []recipe.Item{st(below)}, n(Ingot{Metal: MetalElectrum}, 1))
			} else {
				add(st(conduitOf(kind, tier)), []recipe.Item{st(below)}, n(Ingot{Metal: MetalSteel}, 1), n(item.Diamond{}, 1))
			}
		}
	}
	// The sieve is a block with many states (how full it is); the recipe takes it
	// whatever its state, as the client matches recipe blocks by the meta it is
	// told (any), not by the one state an empty sieve has.
	anySieve := []recipe.Item{st(Machine{Kind: KindSieve}).WithValue("variants", true)}
	add(machineItem(KindAutoSieve), n(iron, 4), anySieve, n(block.NewHopper(), 1))
	// The machine tiers' circuits and kits (docs/design/13-machine-tiers.md §3).
	steel, electrum := Ingot{Metal: MetalSteel}, Ingot{Metal: MetalElectrum}
	infused, reinforced, atomic := TierAlloy{Kind: "infused"}, TierAlloy{Kind: "reinforced"}, TierAlloy{Kind: "atomic"}
	c1, c2, c3, c4 := Circuit{Level: 1}, Circuit{Level: 2}, Circuit{Level: 3}, Circuit{Level: 4}
	shaped(st(c1), 3, 1, redstone, infused, redstone)
	shaped(st(c2), 3, 1, reinforced, c1, reinforced)
	shaped(st(c3), 3, 1, atomic, c2, atomic)
	shaped(st(c4), 3, 3,
		atomic, item.NetheriteScrap{}, atomic,
		ResearchEssence{Kind: EssenceHunting}, c3, ResearchEssence{Kind: EssenceMagic},
		atomic, nil, atomic)
	shaped(st(TierKit{Level: 1}), 3, 3,
		steel, c2, steel,
		electrum, Machine{Kind: KindEnergyCell}, electrum,
		steel, c2, steel)
	shaped(st(TierKit{Level: 2}), 3, 3,
		item.Diamond{}, c3, item.Diamond{},
		reinforced, TierKit{Level: 1}, reinforced,
		item.Diamond{}, c3, item.Diamond{})
	shaped(st(TierKit{Level: 3}), 3, 3,
		item.NetheriteIngot{}, c4, atomic,
		atomic, TierKit{Level: 2}, atomic,
		ResearchEssence{Kind: EssenceAlchemy}, c4, ResearchEssence{Kind: EssenceFarming})
	shaped(st(ConfigCopier{}), 3, 3,
		steel, c2, steel,
		infused, item.LapisLazuli{}, infused,
		nil, item.Stick{}, nil)
	// The magic sieve: an auto sieve (in any state) with eyes of ender, diamonds
	// and amethyst.
	add(machineItem(KindMagicSieve), []recipe.Item{st(Machine{Kind: KindAutoSieve}).WithValue("variants", true)}, n(item.EnderEye{}, 2), n(item.Diamond{}, 2), n(item.AmethystShard{}, 4))
	add(machineItem(KindMiner), n(iron, 6), n(block.NewHopper(), 1))
	add(machineItem(KindHarvester), n(iron, 5), n(block.NewHopper(), 1), n(block.NewChest(), 1))
	// The drawer: a ring of planks round a chest.
	shaped(st(Drawer{}), 3, 3,
		planks, planks, planks,
		planks, block.NewChest(), planks,
		planks, planks, planks)
	add(st(Barrel{Wood: "oak"}), n(planks, 6), n(stick, 2))
	// The stone barrel; a torch under it is the least heat for it.
	add(st(Barrel{Wood: stoneWood}), n(block.Stone{}, 6), n(stick, 2))
	// Hoppers: the wood hopper from planks and a chest; each hopper and its
	// reverse turn into one another when crafted alone.
	woodHopper, woodReverse := newFlowHopper("wood_hopper", cube.FaceDown), newFlowHopper("wood_hopper_reverse", cube.FaceUp)
	ironReverse := newFlowHopper("iron_hopper_reverse", cube.FaceUp)
	add(st(woodHopper), n(planks, 5), n(block.NewChest(), 1))
	add(st(woodReverse), n(woodHopper, 1))
	add(st(woodHopper), n(woodReverse, 1))
	add(st(ironReverse), n(block.NewHopper(), 1))
	add(st(block.NewHopper()), n(ironReverse, 1))
	add(st(DrawerKey{}), n(item.IronNugget{}, 4), n(stick, 1))
	add(st(DrawerUpgrade{Kind: UpgradeCapacity}), n(iron, 4), n(block.NewChest(), 1))
	add(st(DrawerUpgrade{Kind: UpgradeVoid}), n(item.Flint{}, 4), n(iron, 1))
}

// dyeUnlocks are the results the dyeing research opens: the 16 dyes (made
// from petals) and the blocks that take a colour.
func dyeUnlocks() []string {
	var out []string
	for _, c := range petalColours {
		for _, suffix := range []string{"dye", "stained_glass", "stained_glass_pane", "terracotta", "concrete_powder", "candle"} {
			out = append(out, c+"_"+suffix)
		}
	}
	return out
}
