package skygear

import (
	"fmt"
	"github.com/presentkim/skygear/content/skygear/codex"
	"slices"
	"strconv"
	"strings"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/storage"

	"github.com/presentkim/skygear/content/skygear/conduit"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/decor"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Research: every crafting recipe but baseUnlocks is locked. A research,
// paid whole with items at the research table, opens the recipes of its
// results for everyone on the island (recipes.go). Researches form a tree;
// trees and costs are drafts.

var (
	oakPlanks = block.Planks{Wood: block.OakWood()}
	stick     = item.Stick{}
)

// researchTreeSet fills researchTree before any init function runs.
var researchTreeSet = progress.SetTree(progress.WithEssenceCosts(baseResearchTree))

// baseResearchTree are the researches with their material costs.
var baseResearchTree = []progress.Research{
	{ID: "stick", Cost: []progress.Requirement{{Item: oakPlanks, Count: 4}}, Unlocks: []string{"stick"}},
	{ID: "pebbles", Needs: []string{"stick"}, Cost: []progress.Requirement{{Item: sgitem.Pebble{Kind: sgitem.PebbleStone}, Count: 4}},
		Unlocks: []string{"cobblestone", "cobbled_deepslate", "blackstone", "basalt", "tuff", "calcite", "andesite", "granite", "diorite"}},
	{ID: "wood_tools", Needs: []string{"stick"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 8}, {Item: stick, Count: 4}},
		Unlocks: []string{"wooden_pickaxe", "wooden_axe", "wooden_shovel", "wooden_hoe", "wooden_sword"}},
	{ID: "chest", Needs: []string{"stick"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 32}}, Unlocks: []string{"chest"}},
	{ID: "wood_hopper", Needs: []string{"chest"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 32}},
		Unlocks: []string{"skygear:wood_hopper", "skygear:wood_hopper_reverse"}},
	{ID: "slab", Needs: []string{"wood_tools"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 16}}, Unlocks: []string{"oak_slab"}},
	{ID: "compost", Needs: []string{"slab"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 32}}, Unlocks: []string{"composter"}},
	{ID: "barrel", Needs: []string{"compost"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 32}, {Item: stick, Count: 4}}, Unlocks: []string{"skygear:barrel_oak"}},
	{ID: "furnace", Needs: []string{"pebbles"}, Cost: []progress.Requirement{{Item: block.Cobblestone{}, Count: 16}}, Unlocks: []string{"furnace"}},
	{ID: "stone_tools", Needs: []string{"wood_tools", "pebbles"}, Cost: []progress.Requirement{{Item: block.Cobblestone{}, Count: 16}, {Item: stick, Count: 8}},
		Unlocks: []string{"stone_pickaxe", "stone_axe", "stone_shovel", "stone_hoe", "stone_sword"}},
	{ID: "string", Needs: []string{"stone_tools"}, Cost: []progress.Requirement{{Item: sgitem.BarkFiber{}, Count: 8}}, Unlocks: []string{"string"}},
	{ID: "hammer", Needs: []string{"pebbles"}, Cost: []progress.Requirement{{Item: block.Cobblestone{}, Count: 16}, {Item: stick, Count: 4}}, Unlocks: []string{"skygear:stone_hammer"}},
	{ID: "sieve", Needs: []string{"string", "hammer"}, Cost: []progress.Requirement{{Item: block.String{}, Count: 8}, {Item: oakPlanks, Count: 16}}, Unlocks: []string{"skygear:sieve", "skygear:mesh_string", "skygear:mesh_flint"}},
	{ID: "torch", Needs: []string{"furnace"}, Cost: []progress.Requirement{{Item: item.Charcoal{}, Count: 8}, {Item: stick, Count: 8}}, Unlocks: []string{"torch"}},
	{ID: "woodwork", Needs: []string{"chest"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 64}, {Item: stick, Count: 16}},
		Unlocks: []string{"oak_stairs", "oak_fence", "fence_gate", "wooden_door", "trapdoor", "ladder", "bowl"}},
	{ID: "masonry", Needs: []string{"furnace"}, Cost: []progress.Requirement{{Item: block.Stone{}, Count: 32}, {Item: block.Cobblestone{}, Count: 32}},
		Unlocks: []string{"cobblestone_slab", "stone_stairs", "cobblestone_wall", "normal_stone_slab", "normal_stone_stairs", "stone_bricks"}},
	{ID: "glasswork", Needs: []string{"furnace"}, Cost: []progress.Requirement{{Item: block.Glass{}, Count: 16}}, Unlocks: []string{"glass_pane", "glass_bottle"}},
	{ID: "iron", Needs: []string{"furnace", "sieve"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.OreIron}, Count: 16}}, Unlocks: []string{"iron_ore", "iron_ingot", "iron_block", "iron_nugget", "skygear:mesh_iron"}},
	{ID: "wooden_bucket", Needs: []string{"barrel"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 16}}, Unlocks: []string{"skygear:wooden_bucket"}},
	{ID: "bucket", Needs: []string{"iron", "barrel"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 6}}, Unlocks: []string{"bucket"}},
	{ID: "shears", Needs: []string{"iron", "wood_tools"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 4}}, Unlocks: []string{"shears"}},
	{ID: "stone_barrel", Needs: []string{"barrel", "furnace", "bucket", "torch"}, Cost: []progress.Requirement{{Item: block.Stone{}, Count: 32}}, Unlocks: []string{"skygear:barrel_stone"}},
	{ID: "hopper", Needs: []string{"iron", "chest"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 10}, {Item: block.NewChest(), Count: 4}}, Unlocks: []string{"hopper", "skygear:iron_hopper_reverse"}},
	{ID: "generator", Needs: []string{"iron", "furnace"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}}, Unlocks: []string{"skygear:generator"}},
	{ID: "laboratory", Needs: []string{"generator", "glasswork"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: block.Glass{}, Count: 8}, {Item: item.RedstoneWire{}, Count: 8}},
		Unlocks: []string{"skygear:laboratory"}},
	{ID: "energy_conduit", Needs: []string{"generator"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}}, Unlocks: []string{"skygear:conduit_energy_1", "skygear:conduit_binder_mix"}},
	{ID: "item_conduit", Needs: []string{"hopper"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}}, Unlocks: []string{"skygear:conduit_item_1", "skygear:conduit_binder_mix"}},
	{ID: "fluid_conduit", Needs: []string{"item_conduit", "glasswork"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: block.Glass{}, Count: 8}}, Unlocks: []string{"skygear:conduit_fluid_1", "skygear:conduit_binder_mix"}},
	{ID: "auto_sieve", Needs: []string{"sieve", "hopper", "generator"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: block.String{}, Count: 8}}, Unlocks: []string{"skygear:auto_sieve"}},
	{ID: "iron_tools", Needs: []string{"iron", "stone_tools"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 32}},
		Unlocks: []string{"iron_pickaxe", "iron_axe", "iron_shovel", "iron_hoe", "iron_sword"}},
	{ID: "iron_hammer", Needs: []string{"hammer", "iron"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 8}}, Unlocks: []string{"skygear:hammer"}},
	{ID: "auto_miner", Needs: []string{"iron_tools", "hopper", "generator"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: block.NewHopper(), Count: 2}}, Unlocks: []string{"skygear:auto_miner"}},
	{ID: "crusher", Needs: []string{"hammer", "generator", "auto_miner"}, Cost: []progress.Requirement{{Item: block.Cobblestone{}, Count: 64}, {Item: item.IronIngot{}, Count: 8}}, Unlocks: []string{"skygear:crusher"}},
	{ID: "drawer", Needs: []string{"chest"}, Cost: []progress.Requirement{{Item: oakPlanks, Count: 32}, {Item: block.NewChest(), Count: 2}}, Unlocks: []string{"skygear:drawer", "skygear:drawer_key"}},
	{ID: "drawer_upgrades", Needs: []string{"drawer", "iron"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}},
		Unlocks: []string{"skygear:drawer_upgrade_capacity", "skygear:drawer_upgrade_void"}},
	{ID: "farming", Needs: []string{"wood_tools"}, Cost: []progress.Requirement{{Item: item.Wheat{}, Count: 16}}, Unlocks: []string{"bread", "hay_block"}},
	{ID: "harvester", Needs: []string{"farming", "hopper", "generator"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}}, Unlocks: []string{"skygear:harvester"}},
	{ID: "clay", Needs: []string{"furnace", "barrel"}, Cost: []progress.Requirement{{Item: block.Clay{}, Count: 4}}, Unlocks: []string{"clay", "brick_block"}},
	{ID: "gold", Needs: []string{"iron"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.OreGold}, Count: 16}}, Unlocks: []string{"gold_ore", "gold_ingot", "gold_block", "gold_nugget", "skygear:mesh_gold"}},
	{ID: "mesh_diamond", Needs: []string{"gold"}, Cost: []progress.Requirement{{Item: item.Diamond{}, Count: 4}}, Unlocks: []string{"skygear:mesh_diamond"}},
	{ID: "mesh_netherite", Needs: []string{"mesh_diamond"}, Cost: []progress.Requirement{{Item: item.NetheriteScrap{}, Count: 4}}, Unlocks: []string{"skygear:mesh_netherite", "netherite_ingot"}},
	{ID: "copper", Needs: []string{"furnace", "sieve"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.OreCopper}, Count: 16}}, Unlocks: []string{"copper_ore", "copper_ingot", "copper_block", "copper_nugget"}},
	{ID: "tin", Needs: []string{"copper"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.MetalTin}, Count: 16}}, Unlocks: []string{"skygear:tin_ore", "skygear:metal_dust_tin"}},
	{ID: "lead", Needs: []string{"tin", "iron"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.MetalLead}, Count: 16}}, Unlocks: []string{"skygear:lead_ore", "skygear:metal_dust_lead"}},
	{ID: "silver", Needs: []string{"gold"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.MetalSilver}, Count: 16}}, Unlocks: []string{"skygear:silver_ore", "skygear:metal_dust_silver"}},
	{ID: "nickel", Needs: []string{"gold"}, Cost: []progress.Requirement{{Item: sgitem.OreChunk{Ore: sgitem.MetalNickel}, Count: 16}}, Unlocks: []string{"skygear:nickel_ore", "skygear:metal_dust_nickel"}},
	{ID: "alloy_smelter", Needs: []string{"generator", "tin", "crusher"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: sgitem.Ingot{Metal: sgitem.MetalTin}, Count: 8}}, Unlocks: []string{"skygear:alloy_smelter"}},
	{ID: "energy_cell", Needs: []string{"generator", "lead"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalLead}, Count: 8}, {Item: item.IronIngot{}, Count: 8}}, Unlocks: []string{"skygear:energy_cell"}},
	{ID: "conduit_upgrade_2", Needs: []string{"alloy_smelter", "silver", "energy_conduit", "item_conduit"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalElectrum}, Count: 8}}, Unlocks: []string{"skygear:conduit_energy_2", "skygear:conduit_item_2", "skygear:conduit_fluid_2"}},
	{ID: "conduit_upgrade_3", Needs: []string{"conduit_upgrade_2"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 8}, {Item: item.Diamond{}, Count: 4}}, Unlocks: []string{"skygear:conduit_energy_3", "skygear:conduit_item_3", "skygear:conduit_fluid_3"}},
	{ID: "mob_simulator", Needs: []string{"alloy_smelter", "crusher"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalBronze}, Count: 8}, {Item: sgitem.Essence{Mob: sgitem.MobZombie}, Count: 4}, {Item: sgitem.Essence{Mob: sgitem.MobSkeleton}, Count: 4}}, Unlocks: []string{"skygear:mob_simulator"}},
	{ID: "fluid_tank", Needs: []string{"fluid_conduit", "bucket"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: block.Glass{}, Count: 8}}, Unlocks: []string{"skygear:fluid_tank"}},
	{ID: "crucible", Needs: []string{"stone_barrel", "generator", "fluid_tank", "tin"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: sgitem.Ingot{Metal: sgitem.MetalBronze}, Count: 8}}, Unlocks: []string{"skygear:crucible"}},
	{ID: "lava_generator", Needs: []string{"crucible", "alloy_smelter"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 8}}, Unlocks: []string{"skygear:lava_generator"}},
	{ID: "ore_washer", Needs: []string{"fluid_tank", "alloy_smelter"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}, {Item: sgitem.Ingot{Metal: sgitem.MetalBronze}, Count: 4}}, Unlocks: []string{"skygear:ore_washer"}},
	{ID: "dimension_gate", Needs: []string{"mob_simulator", "lava_generator"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 8}, {Item: sgitem.Essence{Mob: sgitem.MobZombie}, Count: 8}, {Item: sgitem.Essence{Mob: sgitem.MobSkeleton}, Count: 8}}, Unlocks: []string{"skygear:dimension_gate"}},
	{ID: "diamond_tools", Needs: []string{"iron_tools", "mesh_diamond"}, Cost: []progress.Requirement{{Item: item.Diamond{}, Count: 8}},
		Unlocks: []string{"diamond_pickaxe", "diamond_axe", "diamond_shovel", "diamond_hoe", "diamond_sword"}},
	{ID: "storage_controller", Needs: []string{"drawer_upgrades", "item_conduit", "tin"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: sgitem.Ingot{Metal: sgitem.MetalBronze}, Count: 8}}, Unlocks: []string{"skygear:storage_controller", "skygear:storage_link"}},
	{ID: "tinker_table", Needs: []string{"iron_tools", "tin"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 24}, {Item: item.Flint{}, Count: 8}}, Unlocks: []string{"skygear:tinker_table"}},
	{ID: "copper_tools", Needs: []string{"copper", "iron_tools"}, Cost: []progress.Requirement{{Item: item.CopperIngot{}, Count: 12}},
		Unlocks: []string{"copper_pickaxe", "copper_axe", "copper_shovel", "copper_hoe", "copper_sword"}},
	{ID: "copper_armor", Needs: []string{"copper_tools"}, Cost: []progress.Requirement{{Item: item.CopperIngot{}, Count: 12}},
		Unlocks: []string{"copper_helmet", "copper_chestplate", "copper_leggings", "copper_boots"}},
	{ID: "iron_armor", Needs: []string{"iron_tools"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}},
		Unlocks: []string{"iron_helmet", "iron_chestplate", "iron_leggings", "iron_boots"}},
	{ID: "gold_tools", Needs: []string{"gold", "iron_tools"}, Cost: []progress.Requirement{{Item: item.GoldIngot{}, Count: 10}},
		Unlocks: []string{"golden_pickaxe", "golden_axe", "golden_shovel", "golden_hoe", "golden_sword"}},
	{ID: "gold_armor", Needs: []string{"gold_tools"}, Cost: []progress.Requirement{{Item: item.GoldIngot{}, Count: 12}},
		Unlocks: []string{"golden_helmet", "golden_chestplate", "golden_leggings", "golden_boots"}},
	{ID: "diamond_armor", Needs: []string{"diamond_tools", "iron_armor"}, Cost: []progress.Requirement{{Item: item.Diamond{}, Count: 12}},
		Unlocks: []string{"diamond_helmet", "diamond_chestplate", "diamond_leggings", "diamond_boots"}},
	{ID: "archery", Needs: []string{"stone_tools", "string"}, Cost: []progress.Requirement{{Item: block.String{}, Count: 6}, {Item: stick, Count: 6}},
		Unlocks: []string{"bow"}},
	{ID: "ironwork", Needs: []string{"iron_tools", "torch"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}},
		Unlocks: []string{"iron_bars", "iron_chain", "lantern", "flint_and_steel"}},
	{ID: "workbenches", Needs: []string{"iron", "woodwork", "masonry"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 8}, {Item: oakPlanks, Count: 16}, {Item: item.Flint{}, Count: 4}},
		Unlocks: []string{"stonecutter_block", "grindstone", "fletching_table", "loom", "smithing_table"}},
	{ID: "anvil", Needs: []string{"ironwork", "masonry"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 24}},
		Unlocks: []string{"anvil"}},
	{ID: "smelters", Needs: []string{"iron_tools", "furnace"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}, {Item: block.Stone{}, Count: 8}},
		Unlocks: []string{"blast_furnace", "smoker", "campfire"}},
	{ID: "redstone_work", Needs: []string{"iron", "generator"}, Cost: []progress.Requirement{{Item: item.RedstoneWire{}, Count: 16}},
		Unlocks: []string{"redstone", "redstone_torch", "redstone_block", "lever", "noteblock"}},
	{ID: "compression", Needs: []string{"mesh_diamond", "furnace"}, Cost: []progress.Requirement{{Item: item.Coal{}, Count: 32}, {Item: item.LapisLazuli{}, Count: 8}},
		Unlocks: []string{"coal", "coal_block", "lapis_lazuli", "lapis_block", "emerald", "emerald_block", "diamond", "diamond_block"}},
	{ID: "quartz_work", Needs: []string{"masonry", "sieve"}, Cost: []progress.Requirement{{Item: item.NetherQuartz{}, Count: 8}},
		Unlocks: []string{"quartz_block", "quartz_bricks", "quartz_pillar", "chiseled_quartz_block", "quartz_slab", "quartz_stairs"}},
	{ID: "sandstone", Needs: []string{"masonry"}, Cost: []progress.Requirement{{Item: block.Sand{}, Count: 32}},
		Unlocks: []string{"sandstone", "cut_sandstone", "chiseled_sandstone", "sandstone_slab", "sandstone_stairs"}},
	{ID: "crystals", Needs: []string{"glasswork", "sieve"}, Cost: []progress.Requirement{{Item: item.AmethystShard{}, Count: 8}},
		Unlocks: []string{"amethyst_block", "tinted_glass"}},
	{ID: "navigation", Needs: []string{"copper", "gold", "redstone_work"}, Cost: []progress.Requirement{{Item: item.CopperIngot{}, Count: 4}, {Item: item.GoldIngot{}, Count: 4}, {Item: item.RedstoneWire{}, Count: 4}},
		Unlocks: []string{"compass", "clock", "spyglass"}},
	{ID: "bone_craft", Needs: []string{"mob_simulator"}, Cost: []progress.Requirement{{Item: item.Bone{}, Count: 8}},
		Unlocks: []string{"bone_meal", "bone_block"}},
	{ID: "slime_block", Needs: []string{"mob_simulator"}, Cost: []progress.Requirement{{Item: item.Slimeball{}, Count: 9}},
		Unlocks: []string{"slime", "slime_ball"}},
	{ID: "blaze_craft", Needs: []string{"mob_simulator", "mesh_netherite"}, Cost: []progress.Requirement{{Item: item.BlazeRod{}, Count: 4}},
		Unlocks: []string{"blaze_powder", "magma_cream", "magma", "ender_eye", "fire_charge"}},
	{ID: "ender_chest", Needs: []string{"blaze_craft", "diamond_tools"}, Cost: []progress.Requirement{{Item: item.EnderPearl{}, Count: 4}, {Item: block.Obsidian{}, Count: 4}},
		Unlocks: []string{"ender_chest"}},
	{ID: "electric_furnace", Needs: []string{"generator", "furnace"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}},
		Unlocks: []string{"skygear:electric_furnace"}},
	{ID: "item_vacuum", Needs: []string{"generator", "hopper"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}},
		Unlocks: []string{"skygear:item_vacuum"}},
	{ID: "tree_feller", Needs: []string{"generator", "iron_tools"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 12}},
		Unlocks: []string{"skygear:tree_feller"}},
	{ID: "auto_crafter", Needs: []string{"generator", "hopper", "redstone_work"}, Cost: []progress.Requirement{{Item: item.IronIngot{}, Count: 16}, {Item: item.RedstoneWire{}, Count: 8}},
		Unlocks: []string{"skygear:auto_crafter"}},
	{ID: "magic_botany", Needs: []string{"dimension_gate"}, Cost: []progress.Requirement{{Item: farm.PlantSeed{Seed: "flower_red"}, Count: 1}, {Item: farm.PlantSeed{Seed: "flower_yellow"}, Count: 1}, {Item: farm.PlantSeed{Seed: "flower_blue"}, Count: 1}}, Unlocks: []string{"skygear:seed_ore_iron", "skygear:seed_ore_gold", "skygear:seed_ore_copper", "skygear:seed_ore_tin", "skygear:seed_ore_lead", "skygear:seed_ore_silver", "skygear:seed_ore_nickel"}},
	{ID: "magic_gems", Needs: []string{"magic_botany"}, Cost: []progress.Requirement{{Item: farm.PlantSeed{Seed: "ore_gold"}, Count: 2}}, Unlocks: []string{"skygear:seed_gem_lapis", "skygear:seed_gem_redstone", "skygear:seed_gem_quartz", "skygear:seed_gem_coal", "skygear:seed_gem_diamond", "skygear:seed_gem_emerald"}},
	{ID: "magic_essence", Needs: []string{"magic_gems", "blaze_craft"}, Cost: []progress.Requirement{{Item: farm.PlantSeed{Seed: "gem_redstone"}, Count: 2}}, Unlocks: []string{"skygear:seed_ess_zombie", "skygear:seed_ess_skeleton", "skygear:seed_ess_creeper", "skygear:seed_ess_spider", "skygear:seed_ess_slime", "skygear:seed_ess_enderman", "skygear:seed_ess_blaze"}},
	{ID: "magic_sieve", Needs: []string{"auto_sieve", "mob_simulator", "magic_botany"}, Cost: []progress.Requirement{{Item: item.Diamond{}, Count: 8}, {Item: item.EnderPearl{}, Count: 8}},
		Unlocks: []string{"skygear:magic_sieve"}},
	{ID: "machine_tier_1", Needs: []string{"auto_crafter", "energy_cell", "conduit_upgrade_2"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 16}, {Item: sgitem.Ingot{Metal: sgitem.MetalElectrum}, Count: 16}, {Item: item.Diamond{}, Count: 4}},
		Unlocks: []string{"skygear:circuit_1", "skygear:circuit_2", "skygear:kit_1"}},
	{ID: "machine_tier_2", Needs: []string{"machine_tier_1", "dimension_gate", "lava_generator"}, Cost: []progress.Requirement{{Item: item.Diamond{}, Count: 16}, {Item: sgitem.Ingot{Metal: sgitem.MetalInvar}, Count: 16}},
		Unlocks: []string{"skygear:circuit_3", "skygear:kit_2"}},
	{ID: "machine_tier_3", Needs: []string{"machine_tier_2", "magic_essence", "magic_sieve"}, Cost: []progress.Requirement{{Item: item.NetheriteIngot{}, Count: 2}},
		Unlocks: []string{"skygear:circuit_4", "skygear:kit_3"}},
	{ID: "config_copier", Needs: []string{"machine_tier_1", "item_conduit", "auto_crafter"}, Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 8}, {Item: item.LapisLazuli{}, Count: 8}},
		Unlocks: []string{"skygear:config_copier"}},
	{ID: "dyeing", Needs: []string{"magic_botany", "glasswork"}, Cost: []progress.Requirement{{Item: sgitem.Petal{Colour: "red"}, Count: 4}, {Item: sgitem.Petal{Colour: "yellow"}, Count: 4}, {Item: sgitem.Petal{Colour: "blue"}, Count: 4}},
		Unlocks: dyeUnlocks()},
	{ID: "harvest", Needs: []string{"farming"}, Cost: []progress.Requirement{{Item: item.MelonSlice{}, Count: 16}, {Item: block.Pumpkin{}, Count: 4}},
		Unlocks: []string{"melon_block", "pumpkin_pie", "melon_seeds", "pumpkin_seeds"}},
	{ID: "golden_food", Needs: []string{"gold", "harvest"}, Cost: []progress.Requirement{{Item: item.GoldIngot{}, Count: 8}, {Item: block.Carrot{}, Count: 4}},
		Unlocks: []string{"golden_apple", "golden_carrot", "glistering_melon_slice"}},
	{ID: "project_industry", Icon: "skygear:storage_controller", Needs: []string{"conduit_upgrade_3", "storage_controller", "auto_crafter", "lava_generator"},
		Cost: []progress.Requirement{{Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 32}, {Item: sgitem.Ingot{Metal: sgitem.MetalElectrum}, Count: 32}, {Item: sgitem.Ingot{Metal: sgitem.MetalInvar}, Count: 32}, {Item: sgitem.Ingot{Metal: sgitem.MetalBronze}, Count: 32}, {Item: item.Diamond{}, Count: 16}}},
	{ID: "project_hunter", Icon: "diamond_sword", Needs: []string{"dimension_gate", "tinker_table", "diamond_armor", "blaze_craft"},
		Cost: []progress.Requirement{{Item: item.BlazeRod{}, Count: 16}, {Item: item.EnderPearl{}, Count: 16}, {Item: item.Slimeball{}, Count: 16}, {Item: item.SpiderEye{}, Count: 16}, {Item: item.Bone{}, Count: 32}}},
	{ID: "project_botanist", Icon: "skygear:petal_red", Needs: []string{"magic_essence", "dyeing"},
		Cost: []progress.Requirement{{Item: farm.PlantSeed{Seed: "ess_enderman"}, Count: 2}, {Item: farm.PlantSeed{Seed: "ess_blaze"}, Count: 2}, {Item: farm.PlantSeed{Seed: "gem_diamond"}, Count: 2}, {Item: farm.PlantSeed{Seed: "gem_emerald"}, Count: 2}, {Item: sgitem.Essence{Mob: sgitem.MobEnder}, Count: 8}, {Item: sgitem.Essence{Mob: sgitem.MobBlaze}, Count: 8}}},
	{ID: "project_dragon", Needs: []string{"project_industry", "project_hunter", "project_botanist"},
		Cost:    []progress.Requirement{{Item: item.Diamond{}, Count: 32}, {Item: item.Emerald{}, Count: 32}, {Item: item.NetheriteIngot{}, Count: 4}, {Item: sgitem.Ingot{Metal: sgitem.MetalSteel}, Count: 32}},
		Unlocks: []string{"dragon_egg"}},
	{ID: "kelp", Needs: []string{"furnace"}, Cost: []progress.Requirement{{Item: item.DriedKelp{}, Count: 16}}, Unlocks: []string{"dried_kelp_block"}},
	// The fairy magic (fairymagic.go): dust from the nightly fairy and a
	// research table; it gives the board at once.
	{ID: "fairy_magic", Cost: []progress.Requirement{{Item: sgitem.FairyDust{}, Count: 2}, {Item: progress.ResearchTable{}, Count: 1}},
		Unlocks: []string{"skygear:fairy_emerald", "skygear:fairy_board"}},
	// Fairy time (fairytime.go): dust and bone meal for the things that
	// cut waiting short.
	{ID: "fairy_time", Needs: []string{"fairy_magic"}, Cost: []progress.Requirement{{Item: sgitem.FairyDust{}, Count: 4}, {Item: item.BoneMeal{}, Count: 8}},
		Unlocks: []string{"skygear:fairy_bone_meal", "skygear:fairy_hourglass"}},
}

