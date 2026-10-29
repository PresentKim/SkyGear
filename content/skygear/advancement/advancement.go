// Package advancement defines the advancement tree: the
// advancements, their chapters and the board layout of each chapter. The game
// package grants them and draws the board.
package advancement

import (
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
)

// Advancement is one step of the advancement tree. It is done by having an
// item, by a research or by an event the game reports (advance).
type Advancement struct {
	ID string
	// NameKey and HintKey are the catalog keys of its name and of the hint
	// that says how to do it, one HUD row per line (Title, HintText).
	NameKey, HintKey string
	// Chapter is the chapter of the advancement screen it is in.
	Chapter Chapter
	// Parents come before it: it is open once they are all done (none for
	// the first). Doing it first is fine; they stay as they are.
	Parents []string
	// Item is the name of an item to have in the inventory
	// (naming.UnlockName form), or empty.
	Item string
	// Research is a research to complete, or empty.
	Research string
	// Icon is the item name shown in the advancement screen; Item when
	// empty.
	Icon string
	// Event marks an advancement the game reports itself (advance).
	Event bool
	// Guide is the research the quest panel leads to for an advancement
	// that names none (an event, or an item no research opens).
	Guide string
}

// Advancements done by events (advance).
const (
	GrowTree = "grow_tree"
	Sift     = "sift"
	Machine  = "machine"
	Chunk    = "chunk"
	// Wave1 and Wave5 are done by clearing levels 1 and 5 of the gate.
	Wave1 = "wave_1"
	// ToolMod and ToolLevel are done by putting a modifier on a tool and
	// by a tool reaching level 1.
	ToolMod = "tool_modifier"
	// Bloom is done by harvesting a magic plant.
	Bloom     = "first_bloom"
	ToolLevel = "tool_level"
	Wave5     = "wave_5"

	Kill       = "first_kill"
	Wave3      = "wave_3"
	Wave8      = "wave_8"
	ToolMade   = "tool_made"
	Tool5      = "tool_level_5"
	OreHarvest = "ore_harvest"
	GemHarvest = "gem_harvest"
	EssHarvest = "ess_harvest"
	Chunks8    = "chunks_8"
	Chunks16   = "chunks_16"
	// EssenceResearch is done by a research that cost research essences.
	EssenceResearch = "essence_research"
	// Dragon is done by placing the dragon egg.
	Dragon = "dragon_egg"
	// MachineTier is done by putting a kit on a machine, Stats by opening
	// the production stats, FairyGift by a gift fairy's gift, Favourite by a
	// favourite in the codex and Lost by taking an item back from the lost
	// and found.
	MachineTier = "machine_tier"
	Stats       = "stats"
	FairyGift   = "fairy_gift"
	Favourite   = "codex_favourite"
	Lost        = "lost_found"
)

