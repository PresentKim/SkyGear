package dragonfactory

import (
	"github.com/presentkim/dragonfactory/common/i18n"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/account"
)

// Advancements (docs/design/02-dragonfactory.md §10, 05-sky-progression.md
// §5): Bedrock has no advancements a server can use, so they are ours. An
// advancement is a step of the progression with a hint of how to do it; the
// quest panel shows the first one open. They belong to the save and are
// shared by everyone on the island like the researches. Every advancement
// is shown from the start, in its chapter; one whose parents are not all
// done yet is drawn darker but can be done all the same. The tree is a
// draft.

// Advancement is one step of the advancement tree. It is done by having an
// item, by a research or by an event the game reports (advance).
type Advancement struct {
	ID string
	// NameKey and HintKey are the catalog keys of its name and of the hint
	// that says how to do it, one HUD row per line (Title, HintText).
	NameKey, HintKey string
	// Chapter is the chapter of the advancement screen it is in.
	Chapter advChapter
	// Parents come before it: it is open once they are all done (none for
	// the first). Doing it first is fine; they stay as they are.
	Parents []string
	// Item is the name of an item to have in the inventory (unlockName
	// form), or empty.
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
	advGrowTree = "grow_tree"
	advSift     = "sift"
	advMachine  = "machine"
	advChunk    = "chunk"
	// advWave1 and advWave5 are done by clearing levels 1 and 5 of the gate
	// (wave.go).
	advWave1 = "wave_1"
	// advToolMod and advToolLevel are done by putting a modifier on a tool and
	// by a tool reaching level 1 (tinkerform.go).
	advToolMod = "tool_modifier"
	// advBloom is done by harvesting a magic plant (magic.go).
	advBloom     = "first_bloom"
	advToolLevel = "tool_level"
	advWave5     = "wave_5"

	advKill       = "first_kill"
	advWave3      = "wave_3"
	advWave8      = "wave_8"
	advToolMade   = "tool_made"
	advTool5      = "tool_level_5"
	advOreHarvest = "ore_harvest"
	advGemHarvest = "gem_harvest"
	advEssHarvest = "ess_harvest"
	advChunks8    = "chunks_8"
	advChunks16   = "chunks_16"
	// advEssenceResearch is done by a research that cost research essences
	// (research.go).
	advEssenceResearch = "essence_research"
	// advDragon is done by placing the dragon egg (project.go).
	advDragon = "dragon_egg"
	// advMachineTier is done by putting a kit on a machine (tier.go),
	// advStats by opening the production stats (stats.go), advFairyGift by
	// a gift fairy's gift (fairy.go), advFavourite by a favourite in the
	// codex (codexgoal.go) and advLost by taking an item back from the lost
	// and found (lostfound.go).
	advMachineTier = "machine_tier"
	advStats       = "stats"
	advFairyGift   = "fairy_gift"
	advFavourite   = "codex_favourite"
	advLost        = "lost_found"
)