// researchIcon is a research's form icon: its first result.
func researchIcon(r *progress.Research) string {
	if len(r.Unlocks) == 0 {
		if it, ok := world.ItemByName(naming.UnlockName(r.Icon), 0); ok {
			return formicon.Of(item.NewStack(it, 1))
		}
		return formicon.Mission
	}
	if it, ok := world.ItemByName(naming.UnlockName(r.Unlocks[0]), 0); ok {
		return formicon.Of(item.NewStack(it, 1))
	}
	return formicon.Mission
}

func (s *session) showResearch(p *player.Player) {
	achievement.Count(p, "research_open", 1)
	s.showResearchSplit(p, researchView{})
}

// researchView is what the research screen shows: the tab (0 to do, 1 done),
// the page of the grid and the chosen research ("" the first of the tab);
// help shows the explanation as the text of the form.
type researchView struct {
	tab, page int
	sel       string
}

// researchKind is how a research stands.
type researchKind int

const (
	researchTodo researchKind = iota
	researchPreview
	researchDone
)

// researchEntry is a research with how it stands.
type researchEntry struct {
	r    *progress.Research
	kind researchKind
}

// researchEntries are the researches of the tabs: the ones to do (those that
// can be done now first, then the grey ones a step from open) and the done
// ones. A research is hidden until everything before it is done, so the tree
// is found step by step. openN is how many are to do without the grey ones.
func (s *session) researchEntries() (todo, done []researchEntry, openN int) {
	var previews []researchEntry
	for i := range progress.Tree {
		r := &progress.Tree[i]
		switch {
		case progress.Researched(s.Session, r.ID):
			done = append(done, researchEntry{r, researchDone})
		case progress.Open(s.Session, r):
			openN++
			todo = append(todo, researchEntry{r, researchTodo})
		default:
			if _, ok := progress.Blocker(s.Session, r); ok {
				previews = append(previews, researchEntry{r, researchPreview})
			}
		}
	}
	for _, r := range progress.LocalOf(s.Session).Copies {
		if progress.Researched(s.Session, r.ID) {
			done = append(done, researchEntry{r, researchDone})
		} else {
			openN++
			todo = append(todo, researchEntry{r, researchTodo})
		}
	}
	return append(todo, previews...), done, openN
}