// Tree is every advancement in the order of the game's flow; the
// quest panel follows this order, so the main path comes first. The screen
// shows each chapter's advancements in this order too.
var Tree = []Advancement{
	{ID: "wood", Chapter: ChapterStart, NameKey: "advancement.wood.name", HintKey: "advancement.wood.hint", Item: "oak_log"},
	{ID: "planks", Chapter: ChapterStart, Parents: []string{"wood"}, NameKey: "advancement.planks.name", HintKey: "advancement.planks.hint", Item: "oak_planks"},
	{ID: "crafting_table", Chapter: ChapterStart, Parents: []string{"planks"}, NameKey: "advancement.crafting_table.name", HintKey: "advancement.crafting_table.hint", Item: "crafting_table"},
	{ID: "research_table", Chapter: ChapterStart, Parents: []string{"crafting_table"}, NameKey: "advancement.research_table.name", HintKey: "advancement.research_table.hint", Item: "skygear:research_table"},
	{ID: "first_research", Chapter: ChapterStart, Parents: []string{"research_table"}, NameKey: "advancement.first_research.name", HintKey: "advancement.first_research.hint", Research: "stick", Icon: "stick"},
	{ID: "pebble", Chapter: ChapterStart, Parents: []string{"first_research"}, NameKey: "advancement.pebble.name", HintKey: "advancement.pebble.hint", Item: "skygear:pebble_stone", Guide: "stick"},
	{ID: "cobble", Chapter: ChapterStart, Parents: []string{"pebble"}, NameKey: "advancement.cobble.name", HintKey: "advancement.cobble.hint", Item: "cobblestone"},
	{ID: "wood_tools", Chapter: ChapterStart, Parents: []string{"first_research"}, NameKey: "advancement.wood_tools.name", HintKey: "advancement.wood_tools.hint", Item: "wooden_pickaxe"},
	{ID: "chest", Chapter: ChapterStart, Parents: []string{"first_research"}, NameKey: "advancement.chest.name", HintKey: "advancement.chest.hint", Item: "chest"},
	{ID: "stone_axe", Chapter: ChapterStart, Parents: []string{"cobble"}, NameKey: "advancement.stone_axe.name", HintKey: "advancement.stone_axe.hint", Item: "stone_axe"},
	{ID: "fiber", Chapter: ChapterStart, Parents: []string{"stone_axe"}, NameKey: "advancement.fiber.name", HintKey: "advancement.fiber.hint", Item: "skygear:bark_fiber", Guide: "stone_tools"},
	{ID: "string", Chapter: ChapterStart, Parents: []string{"fiber"}, NameKey: "advancement.string.name", HintKey: "advancement.string.hint", Item: "string"},
	{ID: "hammer", Chapter: ChapterStart, Parents: []string{"cobble"}, NameKey: "advancement.hammer.name", HintKey: "advancement.hammer.hint", Item: "skygear:stone_hammer"},
	{ID: "gravel", Chapter: ChapterStart, Parents: []string{"hammer"}, NameKey: "advancement.gravel.name", HintKey: "advancement.gravel.hint", Item: "gravel", Guide: "hammer"},
	{ID: "sand", Chapter: ChapterStart, Parents: []string{"gravel"}, NameKey: "advancement.sand.name", HintKey: "advancement.sand.hint", Item: "sand", Guide: "hammer"},
	{ID: "sieve", Chapter: ChapterSieve, Parents: []string{"string"}, NameKey: "advancement.sieve.name", HintKey: "advancement.sieve.hint", Item: "skygear:sieve"},
	{ID: "mesh_string", Chapter: ChapterSieve, Parents: []string{"sieve"}, NameKey: "advancement.mesh_string.name", HintKey: "advancement.mesh_string.hint", Item: "skygear:mesh_string"},
	{ID: Sift, Chapter: ChapterSieve, Parents: []string{"mesh_string", "gravel"}, Event: true, NameKey: "advancement.sift.name", HintKey: "advancement.sift.hint", Icon: "gravel", Guide: "sieve"},
	{ID: "mesh_flint", Chapter: ChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_flint.name", HintKey: "advancement.mesh_flint.hint", Item: "skygear:mesh_flint"},
	{ID: "ore_block", Chapter: ChapterSieve, NameKey: "advancement.ore_block.name", Parents: []string{Sift}, HintKey: "advancement.ore_block.hint", Item: "iron_ore"},
	{ID: "seeds", Chapter: ChapterFarming, Parents: []string{Sift}, NameKey: "advancement.seeds.name", HintKey: "advancement.seeds.hint", Item: "wheat_seeds", Guide: "sieve"},
	{ID: "wheat", Chapter: ChapterFarming, Parents: []string{"seeds"}, NameKey: "advancement.wheat.name", HintKey: "advancement.wheat.hint", Item: "wheat", Guide: "wood_tools"},
	{ID: "bread", Chapter: ChapterFarming, Parents: []string{"wheat"}, NameKey: "advancement.bread.name", HintKey: "advancement.bread.hint", Item: "bread"},
	{ID: "furnace", Chapter: ChapterStart, Parents: []string{"cobble"}, NameKey: "advancement.furnace.name", HintKey: "advancement.furnace.hint", Item: "furnace"},
	{ID: "torch", Chapter: ChapterStart, Parents: []string{"furnace"}, NameKey: "advancement.torch.name", HintKey: "advancement.torch.hint", Item: "torch"},
	{ID: "composter", Chapter: ChapterFarming, Parents: []string{"wood_tools"}, NameKey: "advancement.composter.name", HintKey: "advancement.composter.hint", Item: "composter"},
	{ID: "barrel", Chapter: ChapterFarming, Parents: []string{"composter"}, NameKey: "advancement.barrel.name", HintKey: "advancement.barrel.hint", Item: "skygear:barrel_oak"},
	{ID: GrowTree, Chapter: ChapterStart, Parents: []string{"wood"}, Event: true, NameKey: "advancement.grow_tree.name", HintKey: "advancement.grow_tree.hint", Icon: "oak_sapling"},
	{ID: "dust", Chapter: ChapterStart, Parents: []string{"sand"}, NameKey: "advancement.dust.name", HintKey: "advancement.dust.hint", Item: "skygear:dust", Guide: "hammer"},
	{ID: "clay", Chapter: ChapterFarming, Parents: []string{"barrel", "dust"}, NameKey: "advancement.clay.name", HintKey: "advancement.clay.hint", Item: "clay", Icon: "clay"},
	{ID: "wooden_bucket", Chapter: ChapterFarming, Parents: []string{"barrel"}, NameKey: "advancement.wooden_bucket.name", HintKey: "advancement.wooden_bucket.hint", Item: "skygear:wooden_bucket"},
	{ID: "glass", Chapter: ChapterStart, Parents: []string{"furnace", "sand"}, NameKey: "advancement.glass.name", HintKey: "advancement.glass.hint", Item: "glass", Guide: "furnace"},
	{ID: "iron", Chapter: ChapterSieve, Parents: []string{"ore_block", "furnace"}, NameKey: "advancement.iron.name", HintKey: "advancement.iron.hint", Item: "iron_ingot"},
	{ID: "mesh_iron", Chapter: ChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_iron.name", HintKey: "advancement.mesh_iron.hint", Item: "skygear:mesh_iron"},
	{ID: "ore_dust", Chapter: ChapterSieve, Parents: []string{"ore_block", "hammer"}, NameKey: "advancement.ore_dust.name", HintKey: "advancement.ore_dust.hint", Item: "skygear:metal_dust_iron", Guide: "iron"},
	{ID: "diamond", Chapter: ChapterSieve, Parents: []string{"mesh_iron", Sift}, NameKey: "advancement.diamond.name", HintKey: "advancement.diamond.hint", Item: "diamond"},
	{ID: "bucket", Chapter: ChapterFarming, Parents: []string{"iron"}, NameKey: "advancement.bucket.name", HintKey: "advancement.bucket.hint", Item: "bucket"},
	{ID: "stone_barrel", Chapter: ChapterFarming, Parents: []string{"barrel", "furnace"}, NameKey: "advancement.stone_barrel.name", HintKey: "advancement.stone_barrel.hint", Item: "skygear:barrel_stone", Icon: "skygear:barrel_stone"},
	{ID: "lava", Chapter: ChapterFarming, Parents: []string{"stone_barrel", "bucket"}, NameKey: "advancement.lava.name", HintKey: "advancement.lava.hint", Item: "lava_bucket", Icon: "lava_bucket", Guide: "stone_barrel"},
	{ID: "generator", Chapter: ChapterAutomation, Parents: []string{"iron"}, NameKey: "advancement.generator.name", HintKey: "advancement.generator.hint", Research: "generator", Icon: "skygear:generator"},
	{ID: Machine, Chapter: ChapterAutomation, Parents: []string{"generator"}, Event: true, NameKey: "advancement.machine.name", HintKey: "advancement.machine.hint", Icon: "skygear:crusher", Guide: "generator"},
	{ID: "laboratory", Chapter: ChapterLab, Parents: []string{"generator", "glass"}, NameKey: "advancement.laboratory.name", HintKey: "advancement.laboratory.hint", Item: "skygear:laboratory"},
	{ID: "lab_engineering", Chapter: ChapterLab, Parents: []string{"laboratory"}, NameKey: "advancement.lab_engineering.name", HintKey: "advancement.lab_engineering.hint", Item: "skygear:research_essence_engineering", Guide: "laboratory"},
	{ID: EssenceResearch, Chapter: ChapterLab, Parents: []string{"lab_engineering"}, Event: true, NameKey: "advancement.essence_research.name", HintKey: "advancement.essence_research.hint", Icon: "skygear:research_essence_engineering", Guide: "laboratory"},
	{ID: "hopper", Chapter: ChapterLogistics, Parents: []string{"iron", "chest"}, NameKey: "advancement.hopper.name", HintKey: "advancement.hopper.hint", Research: "hopper", Icon: "hopper"},
	{ID: "energy_conduit", Chapter: ChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.energy_conduit.name", HintKey: "advancement.energy_conduit.hint", Research: "energy_conduit", Icon: "skygear:conduit_energy_1"},
	{ID: "item_conduit", Chapter: ChapterLogistics, Parents: []string{"hopper"}, NameKey: "advancement.item_conduit.name", HintKey: "advancement.item_conduit.hint", Research: "item_conduit", Icon: "skygear:conduit_item_1"},
	{ID: "iron_pickaxe", Chapter: ChapterGear, Parents: []string{"iron"}, NameKey: "advancement.iron_pickaxe.name", HintKey: "advancement.iron_pickaxe.hint", Item: "iron_pickaxe"},
	{ID: "auto_miner", Chapter: ChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.auto_miner.name", HintKey: "advancement.auto_miner.hint", Research: "auto_miner", Icon: "skygear:auto_miner"},
	{ID: "crusher", Chapter: ChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.crusher.name", HintKey: "advancement.crusher.hint", Research: "crusher"},
	{ID: "copper", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.copper.name", HintKey: "advancement.copper.hint", Item: "copper_ingot", Icon: "copper_ingot"},
	{ID: "tin_ingot", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.tin_ingot.name", HintKey: "advancement.tin_ingot.hint", Item: "skygear:ingot_tin", Guide: "tin"},
	{ID: "alloy_smelter", Chapter: ChapterAlloy, Parents: []string{"generator"}, NameKey: "advancement.alloy_smelter.name", HintKey: "advancement.alloy_smelter.hint", Item: "skygear:alloy_smelter"},
	{ID: "bronze_ingot", Chapter: ChapterAlloy, Parents: []string{"alloy_smelter", "tin_ingot"}, NameKey: "advancement.bronze_ingot.name", HintKey: "advancement.bronze_ingot.hint", Item: "skygear:ingot_bronze", Guide: "alloy_smelter"},
	{ID: "lead_ingot", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.lead_ingot.name", HintKey: "advancement.lead_ingot.hint", Item: "skygear:ingot_lead", Guide: "lead"},
	{ID: "energy_cell", Chapter: ChapterAutomation, Parents: []string{"lead_ingot", "generator"}, NameKey: "advancement.energy_cell.name", HintKey: "advancement.energy_cell.hint", Item: "skygear:energy_cell"},
	{ID: "gold", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.gold.name", HintKey: "advancement.gold.hint", Item: "gold_ingot", Icon: "gold_ingot"},
	{ID: "mesh_gold", Chapter: ChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_gold.name", HintKey: "advancement.mesh_gold.hint", Item: "skygear:mesh_gold"},
	{ID: "silver_ingot", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.silver_ingot.name", HintKey: "advancement.silver_ingot.hint", Item: "skygear:ingot_silver", Guide: "silver"},
	{ID: "electrum_ingot", Chapter: ChapterAlloy, Parents: []string{"alloy_smelter", "silver_ingot"}, NameKey: "advancement.electrum_ingot.name", HintKey: "advancement.electrum_ingot.hint", Item: "skygear:ingot_electrum", Guide: "alloy_smelter"},
	{ID: "nickel_ingot", Chapter: ChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.nickel_ingot.name", HintKey: "advancement.nickel_ingot.hint", Item: "skygear:ingot_nickel", Guide: "nickel"},
	{ID: "invar_ingot", Chapter: ChapterAlloy, Parents: []string{"alloy_smelter", "nickel_ingot"}, NameKey: "advancement.invar_ingot.name", HintKey: "advancement.invar_ingot.hint", Item: "skygear:ingot_invar", Guide: "alloy_smelter"},
	{ID: "steel_ingot", Chapter: ChapterAlloy, Parents: []string{"alloy_smelter"}, NameKey: "advancement.steel_ingot.name", HintKey: "advancement.steel_ingot.hint", Item: "skygear:ingot_steel", Guide: "alloy_smelter"},
	{ID: "drawer", Chapter: ChapterLogistics, Parents: []string{"chest"}, NameKey: "advancement.drawer.name", HintKey: "advancement.drawer.hint", Research: "drawer", Icon: "skygear:drawer"},
	{ID: "storage_controller", Chapter: ChapterLogistics, Parents: []string{"drawer", "item_conduit"}, NameKey: "advancement.storage_controller.name", HintKey: "advancement.storage_controller.hint", Research: "storage_controller", Icon: "skygear:storage_controller"},
	{ID: "storage_link", Chapter: ChapterLogistics, Parents: []string{"storage_controller"}, NameKey: "advancement.storage_link.name", HintKey: "advancement.storage_link.hint", Item: "skygear:storage_link"},
	{ID: "auto_sieve", Chapter: ChapterAutomation, Parents: []string{"generator", Sift}, NameKey: "advancement.auto_sieve.name", HintKey: "advancement.auto_sieve.hint", Research: "auto_sieve", Icon: "skygear:auto_sieve"},
	{ID: "electric_furnace", Chapter: ChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.electric_furnace.name", HintKey: "advancement.electric_furnace.hint", Item: "skygear:electric_furnace"},
	{ID: "harvester", Chapter: ChapterAutomation, Parents: []string{"generator", "wheat"}, NameKey: "advancement.harvester.name", HintKey: "advancement.harvester.hint", Research: "harvester", Icon: "skygear:harvester"},
	{ID: "item_vacuum", Chapter: ChapterLogistics, Parents: []string{"hopper"}, NameKey: "advancement.item_vacuum.name", HintKey: "advancement.item_vacuum.hint", Item: "skygear:item_vacuum"},
	{ID: "tree_feller", Chapter: ChapterAutomation, Parents: []string{"generator", GrowTree}, NameKey: "advancement.tree_feller.name", HintKey: "advancement.tree_feller.hint", Item: "skygear:tree_feller"},
	{ID: "redstone", Chapter: ChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.redstone.name", HintKey: "advancement.redstone.hint", Item: "lever"},
	{ID: "auto_crafter", Chapter: ChapterLogistics, Parents: []string{"hopper", "redstone"}, NameKey: "advancement.auto_crafter.name", HintKey: "advancement.auto_crafter.hint", Item: "skygear:auto_crafter"},
	{ID: "conduit_upgrade_2", Chapter: ChapterAutomation, Parents: []string{"electrum_ingot", "energy_conduit"}, NameKey: "advancement.conduit_upgrade_2.name", HintKey: "advancement.conduit_upgrade_2.hint", Item: "skygear:conduit_energy_2"},
	{ID: "conduit_upgrade_3", Chapter: ChapterAutomation, Parents: []string{"steel_ingot", "conduit_upgrade_2"}, NameKey: "advancement.conduit_upgrade_3.name", HintKey: "advancement.conduit_upgrade_3.hint", Item: "skygear:conduit_energy_3"},
	{ID: "mesh_diamond", Chapter: ChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_diamond.name", HintKey: "advancement.mesh_diamond.hint", Item: "skygear:mesh_diamond", Icon: "skygear:mesh_diamond"},
	{ID: "essence", Chapter: ChapterSieve, Parents: []string{"mesh_iron", "dust"}, NameKey: "advancement.essence.name", HintKey: "advancement.essence.hint", Item: "skygear:essence_zombie", Icon: "skygear:essence_zombie", Guide: "iron"},
	{ID: "mob_simulator", Chapter: ChapterHunting, Parents: []string{"essence", "alloy_smelter"}, NameKey: "advancement.mob_simulator.name", HintKey: "advancement.mob_simulator.hint", Item: "skygear:mob_simulator", Icon: "skygear:mob_simulator"},
	{ID: "ender_pearl", Chapter: ChapterHunting, Parents: []string{"mob_simulator", "mesh_diamond"}, NameKey: "advancement.ender_pearl.name", HintKey: "advancement.ender_pearl.hint", Item: "ender_pearl", Icon: "ender_pearl", Guide: "mob_simulator"},
	{ID: "mesh_netherite", Chapter: ChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_netherite.name", HintKey: "advancement.mesh_netherite.hint", Item: "skygear:mesh_netherite", Icon: "skygear:mesh_netherite"},
	{ID: "blaze_rod", Chapter: ChapterHunting, Parents: []string{"mob_simulator", "mesh_netherite"}, NameKey: "advancement.blaze_rod.name", HintKey: "advancement.blaze_rod.hint", Item: "blaze_rod", Icon: "blaze_rod", Guide: "mesh_netherite"},
	{ID: "fluid_conduit", Chapter: ChapterLogistics, Parents: []string{"item_conduit", "glass"}, NameKey: "advancement.fluid_conduit.name", HintKey: "advancement.fluid_conduit.hint", Research: "fluid_conduit", Icon: "skygear:conduit_fluid_1"},
	{ID: "fluid_tank", Chapter: ChapterLogistics, Parents: []string{"fluid_conduit"}, NameKey: "advancement.fluid_tank.name", HintKey: "advancement.fluid_tank.hint", Research: "fluid_tank", Icon: "skygear:fluid_tank"},
	{ID: "ore_washer", Chapter: ChapterAlloy, Parents: []string{"fluid_tank", "crusher"}, NameKey: "advancement.ore_washer.name", HintKey: "advancement.ore_washer.hint", Research: "ore_washer", Icon: "skygear:ore_washer"},
	{ID: "crucible", Chapter: ChapterAlloy, Parents: []string{"fluid_tank", "bronze_ingot"}, NameKey: "advancement.crucible.name", HintKey: "advancement.crucible.hint", Research: "crucible", Icon: "skygear:crucible"},
	{ID: "lava_generator", Chapter: ChapterAlloy, Parents: []string{"crucible"}, NameKey: "advancement.lava_generator.name", HintKey: "advancement.lava_generator.hint", Research: "lava_generator", Icon: "skygear:lava_generator"},
	{ID: "lab_alchemy", Chapter: ChapterLab, Parents: []string{"lab_engineering", "invar_ingot"}, NameKey: "advancement.lab_alchemy.name", HintKey: "advancement.lab_alchemy.hint", Item: "skygear:research_essence_alchemy", Guide: "laboratory"},
	{ID: "lab_farming", Chapter: ChapterLab, Parents: []string{"lab_engineering", "wheat"}, NameKey: "advancement.lab_farming.name", HintKey: "advancement.lab_farming.hint", Item: "skygear:research_essence_farming", Guide: "laboratory"},
	{ID: "lab_hunting", Chapter: ChapterLab, Parents: []string{"lab_engineering", "mob_simulator"}, NameKey: "advancement.lab_hunting.name", HintKey: "advancement.lab_hunting.hint", Item: "skygear:research_essence_hunting", Guide: "laboratory"},
	{ID: "diamond_tools", Chapter: ChapterGear, Parents: []string{"iron_pickaxe", "diamond"}, NameKey: "advancement.diamond_tools.name", HintKey: "advancement.diamond_tools.hint", Item: "diamond_pickaxe"},
	{ID: "obsidian", Chapter: ChapterFarming, Parents: []string{"lava"}, NameKey: "advancement.obsidian.name", HintKey: "advancement.obsidian.hint", Item: "obsidian", Guide: "diamond_tools"},
	{ID: "dimension_gate", Chapter: ChapterHunting, Parents: []string{"ender_pearl", "obsidian"}, NameKey: "advancement.dimension_gate.name", HintKey: "advancement.dimension_gate.hint", Research: "dimension_gate", Icon: "skygear:dimension_gate"},
	{ID: "first_kill", Chapter: ChapterHunting, Parents: []string{"dimension_gate"}, NameKey: "advancement.first_kill.name", HintKey: "advancement.first_kill.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: Wave1, Chapter: ChapterHunting, Parents: []string{Kill}, Event: true, NameKey: "advancement.wave_1.name", HintKey: "advancement.wave_1.hint", Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "wave_3", Chapter: ChapterHunting, Parents: []string{Wave1}, NameKey: "advancement.wave_3.name", HintKey: "advancement.wave_3.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: Wave5, Chapter: ChapterHunting, Parents: []string{Wave3}, Event: true, NameKey: "advancement.wave_5.name", HintKey: "advancement.wave_5.hint", Icon: "diamond_sword", Guide: "dimension_gate"},
	{ID: "tinker_table", Chapter: ChapterGear, Parents: []string{"iron_pickaxe", "bronze_ingot"}, NameKey: "advancement.tinker_table.name", HintKey: "advancement.tinker_table.hint", Research: "tinker_table", Icon: "skygear:tinker_table"},
	{ID: "tool_made", Chapter: ChapterGear, Parents: []string{"tinker_table"}, NameKey: "advancement.tool_made.name", HintKey: "advancement.tool_made.hint", Event: true, Icon: "iron_pickaxe", Guide: "tinker_table"},
	{ID: ToolMod, Chapter: ChapterGear, Parents: []string{ToolMade}, Event: true, NameKey: "advancement.tool_modifier.name", HintKey: "advancement.tool_modifier.hint", Icon: "redstone", Guide: "tinker_table"},
	{ID: ToolLevel, Chapter: ChapterGear, Parents: []string{ToolMade}, Event: true, NameKey: "advancement.tool_level.name", HintKey: "advancement.tool_level.hint", Icon: "experience_bottle", Guide: "tinker_table"},
	{ID: "magic_seed", Chapter: ChapterMagic, Parents: []string{Wave1}, NameKey: "advancement.magic_seed.name", HintKey: "advancement.magic_seed.hint", Item: "skygear:seed_flower_red", Guide: "dimension_gate"},
	{ID: Bloom, Chapter: ChapterMagic, Parents: []string{"magic_seed"}, Event: true, NameKey: "advancement.first_bloom.name", HintKey: "advancement.first_bloom.hint", Icon: "skygear:petal_red"},
	{ID: "lab_magic", Chapter: ChapterLab, Parents: []string{"lab_engineering", Bloom}, NameKey: "advancement.lab_magic.name", HintKey: "advancement.lab_magic.hint", Item: "skygear:research_essence_magic", Guide: "laboratory"},
	{ID: "ore_bloom", Chapter: ChapterMagic, Parents: []string{Bloom}, NameKey: "advancement.ore_bloom.name", HintKey: "advancement.ore_bloom.hint", Item: "skygear:seed_ore_iron"},
	{ID: "magic_sieve", Chapter: ChapterMagic, Parents: []string{"auto_sieve", "ore_bloom"}, NameKey: "advancement.magic_sieve.name", HintKey: "advancement.magic_sieve.hint", Research: "magic_sieve", Icon: "skygear:magic_sieve"},
	{ID: "gem_bloom", Chapter: ChapterMagic, Parents: []string{"ore_bloom"}, NameKey: "advancement.gem_bloom.name", HintKey: "advancement.gem_bloom.hint", Item: "skygear:seed_gem_diamond"},
	{ID: "essence_bloom", Chapter: ChapterMagic, Parents: []string{"gem_bloom"}, NameKey: "advancement.essence_bloom.name", HintKey: "advancement.essence_bloom.hint", Item: "skygear:seed_ess_enderman"},
	{ID: Chunk, Chapter: ChapterStart, Parents: []string{"cobble"}, Event: true, NameKey: "advancement.chunk.name", HintKey: "advancement.chunk.hint", Icon: "grass_block"},
	{ID: "shears", Chapter: ChapterGear, Parents: []string{"iron"}, NameKey: "advancement.shears.name", HintKey: "advancement.shears.hint", Item: "shears"},
	{ID: "iron_armor", Chapter: ChapterGear, Parents: []string{"iron"}, NameKey: "advancement.iron_armor.name", HintKey: "advancement.iron_armor.hint", Item: "iron_chestplate"},
	{ID: "lantern", Chapter: ChapterExtra, Parents: []string{"iron", "torch"}, NameKey: "advancement.lantern.name", HintKey: "advancement.lantern.hint", Item: "lantern"},
	{ID: "anvil", Chapter: ChapterGear, Parents: []string{"iron"}, NameKey: "advancement.anvil.name", HintKey: "advancement.anvil.hint", Item: "anvil"},
	{ID: "workbench", Chapter: ChapterExtra, Parents: []string{"iron"}, NameKey: "advancement.workbench.name", HintKey: "advancement.workbench.hint", Item: "stonecutter_block"},
	{ID: "blast_furnace", Chapter: ChapterExtra, Parents: []string{"furnace", "iron"}, NameKey: "advancement.blast_furnace.name", HintKey: "advancement.blast_furnace.hint", Item: "blast_furnace"},
	{ID: "copper_gear", Chapter: ChapterGear, Parents: []string{"copper"}, NameKey: "advancement.copper_gear.name", HintKey: "advancement.copper_gear.hint", Item: "copper_pickaxe"},
	{ID: "compass", Chapter: ChapterExtra, Parents: []string{"redstone"}, NameKey: "advancement.compass.name", HintKey: "advancement.compass.hint", Item: "compass"},
	{ID: "golden_carrot", Chapter: ChapterFarming, Parents: []string{"gold", "seeds"}, NameKey: "advancement.golden_carrot.name", HintKey: "advancement.golden_carrot.hint", Item: "golden_carrot"},
	{ID: "diamond_block", Chapter: ChapterExtra, Parents: []string{"diamond"}, NameKey: "advancement.diamond_block.name", HintKey: "advancement.diamond_block.hint", Item: "diamond_block"},
	{ID: "sandstone", Chapter: ChapterExtra, Parents: []string{"sand"}, NameKey: "advancement.sandstone.name", HintKey: "advancement.sandstone.hint", Item: "sandstone"},
	{ID: "quartz_block", Chapter: ChapterExtra, Parents: []string{"sand", "barrel"}, NameKey: "advancement.quartz_block.name", HintKey: "advancement.quartz_block.hint", Item: "quartz_block"},
	{ID: "amethyst_block", Chapter: ChapterExtra, Parents: []string{Sift}, NameKey: "advancement.amethyst_block.name", HintKey: "advancement.amethyst_block.hint", Item: "amethyst_block"},
	{ID: "bone_block", Chapter: ChapterHunting, Parents: []string{"mob_simulator"}, NameKey: "advancement.bone_block.name", HintKey: "advancement.bone_block.hint", Item: "bone_block"},
	{ID: "slime_block", Chapter: ChapterHunting, Parents: []string{"mob_simulator"}, NameKey: "advancement.slime_block.name", HintKey: "advancement.slime_block.hint", Item: "slime"},
	{ID: "blaze_powder", Chapter: ChapterHunting, Parents: []string{"blaze_rod"}, NameKey: "advancement.blaze_powder.name", HintKey: "advancement.blaze_powder.hint", Item: "blaze_powder"},
	{ID: "ender_eye", Chapter: ChapterHunting, Parents: []string{"blaze_powder", "ender_pearl"}, NameKey: "advancement.ender_eye.name", HintKey: "advancement.ender_eye.hint", Item: "ender_eye"},
	{ID: "ender_chest", Chapter: ChapterHunting, Parents: []string{"ender_eye", "obsidian"}, NameKey: "advancement.ender_chest.name", HintKey: "advancement.ender_chest.hint", Item: "ender_chest"},
	{ID: "diamond_armor", Chapter: ChapterGear, Parents: []string{"iron_armor", "diamond"}, NameKey: "advancement.diamond_armor.name", HintKey: "advancement.diamond_armor.hint", Item: "diamond_chestplate"},
	{ID: "dye", Chapter: ChapterMagic, Parents: []string{Bloom}, NameKey: "advancement.dye.name", HintKey: "advancement.dye.hint", Item: "red_dye"},
	{ID: "stained_glass", Chapter: ChapterMagic, Parents: []string{"dye", "glass"}, NameKey: "advancement.stained_glass.name", HintKey: "advancement.stained_glass.hint", Item: "blue_stained_glass"},
	{ID: "wave_8", Chapter: ChapterHunting, Parents: []string{Wave5}, NameKey: "advancement.wave_8.name", HintKey: "advancement.wave_8.hint", Event: true, Icon: "netherite_sword", Guide: "dimension_gate"},
	{ID: "tool_level_5", Chapter: ChapterGear, Parents: []string{ToolLevel}, NameKey: "advancement.tool_level_5.name", HintKey: "advancement.tool_level_5.hint", Event: true, Icon: "diamond_pickaxe", Guide: "tinker_table"},
	{ID: "ore_harvest", Chapter: ChapterMagic, Parents: []string{"ore_bloom"}, NameKey: "advancement.ore_harvest.name", HintKey: "advancement.ore_harvest.hint", Event: true, Icon: "iron_ingot", Guide: "magic_botany"},
	{ID: "gem_harvest", Chapter: ChapterMagic, Parents: []string{"gem_bloom"}, NameKey: "advancement.gem_harvest.name", HintKey: "advancement.gem_harvest.hint", Event: true, Icon: "diamond", Guide: "magic_gems"},
	{ID: "ess_harvest", Chapter: ChapterMagic, Parents: []string{"essence_bloom"}, NameKey: "advancement.ess_harvest.name", HintKey: "advancement.ess_harvest.hint", Event: true, Icon: "ender_pearl", Guide: "magic_essence"},
	{ID: "chunks_8", Chapter: ChapterExtra, Parents: []string{Chunk}, NameKey: "advancement.chunks_8.name", HintKey: "advancement.chunks_8.hint", Event: true, Icon: "grass_block"},
	{ID: "chunks_16", Chapter: ChapterExtra, Parents: []string{Chunks8}, NameKey: "advancement.chunks_16.name", HintKey: "advancement.chunks_16.hint", Event: true, Icon: "grass_block"},
	{ID: "project_industry", Chapter: ChapterProject, Parents: []string{"conduit_upgrade_3", "storage_controller", "auto_crafter", "lava_generator"}, NameKey: "advancement.project_industry.name", HintKey: "advancement.project_industry.hint", Research: "project_industry", Icon: "skygear:storage_controller"},
	{ID: "project_hunter", Chapter: ChapterProject, Parents: []string{Wave5, ToolMade, "diamond_armor", "blaze_rod"}, NameKey: "advancement.project_hunter.name", HintKey: "advancement.project_hunter.hint", Research: "project_hunter", Icon: "diamond_sword"},
	{ID: "project_botanist", Chapter: ChapterProject, Parents: []string{"essence_bloom", "dye"}, NameKey: "advancement.project_botanist.name", HintKey: "advancement.project_botanist.hint", Research: "project_botanist", Icon: "skygear:petal_red"},
	{ID: "project_dragon", Chapter: ChapterProject, Parents: []string{"project_industry", "project_hunter", "project_botanist"}, NameKey: "advancement.project_dragon.name", HintKey: "advancement.project_dragon.hint", Research: "project_dragon", Icon: "ender_eye"},
	{ID: Dragon, Chapter: ChapterProject, Parents: []string{"project_dragon"}, NameKey: "advancement.dragon_egg.name", HintKey: "advancement.dragon_egg.hint", Event: true, Icon: "dragon_egg", Guide: "project_dragon"},
	{ID: "wood_hopper", Chapter: ChapterLogistics, NameKey: "advancement.wood_hopper.name", Parents: []string{"chest"}, HintKey: "advancement.wood_hopper.hint", Item: "skygear:wood_hopper"},
	{ID: Lost, Chapter: ChapterExtra, Event: true, NameKey: "advancement.lost_found.name", Parents: []string{"chest"}, HintKey: "advancement.lost_found.hint", Icon: "chest"},
	{ID: FairyGift, Chapter: ChapterFairy, Event: true, NameKey: "advancement.fairy_gift.name", Parents: []string{"wood"}, HintKey: "advancement.fairy_gift.hint", Icon: "cake"},
	{ID: "fairy_dust", Chapter: ChapterFairy, Parents: []string{FairyGift}, NameKey: "advancement.fairy_dust.name", HintKey: "advancement.fairy_dust.hint", Item: "skygear:fairy_dust", Guide: "fairy_magic"},
	{ID: "fairy_emerald", Chapter: ChapterFairy, Parents: []string{"fairy_dust"}, NameKey: "advancement.fairy_emerald.name", HintKey: "advancement.fairy_emerald.hint", Item: "skygear:fairy_emerald", Guide: "fairy_magic"},
	{ID: "fairy_board", Chapter: ChapterFairy, Parents: []string{"fairy_dust"}, NameKey: "advancement.fairy_board.name", HintKey: "advancement.fairy_board.hint", Item: "skygear:fairy_board", Guide: "fairy_magic"},
	{ID: Favourite, Chapter: ChapterExtra, Event: true, NameKey: "advancement.codex_favourite.name", Parents: []string{"wood"}, HintKey: "advancement.codex_favourite.hint", Icon: "book"},
	{ID: "witch_water", Chapter: ChapterFarming, NameKey: "advancement.witch_water.name", Parents: []string{"barrel", Sift}, HintKey: "advancement.witch_water.hint", Item: "skygear:witch_water_bucket", Guide: "sieve"},
	{ID: "drawer_upgrade", Chapter: ChapterLogistics, NameKey: "advancement.drawer_upgrade.name", Parents: []string{"drawer"}, HintKey: "advancement.drawer_upgrade.hint", Item: "skygear:drawer_upgrade_capacity"},
	{ID: Stats, Chapter: ChapterAutomation, Event: true, NameKey: "advancement.stats.name", Parents: []string{"generator"}, HintKey: "advancement.stats.hint", Icon: "clock"},
	{ID: MachineTier, Chapter: ChapterAutomation, Event: true, NameKey: "advancement.machine_tier.name", Parents: []string{"auto_crafter", "energy_cell", "conduit_upgrade_2"}, HintKey: "advancement.machine_tier.hint", Icon: "skygear:kit_1", Guide: "machine_tier_1"},
	{ID: "config_copier", Chapter: ChapterAutomation, NameKey: "advancement.config_copier.name", Parents: []string{MachineTier}, HintKey: "advancement.config_copier.hint", Item: "skygear:config_copier"},
}

// advancementIndex is the index in Tree of each ID's first advancement.
var advancementIndex = func() map[string]int {
	m := map[string]int{}
	for i := range Tree {
		if _, ok := m[Tree[i].ID]; !ok {
			m[Tree[i].ID] = i
		}
	}
	return m
}()

// ByID returns an advancement of the tree.
func ByID(id string) (*Advancement, bool) {
	if i, ok := advancementIndex[id]; ok {
		return &Tree[i], true
	}
	return nil, false
}

// WithAncestors is a with every advancement before it, the
// first ones first.
func WithAncestors(a *Advancement) []*Advancement {
	var out []*Advancement
	seen := map[string]bool{}
	var walk func(a *Advancement)
	walk = func(a *Advancement) {
		if seen[a.ID] {
			return
		}
		seen[a.ID] = true
		for _, id := range a.Parents {
			if p, ok := ByID(id); ok {
				walk(p)
			}
		}
		out = append(out, a)
	}
	walk(a)
	return out
}

// Items are the naming.UnlockName of each advancement's Item, by index in
// Tree.
var Items = func() []string {
	out := make([]string, len(Tree))
	for i, a := range Tree {
		if a.Item != "" {
			out[i] = naming.UnlockName(a.Item)
		}
	}
	return out
}()

// PlayerItems are the names of the items in a player's inventory.
type PlayerItems struct {
	P     *player.Player
	Names map[string]bool
}

// InventoryNames are the item names in the inventory of each player in tx.
func InventoryNames(tx *world.Tx) []PlayerItems {
	var out []PlayerItems
	for e := range tx.Players() {
		p := e.(*player.Player)
		names := map[string]bool{}
		for _, st := range p.Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			names[name] = true
		}
		out = append(out, PlayerItems{P: p, Names: names})
	}
	return out
}

// HeldItemNames are the names of the items any of the players hold.
func HeldItemNames(held []PlayerItems) map[string]bool {
	names := map[string]bool{}
	for _, h := range held {
		for name := range h.Names {
			names[name] = true
		}
	}
	return names
}

// The player settings of how an advancement done is told: a toast, a chat
// line, the sound; whether the quest panel shows the next advancement on the
// screen. The vein mining setting is listed with them.
const (
	SettingToast      = "notify_advancement_toast"
	SettingChat       = "notify_advancement_chat"
	SettingSound      = "notify_advancement_sound"
	SettingQuestPanel = "hud_quest_panel"
)

// Title is the advancement's name in lang.
func (a *Advancement) Title(lang string) string { return i18n.T(lang, a.NameKey) }

// HintText is the advancement's hint in lang.
func (a *Advancement) HintText(lang string) string { return i18n.T(lang, a.HintKey) }

// FlatHint is an advancement's hint in lang on one line.
func FlatHint(lang string, a *Advancement) string {
	return strings.ReplaceAll(a.HintText(lang), "\n", " ")
}

// Icon is the icon of an advancement in the advancement screen.
func Icon(a *Advancement) string {
	name := a.Icon
	if name == "" {
		name = a.Item
	}
	if it, ok := world.ItemByName(naming.UnlockName(name), 0); ok {
		return formicon.Of(item.NewStack(it, 1))
	}
	return formicon.Mission
}

// Stack is the item an advancement is about: the one it asks
// for, else its icon.
func Stack(a *Advancement) (item.Stack, bool) {
	for _, name := range []string{a.Item, a.Icon} {
		if name == "" {
			continue
		}
		if it, ok := world.ItemByName(naming.UnlockName(name), 0); ok {
			return item.NewStack(it, 1), true
		}
	}
	return item.Stack{}, false
}