// advancementTree is every advancement in the order of the game's flow; the
// quest panel follows this order, so the main path comes first. The screen
// shows each chapter's advancements in this order too.
var advancementTree = []Advancement{
	{ID: "wood", Chapter: advChapterStart, NameKey: "dragonfactory.advancement.init.1", HintKey: "dragonfactory.advancement.init.2", Item: "oak_log"},
	{ID: "planks", Chapter: advChapterStart, Parents: []string{"wood"}, NameKey: "dragonfactory.advancement.init.3", HintKey: "dragonfactory.advancement.init.4", Item: "oak_planks"},
	{ID: "crafting_table", Chapter: advChapterStart, Parents: []string{"planks"}, NameKey: "dragonfactory.advancement.init.5", HintKey: "dragonfactory.advancement.init.6", Item: "crafting_table"},
	{ID: "research_table", Chapter: advChapterStart, Parents: []string{"crafting_table"}, NameKey: "dragonfactory.advancement.init.7", HintKey: "dragonfactory.advancement.init.8", Item: "dragonfactory:research_table"},
	{ID: "first_research", Chapter: advChapterStart, Parents: []string{"research_table"}, NameKey: "dragonfactory.advancement.init.9", HintKey: "dragonfactory.advancement.init.10", Research: "stick", Icon: "stick"},
	{ID: "pebble", Chapter: advChapterStart, Parents: []string{"first_research"}, NameKey: "dragonfactory.advancement.init.11", HintKey: "dragonfactory.advancement.init.12", Item: "dragonfactory:pebble_stone", Guide: "stick"},
	{ID: "cobble", Chapter: advChapterStart, Parents: []string{"pebble"}, NameKey: "dragonfactory.advancement.init.13", HintKey: "dragonfactory.advancement.init.14", Item: "cobblestone"},
	{ID: "wood_tools", Chapter: advChapterStart, Parents: []string{"first_research"}, NameKey: "dragonfactory.advancement.init.17", HintKey: "dragonfactory.advancement.init.18", Item: "wooden_pickaxe"},
	{ID: "chest", Chapter: advChapterStart, Parents: []string{"first_research"}, NameKey: "advancement.chest.name", HintKey: "advancement.chest.hint", Item: "chest"},
	{ID: "stone_axe", Chapter: advChapterStart, Parents: []string{"cobble"}, NameKey: "dragonfactory.advancement.init.23", HintKey: "dragonfactory.advancement.init.24", Item: "stone_axe"},
	{ID: "fiber", Chapter: advChapterStart, Parents: []string{"stone_axe"}, NameKey: "dragonfactory.advancement.init.25", HintKey: "dragonfactory.advancement.init.26", Item: "dragonfactory:bark_fiber", Guide: "stone_tools"},
	{ID: "string", Chapter: advChapterStart, Parents: []string{"fiber"}, NameKey: "dragonfactory.advancement.init.27", HintKey: "dragonfactory.advancement.init.28", Item: "string"},
	{ID: "hammer", Chapter: advChapterStart, Parents: []string{"cobble"}, NameKey: "dragonfactory.advancement.init.29", HintKey: "dragonfactory.advancement.init.30", Item: "dragonfactory:stone_hammer"},
	{ID: "gravel", Chapter: advChapterStart, Parents: []string{"hammer"}, NameKey: "dragonfactory.advancement.init.31", HintKey: "dragonfactory.advancement.init.32", Item: "gravel", Guide: "hammer"},
	{ID: "sand", Chapter: advChapterStart, Parents: []string{"gravel"}, NameKey: "advancement.sand.name", HintKey: "advancement.sand.hint", Item: "sand", Guide: "hammer"},
	{ID: "sieve", Chapter: advChapterSieve, Parents: []string{"string"}, NameKey: "dragonfactory.advancement.init.33", HintKey: "dragonfactory.advancement.init.34", Item: "dragonfactory:sieve"},
	{ID: "mesh_string", Chapter: advChapterSieve, Parents: []string{"sieve"}, NameKey: "advancement.mesh_string.name", HintKey: "advancement.mesh_string.hint", Item: "dragonfactory:mesh_string"},
	{ID: advSift, Chapter: advChapterSieve, Parents: []string{"mesh_string", "gravel"}, Event: true, NameKey: "dragonfactory.advancement.init.35", HintKey: "dragonfactory.advancement.init.36", Icon: "gravel", Guide: "sieve"},
	{ID: "mesh_flint", Chapter: advChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_flint.name", HintKey: "advancement.mesh_flint.hint", Item: "dragonfactory:mesh_flint"},
	{ID: "ore_block", Chapter: advChapterSieve, NameKey: "advancement.ore_block.name", Parents: []string{advSift}, HintKey: "advancement.ore_block.hint", Item: "iron_ore"},
	{ID: "seeds", Chapter: advChapterFarming, Parents: []string{advSift}, NameKey: "advancement.seeds.name", HintKey: "advancement.seeds.hint", Item: "wheat_seeds", Guide: "sieve"},
	{ID: "wheat", Chapter: advChapterFarming, Parents: []string{"seeds"}, NameKey: "advancement.wheat.name", HintKey: "advancement.wheat.hint", Item: "wheat", Guide: "wood_tools"},
	{ID: "bread", Chapter: advChapterFarming, Parents: []string{"wheat"}, NameKey: "advancement.bread.name", HintKey: "advancement.bread.hint", Item: "bread"},
	{ID: "furnace", Chapter: advChapterStart, Parents: []string{"cobble"}, NameKey: "dragonfactory.advancement.init.39", HintKey: "dragonfactory.advancement.init.40", Item: "furnace"},
	{ID: "torch", Chapter: advChapterStart, Parents: []string{"furnace"}, NameKey: "advancement.torch.name", HintKey: "advancement.torch.hint", Item: "torch"},
	{ID: "composter", Chapter: advChapterFarming, Parents: []string{"wood_tools"}, NameKey: "dragonfactory.advancement.init.19", HintKey: "dragonfactory.advancement.init.20", Item: "composter"},
	{ID: "barrel", Chapter: advChapterFarming, Parents: []string{"composter"}, NameKey: "dragonfactory.advancement.init.21", HintKey: "dragonfactory.advancement.init.22", Item: "dragonfactory:barrel_oak"},
	{ID: advGrowTree, Chapter: advChapterStart, Parents: []string{"wood"}, Event: true, NameKey: "dragonfactory.advancement.init.15", HintKey: "dragonfactory.advancement.init.16", Icon: "oak_sapling"},
	{ID: "dust", Chapter: advChapterStart, Parents: []string{"sand"}, NameKey: "dragonfactory.advancement.init.37", HintKey: "dragonfactory.advancement.init.38", Item: "dragonfactory:dust", Guide: "hammer"},
	{ID: "clay", Chapter: advChapterFarming, Parents: []string{"barrel", "dust"}, NameKey: "advancement.clay.name", HintKey: "advancement.clay.hint", Item: "clay", Icon: "clay"},
	{ID: "wooden_bucket", Chapter: advChapterFarming, Parents: []string{"barrel"}, NameKey: "advancement.wooden_bucket.name", HintKey: "advancement.wooden_bucket.hint", Item: "dragonfactory:wooden_bucket"},
	{ID: "glass", Chapter: advChapterStart, Parents: []string{"furnace", "sand"}, NameKey: "advancement.glass.name", HintKey: "advancement.glass.hint", Item: "glass", Guide: "furnace"},
	{ID: "iron", Chapter: advChapterSieve, Parents: []string{"ore_block", "furnace"}, NameKey: "dragonfactory.advancement.init.41", HintKey: "dragonfactory.advancement.init.42", Item: "iron_ingot"},
	{ID: "mesh_iron", Chapter: advChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_iron.name", HintKey: "advancement.mesh_iron.hint", Item: "dragonfactory:mesh_iron"},
	{ID: "ore_dust", Chapter: advChapterSieve, Parents: []string{"ore_block", "hammer"}, NameKey: "advancement.ore_dust.name", HintKey: "advancement.ore_dust.hint", Item: "dragonfactory:metal_dust_iron", Guide: "iron"},
	{ID: "diamond", Chapter: advChapterSieve, Parents: []string{"mesh_iron", advSift}, NameKey: "advancement.diamond.name", HintKey: "advancement.diamond.hint", Item: "diamond"},
	{ID: "bucket", Chapter: advChapterFarming, Parents: []string{"iron"}, NameKey: "advancement.bucket.name", HintKey: "advancement.bucket.hint", Item: "bucket"},
	{ID: "stone_barrel", Chapter: advChapterFarming, Parents: []string{"barrel", "furnace"}, NameKey: "advancement.stone_barrel.name", HintKey: "advancement.stone_barrel.hint", Item: "dragonfactory:barrel_stone", Icon: "dragonfactory:barrel_stone"},
	{ID: "lava", Chapter: advChapterFarming, Parents: []string{"stone_barrel", "bucket"}, NameKey: "advancement.lava.name", HintKey: "advancement.lava.hint", Item: "lava_bucket", Icon: "lava_bucket", Guide: "stone_barrel"},
	{ID: "generator", Chapter: advChapterAutomation, Parents: []string{"iron"}, NameKey: "advancement.generator.name", HintKey: "advancement.generator.hint", Research: "generator", Icon: "dragonfactory:generator"},
	{ID: advMachine, Chapter: advChapterAutomation, Parents: []string{"generator"}, Event: true, NameKey: "dragonfactory.advancement.init.43", HintKey: "dragonfactory.advancement.init.44", Icon: "dragonfactory:crusher", Guide: "generator"},
	{ID: "laboratory", Chapter: advChapterLab, Parents: []string{"generator", "glass"}, NameKey: "advancement.laboratory.name", HintKey: "advancement.laboratory.hint", Item: "dragonfactory:laboratory"},
	{ID: "lab_engineering", Chapter: advChapterLab, Parents: []string{"laboratory"}, NameKey: "advancement.lab_engineering.name", HintKey: "advancement.lab_engineering.hint", Item: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: advEssenceResearch, Chapter: advChapterLab, Parents: []string{"lab_engineering"}, Event: true, NameKey: "advancement.essence_research.name", HintKey: "advancement.essence_research.hint", Icon: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: "hopper", Chapter: advChapterLogistics, Parents: []string{"iron", "chest"}, NameKey: "advancement.hopper.name", HintKey: "advancement.hopper.hint", Research: "hopper", Icon: "hopper"},
	{ID: "energy_conduit", Chapter: advChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.energy_conduit.name", HintKey: "advancement.energy_conduit.hint", Research: "energy_conduit", Icon: "dragonfactory:conduit_energy_1"},
	{ID: "item_conduit", Chapter: advChapterLogistics, Parents: []string{"hopper"}, NameKey: "advancement.item_conduit.name", HintKey: "advancement.item_conduit.hint", Research: "item_conduit", Icon: "dragonfactory:conduit_item_1"},
	{ID: "iron_pickaxe", Chapter: advChapterGear, Parents: []string{"iron"}, NameKey: "advancement.iron_pickaxe.name", HintKey: "advancement.iron_pickaxe.hint", Item: "iron_pickaxe"},
	{ID: "auto_miner", Chapter: advChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.auto_miner.name", HintKey: "advancement.auto_miner.hint", Research: "auto_miner", Icon: "dragonfactory:auto_miner"},
	{ID: "crusher", Chapter: advChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.crusher.name", HintKey: "advancement.crusher.hint", Research: "crusher"},
	{ID: "copper", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.copper.name", HintKey: "advancement.copper.hint", Item: "copper_ingot", Icon: "copper_ingot"},
	{ID: "tin_ingot", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.tin_ingot.name", HintKey: "advancement.tin_ingot.hint", Item: "dragonfactory:ingot_tin", Guide: "tin"},
	{ID: "alloy_smelter", Chapter: advChapterAlloy, Parents: []string{"generator"}, NameKey: "advancement.alloy_smelter.name", HintKey: "advancement.alloy_smelter.hint", Item: "dragonfactory:alloy_smelter"},
	{ID: "bronze_ingot", Chapter: advChapterAlloy, Parents: []string{"alloy_smelter", "tin_ingot"}, NameKey: "advancement.bronze_ingot.name", HintKey: "advancement.bronze_ingot.hint", Item: "dragonfactory:ingot_bronze", Guide: "alloy_smelter"},
	{ID: "lead_ingot", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.lead_ingot.name", HintKey: "advancement.lead_ingot.hint", Item: "dragonfactory:ingot_lead", Guide: "lead"},
	{ID: "energy_cell", Chapter: advChapterAutomation, Parents: []string{"lead_ingot", "generator"}, NameKey: "advancement.energy_cell.name", HintKey: "advancement.energy_cell.hint", Item: "dragonfactory:energy_cell"},
	{ID: "gold", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.gold.name", HintKey: "advancement.gold.hint", Item: "gold_ingot", Icon: "gold_ingot"},
	{ID: "mesh_gold", Chapter: advChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_gold.name", HintKey: "advancement.mesh_gold.hint", Item: "dragonfactory:mesh_gold"},
	{ID: "silver_ingot", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.silver_ingot.name", HintKey: "advancement.silver_ingot.hint", Item: "dragonfactory:ingot_silver", Guide: "silver"},
	{ID: "electrum_ingot", Chapter: advChapterAlloy, Parents: []string{"alloy_smelter", "silver_ingot"}, NameKey: "advancement.electrum_ingot.name", HintKey: "advancement.electrum_ingot.hint", Item: "dragonfactory:ingot_electrum", Guide: "alloy_smelter"},
	{ID: "nickel_ingot", Chapter: advChapterSieve, Parents: []string{"ore_block"}, NameKey: "advancement.nickel_ingot.name", HintKey: "advancement.nickel_ingot.hint", Item: "dragonfactory:ingot_nickel", Guide: "nickel"},
	{ID: "invar_ingot", Chapter: advChapterAlloy, Parents: []string{"alloy_smelter", "nickel_ingot"}, NameKey: "advancement.invar_ingot.name", HintKey: "advancement.invar_ingot.hint", Item: "dragonfactory:ingot_invar", Guide: "alloy_smelter"},
	{ID: "steel_ingot", Chapter: advChapterAlloy, Parents: []string{"alloy_smelter"}, NameKey: "advancement.steel_ingot.name", HintKey: "advancement.steel_ingot.hint", Item: "dragonfactory:ingot_steel", Guide: "alloy_smelter"},
	{ID: "drawer", Chapter: advChapterLogistics, Parents: []string{"chest"}, NameKey: "advancement.drawer.name", HintKey: "advancement.drawer.hint", Research: "drawer", Icon: "dragonfactory:drawer"},
	{ID: "storage_controller", Chapter: advChapterLogistics, Parents: []string{"drawer", "item_conduit"}, NameKey: "advancement.storage_controller.name", HintKey: "advancement.storage_controller.hint", Research: "storage_controller", Icon: "dragonfactory:storage_controller"},
	{ID: "storage_link", Chapter: advChapterLogistics, Parents: []string{"storage_controller"}, NameKey: "advancement.storage_link.name", HintKey: "advancement.storage_link.hint", Item: "dragonfactory:storage_link"},
	{ID: "auto_sieve", Chapter: advChapterAutomation, Parents: []string{"generator", advSift}, NameKey: "advancement.auto_sieve.name", HintKey: "advancement.auto_sieve.hint", Research: "auto_sieve", Icon: "dragonfactory:auto_sieve"},
	{ID: "electric_furnace", Chapter: advChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.electric_furnace.name", HintKey: "advancement.electric_furnace.hint", Item: "dragonfactory:electric_furnace"},
	{ID: "harvester", Chapter: advChapterAutomation, Parents: []string{"generator", "wheat"}, NameKey: "advancement.harvester.name", HintKey: "advancement.harvester.hint", Research: "harvester", Icon: "dragonfactory:harvester"},
	{ID: "item_vacuum", Chapter: advChapterLogistics, Parents: []string{"hopper"}, NameKey: "advancement.item_vacuum.name", HintKey: "advancement.item_vacuum.hint", Item: "dragonfactory:item_vacuum"},
	{ID: "tree_feller", Chapter: advChapterAutomation, Parents: []string{"generator", advGrowTree}, NameKey: "advancement.tree_feller.name", HintKey: "advancement.tree_feller.hint", Item: "dragonfactory:tree_feller"},
	{ID: "redstone", Chapter: advChapterAutomation, Parents: []string{"generator"}, NameKey: "advancement.redstone.name", HintKey: "advancement.redstone.hint", Item: "lever"},
	{ID: "auto_crafter", Chapter: advChapterLogistics, Parents: []string{"hopper", "redstone"}, NameKey: "advancement.auto_crafter.name", HintKey: "advancement.auto_crafter.hint", Item: "dragonfactory:auto_crafter"},
	{ID: "conduit_upgrade_2", Chapter: advChapterAutomation, Parents: []string{"electrum_ingot", "energy_conduit"}, NameKey: "advancement.conduit_upgrade_2.name", HintKey: "advancement.conduit_upgrade_2.hint", Item: "dragonfactory:conduit_energy_2"},
	{ID: "conduit_upgrade_3", Chapter: advChapterAutomation, Parents: []string{"steel_ingot", "conduit_upgrade_2"}, NameKey: "advancement.conduit_upgrade_3.name", HintKey: "advancement.conduit_upgrade_3.hint", Item: "dragonfactory:conduit_energy_3"},
	{ID: "mesh_diamond", Chapter: advChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_diamond.name", HintKey: "advancement.mesh_diamond.hint", Item: "dragonfactory:mesh_diamond", Icon: "dragonfactory:mesh_diamond"},
	{ID: "essence", Chapter: advChapterSieve, Parents: []string{"mesh_iron", "dust"}, NameKey: "advancement.essence.name", HintKey: "advancement.essence.hint", Item: "dragonfactory:essence_zombie", Icon: "dragonfactory:essence_zombie", Guide: "iron"},
	{ID: "mob_simulator", Chapter: advChapterHunting, Parents: []string{"essence", "alloy_smelter"}, NameKey: "advancement.mob_simulator.name", HintKey: "advancement.mob_simulator.hint", Item: "dragonfactory:mob_simulator", Icon: "dragonfactory:mob_simulator"},
	{ID: "ender_pearl", Chapter: advChapterHunting, Parents: []string{"mob_simulator", "mesh_diamond"}, NameKey: "advancement.ender_pearl.name", HintKey: "advancement.ender_pearl.hint", Item: "ender_pearl", Icon: "ender_pearl", Guide: "mob_simulator"},
	{ID: "mesh_netherite", Chapter: advChapterSieve, Parents: []string{"mesh_string"}, NameKey: "advancement.mesh_netherite.name", HintKey: "advancement.mesh_netherite.hint", Item: "dragonfactory:mesh_netherite", Icon: "dragonfactory:mesh_netherite"},
	{ID: "blaze_rod", Chapter: advChapterHunting, Parents: []string{"mob_simulator", "mesh_netherite"}, NameKey: "advancement.blaze_rod.name", HintKey: "advancement.blaze_rod.hint", Item: "blaze_rod", Icon: "blaze_rod", Guide: "mesh_netherite"},
	{ID: "fluid_conduit", Chapter: advChapterLogistics, Parents: []string{"item_conduit", "glass"}, NameKey: "dragonfactory.advancement.init.47", HintKey: "dragonfactory.advancement.init.48", Research: "fluid_conduit", Icon: "dragonfactory:conduit_fluid_1"},
	{ID: "fluid_tank", Chapter: advChapterLogistics, Parents: []string{"fluid_conduit"}, NameKey: "advancement.fluid_tank.name", HintKey: "advancement.fluid_tank.hint", Research: "fluid_tank", Icon: "dragonfactory:fluid_tank"},
	{ID: "ore_washer", Chapter: advChapterAlloy, Parents: []string{"fluid_tank", "crusher"}, NameKey: "advancement.ore_washer.name", HintKey: "advancement.ore_washer.hint", Research: "ore_washer", Icon: "dragonfactory:ore_washer"},
	{ID: "crucible", Chapter: advChapterAlloy, Parents: []string{"fluid_tank", "bronze_ingot"}, NameKey: "advancement.crucible.name", HintKey: "advancement.crucible.hint", Research: "crucible", Icon: "dragonfactory:crucible"},
	{ID: "lava_generator", Chapter: advChapterAlloy, Parents: []string{"crucible"}, NameKey: "advancement.lava_generator.name", HintKey: "advancement.lava_generator.hint", Research: "lava_generator", Icon: "dragonfactory:lava_generator"},
	{ID: "lab_alchemy", Chapter: advChapterLab, Parents: []string{"lab_engineering", "invar_ingot"}, NameKey: "advancement.lab_alchemy.name", HintKey: "advancement.lab_alchemy.hint", Item: "dragonfactory:research_essence_alchemy", Guide: "laboratory"},
	{ID: "lab_farming", Chapter: advChapterLab, Parents: []string{"lab_engineering", "wheat"}, NameKey: "advancement.lab_farming.name", HintKey: "advancement.lab_farming.hint", Item: "dragonfactory:research_essence_farming", Guide: "laboratory"},
	{ID: "lab_hunting", Chapter: advChapterLab, Parents: []string{"lab_engineering", "mob_simulator"}, NameKey: "advancement.lab_hunting.name", HintKey: "advancement.lab_hunting.hint", Item: "dragonfactory:research_essence_hunting", Guide: "laboratory"},
	{ID: "diamond_tools", Chapter: advChapterGear, Parents: []string{"iron_pickaxe", "diamond"}, NameKey: "advancement.diamond_tools.name", HintKey: "advancement.diamond_tools.hint", Item: "diamond_pickaxe"},
	{ID: "obsidian", Chapter: advChapterFarming, Parents: []string{"lava"}, NameKey: "advancement.obsidian.name", HintKey: "advancement.obsidian.hint", Item: "obsidian", Guide: "diamond_tools"},
	{ID: "dimension_gate", Chapter: advChapterHunting, Parents: []string{"ender_pearl", "obsidian"}, NameKey: "advancement.dimension_gate.name", HintKey: "advancement.dimension_gate.hint", Research: "dimension_gate", Icon: "dragonfactory:dimension_gate"},
	{ID: "first_kill", Chapter: advChapterHunting, Parents: []string{"dimension_gate"}, NameKey: "advancement.first_kill.name", HintKey: "advancement.first_kill.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: advWave1, Chapter: advChapterHunting, Parents: []string{advKill}, Event: true, NameKey: "advancement.wave_1.name", HintKey: "advancement.wave_1.hint", Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "wave_3", Chapter: advChapterHunting, Parents: []string{advWave1}, NameKey: "advancement.wave_3.name", HintKey: "advancement.wave_3.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: advWave5, Chapter: advChapterHunting, Parents: []string{advWave3}, Event: true, NameKey: "advancement.wave_5.name", HintKey: "advancement.wave_5.hint", Icon: "diamond_sword", Guide: "dimension_gate"},
	{ID: "tinker_table", Chapter: advChapterGear, Parents: []string{"iron_pickaxe", "bronze_ingot"}, NameKey: "advancement.tinker_table.name", HintKey: "advancement.tinker_table.hint", Research: "tinker_table", Icon: "dragonfactory:tinker_table"},
	{ID: "tool_made", Chapter: advChapterGear, Parents: []string{"tinker_table"}, NameKey: "advancement.tool_made.name", HintKey: "advancement.tool_made.hint", Event: true, Icon: "iron_pickaxe", Guide: "tinker_table"},
	{ID: advToolMod, Chapter: advChapterGear, Parents: []string{advToolMade}, Event: true, NameKey: "advancement.tool_modifier.name", HintKey: "advancement.tool_modifier.hint", Icon: "redstone", Guide: "tinker_table"},
	{ID: advToolLevel, Chapter: advChapterGear, Parents: []string{advToolMade}, Event: true, NameKey: "advancement.tool_level.name", HintKey: "advancement.tool_level.hint", Icon: "experience_bottle", Guide: "tinker_table"},
	{ID: "magic_seed", Chapter: advChapterMagic, Parents: []string{advWave1}, NameKey: "advancement.magic_seed.name", HintKey: "advancement.magic_seed.hint", Item: "dragonfactory:seed_flower_red", Guide: "dimension_gate"},
	{ID: advBloom, Chapter: advChapterMagic, Parents: []string{"magic_seed"}, Event: true, NameKey: "advancement.first_bloom.name", HintKey: "advancement.first_bloom.hint", Icon: "dragonfactory:petal_red"},
	{ID: "lab_magic", Chapter: advChapterLab, Parents: []string{"lab_engineering", advBloom}, NameKey: "advancement.lab_magic.name", HintKey: "advancement.lab_magic.hint", Item: "dragonfactory:research_essence_magic", Guide: "laboratory"},
	{ID: "ore_bloom", Chapter: advChapterMagic, Parents: []string{advBloom}, NameKey: "advancement.ore_bloom.name", HintKey: "advancement.ore_bloom.hint", Item: "dragonfactory:seed_ore_iron"},
	{ID: "magic_sieve", Chapter: advChapterMagic, Parents: []string{"auto_sieve", "ore_bloom"}, NameKey: "advancement.magic_sieve.name", HintKey: "advancement.magic_sieve.hint", Research: "magic_sieve", Icon: "dragonfactory:magic_sieve"},
	{ID: "gem_bloom", Chapter: advChapterMagic, Parents: []string{"ore_bloom"}, NameKey: "advancement.gem_bloom.name", HintKey: "advancement.gem_bloom.hint", Item: "dragonfactory:seed_gem_diamond"},
	{ID: "essence_bloom", Chapter: advChapterMagic, Parents: []string{"gem_bloom"}, NameKey: "advancement.essence_bloom.name", HintKey: "advancement.essence_bloom.hint", Item: "dragonfactory:seed_ess_enderman"},
	{ID: advChunk, Chapter: advChapterStart, Parents: []string{"cobble"}, Event: true, NameKey: "dragonfactory.advancement.init.45", HintKey: "dragonfactory.advancement.init.46", Icon: "grass_block"},
	{ID: "shears", Chapter: advChapterGear, Parents: []string{"iron"}, NameKey: "advancement.shears.name", HintKey: "advancement.shears.hint", Item: "shears"},
	{ID: "iron_armor", Chapter: advChapterGear, Parents: []string{"iron"}, NameKey: "advancement.iron_armor.name", HintKey: "advancement.iron_armor.hint", Item: "iron_chestplate"},
	{ID: "lantern", Chapter: advChapterExtra, Parents: []string{"iron", "torch"}, NameKey: "advancement.lantern.name", HintKey: "advancement.lantern.hint", Item: "lantern"},
	{ID: "anvil", Chapter: advChapterGear, Parents: []string{"iron"}, NameKey: "advancement.anvil.name", HintKey: "advancement.anvil.hint", Item: "anvil"},
	{ID: "workbench", Chapter: advChapterExtra, Parents: []string{"iron"}, NameKey: "advancement.workbench.name", HintKey: "advancement.workbench.hint", Item: "stonecutter_block"},
	{ID: "blast_furnace", Chapter: advChapterExtra, Parents: []string{"furnace", "iron"}, NameKey: "advancement.blast_furnace.name", HintKey: "advancement.blast_furnace.hint", Item: "blast_furnace"},
	{ID: "copper_gear", Chapter: advChapterGear, Parents: []string{"copper"}, NameKey: "advancement.copper_gear.name", HintKey: "advancement.copper_gear.hint", Item: "copper_pickaxe"},
	{ID: "compass", Chapter: advChapterExtra, Parents: []string{"redstone"}, NameKey: "advancement.compass.name", HintKey: "advancement.compass.hint", Item: "compass"},
	{ID: "golden_carrot", Chapter: advChapterFarming, Parents: []string{"gold", "seeds"}, NameKey: "advancement.golden_carrot.name", HintKey: "advancement.golden_carrot.hint", Item: "golden_carrot"},
	{ID: "diamond_block", Chapter: advChapterExtra, Parents: []string{"diamond"}, NameKey: "advancement.diamond_block.name", HintKey: "advancement.diamond_block.hint", Item: "diamond_block"},
	{ID: "sandstone", Chapter: advChapterExtra, Parents: []string{"sand"}, NameKey: "advancement.sandstone.name", HintKey: "advancement.sandstone.hint", Item: "sandstone"},
	{ID: "quartz_block", Chapter: advChapterExtra, Parents: []string{"sand", "barrel"}, NameKey: "advancement.quartz_block.name", HintKey: "advancement.quartz_block.hint", Item: "quartz_block"},
	{ID: "amethyst_block", Chapter: advChapterExtra, Parents: []string{advSift}, NameKey: "advancement.amethyst_block.name", HintKey: "advancement.amethyst_block.hint", Item: "amethyst_block"},
	{ID: "bone_block", Chapter: advChapterHunting, Parents: []string{"mob_simulator"}, NameKey: "advancement.bone_block.name", HintKey: "advancement.bone_block.hint", Item: "bone_block"},
	{ID: "slime_block", Chapter: advChapterHunting, Parents: []string{"mob_simulator"}, NameKey: "advancement.slime_block.name", HintKey: "advancement.slime_block.hint", Item: "slime"},
	{ID: "blaze_powder", Chapter: advChapterHunting, Parents: []string{"blaze_rod"}, NameKey: "advancement.blaze_powder.name", HintKey: "advancement.blaze_powder.hint", Item: "blaze_powder"},
	{ID: "ender_eye", Chapter: advChapterHunting, Parents: []string{"blaze_powder", "ender_pearl"}, NameKey: "advancement.ender_eye.name", HintKey: "advancement.ender_eye.hint", Item: "ender_eye"},
	{ID: "ender_chest", Chapter: advChapterHunting, Parents: []string{"ender_eye", "obsidian"}, NameKey: "advancement.ender_chest.name", HintKey: "advancement.ender_chest.hint", Item: "ender_chest"},
	{ID: "diamond_armor", Chapter: advChapterGear, Parents: []string{"iron_armor", "diamond"}, NameKey: "advancement.diamond_armor.name", HintKey: "advancement.diamond_armor.hint", Item: "diamond_chestplate"},
	{ID: "dye", Chapter: advChapterMagic, Parents: []string{advBloom}, NameKey: "advancement.dye.name", HintKey: "advancement.dye.hint", Item: "red_dye"},
	{ID: "stained_glass", Chapter: advChapterMagic, Parents: []string{"dye", "glass"}, NameKey: "advancement.stained_glass.name", HintKey: "advancement.stained_glass.hint", Item: "blue_stained_glass"},
	{ID: "wave_8", Chapter: advChapterHunting, Parents: []string{advWave5}, NameKey: "advancement.wave_8.name", HintKey: "advancement.wave_8.hint", Event: true, Icon: "netherite_sword", Guide: "dimension_gate"},
	{ID: "tool_level_5", Chapter: advChapterGear, Parents: []string{advToolLevel}, NameKey: "advancement.tool_level_5.name", HintKey: "advancement.tool_level_5.hint", Event: true, Icon: "diamond_pickaxe", Guide: "tinker_table"},
	{ID: "ore_harvest", Chapter: advChapterMagic, Parents: []string{"ore_bloom"}, NameKey: "advancement.ore_harvest.name", HintKey: "advancement.ore_harvest.hint", Event: true, Icon: "iron_ingot", Guide: "magic_botany"},
	{ID: "gem_harvest", Chapter: advChapterMagic, Parents: []string{"gem_bloom"}, NameKey: "advancement.gem_harvest.name", HintKey: "advancement.gem_harvest.hint", Event: true, Icon: "diamond", Guide: "magic_gems"},
	{ID: "ess_harvest", Chapter: advChapterMagic, Parents: []string{"essence_bloom"}, NameKey: "advancement.ess_harvest.name", HintKey: "advancement.ess_harvest.hint", Event: true, Icon: "ender_pearl", Guide: "magic_essence"},
	{ID: "chunks_8", Chapter: advChapterExtra, Parents: []string{advChunk}, NameKey: "advancement.chunks_8.name", HintKey: "advancement.chunks_8.hint", Event: true, Icon: "grass_block"},
	{ID: "chunks_16", Chapter: advChapterExtra, Parents: []string{advChunks8}, NameKey: "advancement.chunks_16.name", HintKey: "advancement.chunks_16.hint", Event: true, Icon: "grass_block"},
	{ID: "project_industry", Chapter: advChapterProject, Parents: []string{"conduit_upgrade_3", "storage_controller", "auto_crafter", "lava_generator"}, NameKey: "advancement.project_industry.name", HintKey: "advancement.project_industry.hint", Research: "project_industry", Icon: "dragonfactory:storage_controller"},
	{ID: "project_hunter", Chapter: advChapterProject, Parents: []string{advWave5, advToolMade, "diamond_armor", "blaze_rod"}, NameKey: "advancement.project_hunter.name", HintKey: "advancement.project_hunter.hint", Research: "project_hunter", Icon: "diamond_sword"},
	{ID: "project_botanist", Chapter: advChapterProject, Parents: []string{"essence_bloom", "dye"}, NameKey: "advancement.project_botanist.name", HintKey: "advancement.project_botanist.hint", Research: "project_botanist", Icon: "dragonfactory:petal_red"},
	{ID: "project_dragon", Chapter: advChapterProject, Parents: []string{"project_industry", "project_hunter", "project_botanist"}, NameKey: "advancement.project_dragon.name", HintKey: "advancement.project_dragon.hint", Research: "project_dragon", Icon: "ender_eye"},
	{ID: advDragon, Chapter: advChapterProject, Parents: []string{"project_dragon"}, NameKey: "advancement.dragon_egg.name", HintKey: "advancement.dragon_egg.hint", Event: true, Icon: "dragon_egg", Guide: "project_dragon"},
	{ID: "wood_hopper", Chapter: advChapterLogistics, NameKey: "advancement.wood_hopper.name", Parents: []string{"chest"}, HintKey: "advancement.wood_hopper.hint", Item: "dragonfactory:wood_hopper"},
	{ID: advLost, Chapter: advChapterExtra, Event: true, NameKey: "advancement.lost_found.name", Parents: []string{"chest"}, HintKey: "advancement.lost_found.hint", Icon: "chest"},
	{ID: advFairyGift, Chapter: advChapterFairy, Event: true, NameKey: "advancement.fairy_gift.name", Parents: []string{"wood"}, HintKey: "advancement.fairy_gift.hint", Icon: "cake"},
	{ID: "fairy_dust", Chapter: advChapterFairy, Parents: []string{advFairyGift}, NameKey: "advancement.fairy_dust.name", HintKey: "advancement.fairy_dust.hint", Item: "dragonfactory:fairy_dust", Guide: "fairy_magic"},
	{ID: "fairy_emerald", Chapter: advChapterFairy, Parents: []string{"fairy_dust"}, NameKey: "advancement.fairy_emerald.name", HintKey: "advancement.fairy_emerald.hint", Item: "dragonfactory:fairy_emerald", Guide: "fairy_magic"},
	{ID: "fairy_board", Chapter: advChapterFairy, Parents: []string{"fairy_dust"}, NameKey: "advancement.fairy_board.name", HintKey: "advancement.fairy_board.hint", Item: "dragonfactory:fairy_board", Guide: "fairy_magic"},
	{ID: advFavourite, Chapter: advChapterExtra, Event: true, NameKey: "advancement.codex_favourite.name", Parents: []string{"wood"}, HintKey: "advancement.codex_favourite.hint", Icon: "book"},
	{ID: "witch_water", Chapter: advChapterFarming, NameKey: "advancement.witch_water.name", Parents: []string{"barrel", advSift}, HintKey: "advancement.witch_water.hint", Item: "dragonfactory:witch_water_bucket", Guide: "sieve"},
	{ID: "drawer_upgrade", Chapter: advChapterLogistics, NameKey: "advancement.drawer_upgrade.name", Parents: []string{"drawer"}, HintKey: "advancement.drawer_upgrade.hint", Item: "dragonfactory:drawer_upgrade_capacity"},
	{ID: advStats, Chapter: advChapterAutomation, Event: true, NameKey: "advancement.stats.name", Parents: []string{"generator"}, HintKey: "advancement.stats.hint", Icon: "clock"},
	{ID: advMachineTier, Chapter: advChapterAutomation, Event: true, NameKey: "advancement.machine_tier.name", Parents: []string{"auto_crafter", "energy_cell", "conduit_upgrade_2"}, HintKey: "advancement.machine_tier.hint", Icon: "dragonfactory:kit_1", Guide: "machine_tier_1"},
	{ID: "config_copier", Chapter: advChapterAutomation, NameKey: "advancement.config_copier.name", Parents: []string{advMachineTier}, HintKey: "advancement.config_copier.hint", Item: "dragonfactory:config_copier"},
}

// advancementIndex is the index in advancementTree of each ID's first
// advancement.
var advancementIndex = func() map[string]int {
	m := map[string]int{}
	for i := range advancementTree {
		if _, ok := m[advancementTree[i].ID]; !ok {
			m[advancementTree[i].ID] = i
		}
	}
	return m
}()

// advancementByID returns an advancement of the tree.
func advancementByID(id string) (*Advancement, bool) {
	if i, ok := advancementIndex[id]; ok {
		return &advancementTree[i], true
	}
	return nil, false
}

// advancementWithAncestors is a with every advancement before it, the
// first ones first.
func advancementWithAncestors(a *Advancement) []*Advancement {
	var out []*Advancement
	seen := map[string]bool{}
	var walk func(a *Advancement)
	walk = func(a *Advancement) {
		if seen[a.ID] {
			return
		}
		seen[a.ID] = true
		for _, id := range a.Parents {
			if p, ok := advancementByID(id); ok {
				walk(p)
			}
		}
		out = append(out, a)
	}
	walk(a)
	return out
}

// advanced reports whether the save has done an advancement.
func (s *session) advanced(id string) bool { return s.state.Advancements[id] }

// advancementOpen reports whether an advancement is not done yet and its
// parents all are.
func (s *session) advancementOpen(a *Advancement) bool {
	if s.advanced(a.ID) {
		return false
	}
	for _, id := range a.Parents {
		if !s.advanced(id) {
			return false
		}
	}
	return true
}

// nextAdvancement is the tracked advancement while it is not done, else the
// first open one in tree order.
func (s *session) nextAdvancement() (*Advancement, bool) {
	if a, ok := advancementByID(s.state.Tracked); ok && !s.advanced(a.ID) {
		return a, true
	}
	for i := range advancementTree {
		if a := &advancementTree[i]; s.advancementOpen(a) {
			return a, true
		}
	}
	return nil, false
}

// track makes the quest panel follow an advancement not done yet; tracking the one
// followed already goes back to the tree order.
func (s *session) track(tx *world.Tx, a *Advancement) {
	if s.state.Tracked == a.ID {
		s.state.Tracked = ""
	} else {
		s.state.Tracked = a.ID
	}
	s.storeState()
	s.updateBoards(tx)
}

// goalResearch is the research an advancement waits on: the one it asks
// for, the one that unlocks the recipe of the item it asks for, or its
// guide.
func goalResearch(a *Advancement) (*Research, bool) {
	if a.Research != "" {
		return researchByID(a.Research)
	}
	if a.Item != "" {
		for i := range researchTree {
			for _, u := range researchTree[i].Unlocks {
				if unlockName(u) == unlockName(a.Item) {
					return &researchTree[i], true
				}
			}
		}
	}
	if a.Guide != "" {
		return researchByID(a.Guide)
	}
	return nil, false
}

// researchStep is the research to do now on the way to r: r itself when it
// is open, else the first open research among what it needs.
func (s *session) researchStep(r *Research) (*Research, bool) {
	if s.researched(r.ID) {
		return nil, false
	}
	if s.researchOpen(r) {
		return r, true
	}
	for _, n := range r.Needs {
		if need, ok := researchByID(n); ok {
			if step, ok := s.researchStep(need); ok {
				return step, true
			}
		}
	}
	return nil, false
}

// advance completes an advancement reported by an event.
func (s *session) advance(tx *world.Tx, id string) {
	if a, ok := advancementByID(id); ok && !s.advanced(id) {
		s.completeAdvancements(tx, []*Advancement{a})
	}
}

// advancementItems are the unlockName of each advancement's Item, by index
// in advancementTree.
var advancementItems = func() []string {
	out := make([]string, len(advancementTree))
	for i, a := range advancementTree {
		if a.Item != "" {
			out[i] = unlockName(a.Item)
		}
	}
	return out
}()

// checkAdvancements completes the item and research advancements met by
// the players in tx.
func (s *session) checkAdvancements(tx *world.Tx) {
	s.checkAdvancementsHeld(tx, inventoryNames(tx))
}

// checkAdvancementsHeld is checkAdvancements with the players' item names
// (inventoryNames). It runs every second.
func (s *session) checkAdvancementsHeld(tx *world.Tx, held []playerItems) {
	var have map[string]bool
	var done []*Advancement
	for i := range advancementTree {
		a := &advancementTree[i]
		if s.advanced(a.ID) {
			continue
		}
		switch {
		case a.Research != "":
			if s.researched(a.Research) {
				done = append(done, a)
			}
		case a.Item != "":
			if have == nil {
				have = heldItemNames(held)
			}
			if have[advancementItems[i]] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		s.completeAdvancements(tx, done)
	}
}

// playerItems are the names of the items in a player's inventory.
type playerItems struct {
	p     *player.Player
	names map[string]bool
}

// inventoryNames are the item names in the inventory of each player in tx.
func inventoryNames(tx *world.Tx) []playerItems {
	var out []playerItems
	for e := range tx.Players() {
		p := e.(*player.Player)
		names := map[string]bool{}
		for _, st := range p.Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			names[name] = true
		}
		out = append(out, playerItems{p: p, names: names})
	}
	return out
}

// heldItemNames are the names of the items any of the players hold.
func heldItemNames(held []playerItems) map[string]bool {
	names := map[string]bool{}
	for _, h := range held {
		for name := range h.names {
			names[name] = true
		}
	}
	return names
}

// The player settings of how an advancement done is told: a toast, a chat
// line, the sound; whether the quest panel shows the next advancement on the
// screen. The vein mining setting is listed with them.
const (
	settingAdvancementToast = "notify_advancement_toast"
	settingAdvancementChat  = "notify_advancement_chat"
	settingAdvancementSound = "notify_advancement_sound"
	settingQuestPanel       = "hud_quest_panel"
)

func init() {
	account.RegisterSetting(
		account.Setting{Key: settingAdvancementToast, NameKey: "setting.advancement.toast", Default: true},
		account.Setting{Key: settingAdvancementChat, NameKey: "setting.advancement.chat", Default: true},
		account.Setting{Key: settingAdvancementSound, NameKey: "setting.advancement.sound", Default: true},
		account.Setting{Key: settingQuestPanel, NameKey: "setting.advancement.panel", Default: true},
		account.Setting{Key: veinDiagonalSetting, NameKey: "setting.vein.diagonal"},
	)
}

// completeAdvancements marks advancements done, whether their parents are
// or not, and tells everyone on the island what they did, as each one's
// settings say.
func (s *session) completeAdvancements(tx *world.Tx, as []*Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var newly []string
	for _, a := range as {
		if !s.advanced(a.ID) {
			s.state.Advancements[a.ID] = true
			newly = append(newly, a.ID)
		}
	}
	s.grantAdvancementAchievements(tx, newly)
	countAll(tx, "adv_done", int64(len(newly)))
	if len(s.state.Advancements) >= len(advancementTree) {
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), achAllAdv)
		}
	}
	s.storeState()
	s.updateBoards(tx)
	for e := range tx.Players() {
		p := e.(*player.Player)
		lang := i18n.Of(p)
		var names []string
		for _, a := range as {
			names = append(names, a.Title(lang))
		}
		if account.On(p, settingAdvancementToast) {
			p.SendToast(i18n.T(lang, "dragonfactory.advancement.completeAdvancements.3"), strings.Join(names, ", "))
		}
		if account.On(p, settingAdvancementChat) {
			p.Message(i18n.T(lang, "dragonfactory.advancement.completeAdvancements.1") + strings.Join(names, ", "))
		}
		if account.On(p, settingAdvancementSound) {
			p.PlaySound(sound.LevelUp{})
		}
	}
}

// Title is the advancement's name in lang.
func (a *Advancement) Title(lang string) string { return i18n.T(lang, a.NameKey) }

// HintText is the advancement's hint in lang.
func (a *Advancement) HintText(lang string) string { return i18n.T(lang, a.HintKey) }

// flatHint is an advancement's hint in lang on one line.
func flatHint(lang string, a *Advancement) string {
	return strings.ReplaceAll(a.HintText(lang), "\n", " ")
}

// advancementIcon is the icon of an advancement in the advancement screen.
func advancementIcon(a *Advancement) string {
	name := a.Icon
	if name == "" {
		name = a.Item
	}
	if it, ok := world.ItemByName(unlockName(name), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}

// advancementStack is the item an advancement is about: the one it asks
// for, else its icon.
func advancementStack(a *Advancement) (item.Stack, bool) {
	for _, name := range []string{a.Item, a.Icon} {
		if name == "" {
			continue
		}
		if it, ok := world.ItemByName(unlockName(name), 0); ok {
			return item.NewStack(it, 1), true
		}
	}
	return item.Stack{}, false
}