// researchSlot is the grid slot of a research. The tooltip of a research to
// do is its name in yellow, in green when p has all it spends and can do it
// now. p may be nil.
func (s *session) researchSlot(p *player.Player, e researchEntry, show func(p *player.Player, tx *world.Tx)) ui.ItemSlot {
	r := e.r
	switch e.kind {
	case researchDone:
		// The done tab holds only done researches: no mark.
		return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), "§7", r), Run: show}
	case researchTodo:
		colour := "§e"
		if p != nil && len(progress.Missing(s.Session, p, r)) == 0 {
			colour = "§a"
		}
		return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), colour, r), Run: show}
	}
	return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), "§8", r), Run: show}
}

// showResearchSplit shows the research screen (ui.SplitMenu): on the left
// the tabs, the grid of researches and the page buttons, on the right the
// research chosen. Choosing a research, a tab or a page shows the screen
// again.
func (s *session) showResearchSplit(p *player.Player, v researchView) {
	lang := i18n.Of(p)
	todo, done, _ := s.researchEntries()
	list := todo
	if v.tab == 1 {
		list = done
	}
	idx := -1
	for i, e := range list {
		if e.r.ID == v.sel {
			idx = i
		}
	}
	pages := max(1, (len(list)+ui.SplitCells-1)/ui.SplitCells)
	switch {
	case idx >= 0:
		v.page = idx / ui.SplitCells
	case len(list) > 0:
		v.page = max(0, min(v.page, pages-1))
		idx = v.page * ui.SplitCells
		v.sel = list[idx].r.ID
	default:
		v.sel = ""
	}
	pick := func(tab, page int, sel string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) {
			s.showResearchSplit(p, researchView{tab: tab, page: page, sel: sel})
		}
	}
	var slots []ui.ItemSlot
	for i := v.page * ui.SplitCells; i < min(len(list), (v.page+1)*ui.SplitCells); i++ {
		e := list[i]
		sl := s.researchSlot(p, e, pick(v.tab, v.page, e.r.ID))
		if e.r.ID == v.sel {
			sl.Selected, sl.Run = true, nil
		}
		slots = append(slots, sl)
	}
	tab := func(key string, icon string, t int) ui.Option {
		o := ui.Tab(i18n.T(lang, key), icon, v.tab == t, pick(t, 0, ""))
		return s.Opt(o.Label, o.Icon, o.Run)
	}
	sp := ui.Split{
		Tabs: []ui.Option{
			tab("researchtab.todo", formicon.Mission, 0),
			tab("researchtab.done", formicon.Done, 1),
		},
		Slots: slots,
		Page:  v.page, Pages: pages,
		Turn: func(p *player.Player, tx *world.Tx, to int) {
			if tx.World() == s.World {
				s.showResearchSplit(p, researchView{tab: v.tab, page: to})
			}
		},
	}
	tip := i18n.T(lang, "researchlist.tip")
	if idx >= 0 {
		again := func(p *player.Player) { s.showResearchSplit(p, v) }
		finished := func(p *player.Player) { s.showResearchSplit(p, researchView{tab: v.tab, page: v.page}) }
		goTo := func(p *player.Player, r *progress.Research) { s.showResearchAt(p, r) }
		d, rows, opts := s.researchNodeWith(p, list[idx].r, again, finished, goTo)
		sp.Detail, sp.Rows, sp.Actions = d, rows, opts
	}
	// the help is the button's tooltip; pressing it keeps the screen
	sp.Actions = append(sp.Actions, s.Opt(ui.Help(tip, nil).Label, "", func(p *player.Player, _ *world.Tx) { s.showResearchSplit(p, v) }), s.BackOption(lang))
	ui.SplitMenu(p, i18n.T(lang, "researchlist.title"), sp)
}

// showResearchAt shows the research screen with a research chosen, on the tab
// it belongs to.
func (s *session) showResearchAt(p *player.Player, r *progress.Research) {
	v := researchView{sel: r.ID}
	if progress.Researched(s.Session, r.ID) {
		v.tab = 1
	}
	s.showResearchSplit(p, v)
}

// researchSlots builds the two sections of the research list: the slots of the
// researches to do and of the done ones, with how many are to do (without the
// grey ones) and done. p may be nil.
func (s *session) researchSlots(p *player.Player) (todo, done []ui.ItemSlot, todoN, doneN int) {
	todoE, doneE, openN := s.researchEntries()
	for _, e := range todoE {
		r := e.r
		todo = append(todo, s.researchSlot(p, e, func(p *player.Player, _ *world.Tx) { s.showResearchAt(p, r) }))
	}
	for _, e := range doneE {
		r := e.r
		done = append(done, s.researchSlot(p, e, func(p *player.Player, _ *world.Tx) { s.showResearchAt(p, r) }))
	}
	return todo, done, openN, len(doneE)
}

// researchTip is the hover text of a research in the list: its name in
// colour and the short description.
func researchTip(lang, colour string, r *progress.Research) string {
	tip := colour + r.Title(lang)
	if short := progress.Text(lang, r.ID).Short; short != "" {
		tip += "\n§7" + short
	}
	return tip
}

// researchNode builds the screen of a research with a way back to the
// research screen. p may be nil (the owned counts are then left out).
func (s *session) researchNode(p *player.Player, r *progress.Research) (ui.Detail, []ui.SlotRow, []ui.Option) {
	d, rows, opts := s.researchNodeWith(p, r,
		func(p *player.Player) { s.showResearchAt(p, r) },
		func(p *player.Player) { s.showResearch(p) },
		func(p *player.Player, n *progress.Research) { s.showResearchAt(p, n) })
	back := ui.Back(i18n.T(i18n.Of(p), "researchnode.back"), func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
	return d, rows, append(opts, s.Opt(back.Label, back.Icon, back.Run))
}

// researchNodeWith builds the detail of a research: the card, the text, the
// materials and the recipes it opens, and the buttons to do it and to go to
// what it needs. redo shows the screen again, finished shows it after the
// research was done, goTo shows another research. p may be nil.
func (s *session) researchNodeWith(p *player.Player, r *progress.Research, redo, finished func(p *player.Player), goTo func(p *player.Player, r *progress.Research)) (ui.Detail, []ui.SlotRow, []ui.Option) {
	txt := progress.Text(i18n.Of(p), r.ID)
	done := progress.Researched(s.Session, r.ID)
	var b strings.Builder
	if txt.About != "" {
		fmt.Fprintf(&b, "§f%s\n", txt.About)
	}
	d := ui.Detail{Icon: researchIcon(r), Name: "§6" + r.Title(i18n.Of(p))}
	var opts []ui.Option
	switch {
	case done:
		d.Status = i18n.T(i18n.Of(p), "researchnode.complete")
	case progress.Open(s.Session, r):
		opts = append(opts, s.Opt(i18n.T(i18n.Of(p), "researchnode.start"), formicon.Mission, func(p *player.Player, tx *world.Tx) {
			if r.IsCopy() {
				progress.LocalOf(s.Session).CopiesDone[r.ID] = true
				finished(p)
				return
			}
			missing, ok := progress.Do(s.Session, p, tx, r)
			switch {
			case ok:
				ui.Flash(p, "%s", i18n.T(i18n.Of(p), "researchnode.finished"))
				finished(p)
				return
			case len(missing) > 0:
				var names []string
				for _, m := range missing {
					names = append(names, island.CostText(i18n.Of(p), m))
				}
				ui.Notify(p, i18n.T(i18n.Of(p), "researchnode.missing"), strings.Join(names, ", "))
			}
			redo(p)
		}))
	}
	// What is still to do before it: a button that goes to that research.
	for _, n := range r.Needs {
		if need, ok := progress.ByID(n); ok && !done && !progress.Researched(s.Session, n) {
			opts = append(opts, s.Opt(fmt.Sprintf(i18n.T(i18n.Of(p), "researchneeds.button"), need.Title(i18n.Of(p))), researchIcon(need), func(p *player.Player, _ *world.Tx) { goTo(p, need) }))
		}
	}
	d.Body = b.String()
	// A material or recipe slot opens the item's codex page, whose back
	// button comes back here; an item the codex does not know cannot be
	// pressed.
	toCodex := func(st item.Stack, tip string) ui.ItemSlot {
		if _, ok := codex.ItemByKey(naming.ItemKey(st.Item())); !ok {
			return ui.ItemSlot{Icon: formicon.Of(st), Tip: tip + "\n§8" + i18n.T(i18n.Of(p), "codex.fromResearch.none")}
		}
		return ui.ItemSlot{Icon: formicon.Of(st), Tip: tip + "\n§8" + i18n.T(i18n.Of(p), "codex.fromResearch.go"), Run: func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.World {
				codex.From(s.Session, p, redo, st)
			}
		}}
	}
	// The materials: the count shown is what the research spends (green
	// once p has enough, or the research is done); the tooltip says what
	// p has.
	var cost []ui.ItemSlot
	for _, c := range r.Cost {
		need := progress.Need(s.Session, r, c)
		st := item.NewStack(c.Item, 1)
		mark, tip := "§f", fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.need"), naming.UIItemName(i18n.Of(p), st), need)
		switch {
		case done:
			mark, tip = "§a", fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.uses"), naming.UIItemName(i18n.Of(p), st), c.Count)
			need = c.Count
		case p != nil:
			have := core.CountPlain(p.Inventory(), c.Item)
			if have >= need {
				mark = "§a"
			}
			tip += fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.have"), mark, have)
		}
		sl := toCodex(st, tip)
		sl.Count = mark + strconv.Itoa(need)
		cost = append(cost, sl)
	}
	// The recipes it opens.
	var opens []ui.ItemSlot
	for _, n := range r.Unlocks {
		it, ok := world.ItemByName(naming.UnlockName(n), 0)
		if !ok {
			continue
		}
		st := item.NewStack(it, 1)
		tip := decor.TipName(i18n.Of(p), st) + i18n.T(i18n.Of(p), "researchnode.unlocks")
		c := progress.RewardCount(r, n)
		if c > 0 {
			tip += "\n" + fmt.Sprintf(i18n.T(i18n.Of(p), "researchreward.tip"), c)
		}
		sl := toCodex(st, tip)
		if c > 0 {
			sl.Count = "§e+" + strconv.Itoa(c)
		}
		opens = append(opens, sl)
	}
	rows := []ui.SlotRow{{Heading: i18n.T(i18n.Of(p), "researchnode.materials"), Slots: cost}, {Heading: i18n.T(i18n.Of(p), "researchnode.rewards"), Slots: opens}}
	return d, rows, opts
}

func init() {
	gameRecipes()
	progress.ShowScreen = func(c *core.Session, p *player.Player) {
		if s, ok := activeGame.sessionOf(c.World); ok {
			s.showResearch(p)
		}
	}
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
	// The hammers: two of the material and two sticks in a diagonal
	// pattern, the stone hammer of cobblestone, the iron hammer of iron.
	shaped(st(sgitem.Hammer{Stone: true}), 3, 3,
		nil, cobble, nil,
		nil, stick, cobble,
		stick, nil, nil)
	shaped(st(sgitem.Hammer{}), 3, 3,
		nil, iron, nil,
		nil, stick, iron,
		stick, nil, nil)
	// The wooden bucket: three planks in a V, like the iron bucket.
	shaped(st(sgitem.WoodenBucket{}), 3, 2,
		planks, nil, planks,
		nil, planks, nil)
	// The sieve: a row of planks on two stick legs. The mesh is made and
	// fitted separately (sieve.go).
	shaped(machine.Item(machine.KindSieve), 3, 2,
		planks, planks, planks,
		stick, nil, stick)
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
	add(st(farm.Barrel{Wood: "oak"}), n(planks, 6), n(stick, 2))
	// The stone barrel; a torch under it is the least heat for it.
	add(st(farm.Barrel{Wood: farm.StoneWood}), n(block.Stone{}, 6), n(stick, 2))
	// Hoppers: the wood hopper from planks and a chest; each hopper and its
	// reverse turn into one another when crafted alone.
	woodHopper, woodReverse := hopper.NewFlowHopper("wood_hopper", cube.FaceDown), hopper.NewFlowHopper("wood_hopper_reverse", cube.FaceUp)
	ironReverse := hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp)
	add(st(woodHopper), n(planks, 5), n(block.NewChest(), 1))
	add(st(woodReverse), n(woodHopper, 1))
	add(st(woodHopper), n(woodReverse, 1))
	add(st(ironReverse), n(block.NewHopper(), 1))
	add(st(block.NewHopper()), n(ironReverse, 1))
	add(st(sgitem.DrawerKey{}), n(item.IronNugget{}, 4), n(stick, 1))
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
	shaped(item.NewStack(stick, 16), 1, 2, log, log)
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
