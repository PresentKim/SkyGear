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
// shared by everyone on the island like the researches. The tree is a
// draft.

// Advancement is one step of the advancement tree. It is done by having an
// item, by a research or by an event the game reports (advance).
type Advancement struct {
	ID, Name string
	// Hint says how to do it, one HUD row per line.
	Hint string
	// Parent must be done before it opens (empty for the first).
	Parent string
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
)

// advancementTree is every advancement; the quest panel follows this
// order, so the main path comes first.
var advancementTree = []Advancement{
	{ID: "wood", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.1"), Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.2"), Item: "oak_log"},
	{ID: "planks", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.3"), Parent: "wood", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.4"), Item: "oak_planks"},
	{ID: "crafting_table", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.5"), Parent: "planks", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.6"), Item: "crafting_table"},
	{ID: "research_table", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.7"), Parent: "crafting_table", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.8"), Item: "dragonfactory:research_table"},
	{ID: "first_research", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.9"), Parent: "research_table", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.10"), Research: "stick", Icon: "stick"},
	{ID: "pebble", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.11"), Parent: "first_research", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.12"), Item: "dragonfactory:pebble_stone", Guide: "stick"},
	{ID: "cobble", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.13"), Parent: "pebble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.14"), Item: "cobblestone"},
	{ID: "wood_tools", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.17"), Parent: "first_research", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.18"), Item: "wooden_pickaxe"},
	{ID: "chest", Name: i18n.T(i18n.Default, "advancement.chest.name"), Parent: "first_research", Hint: i18n.T(i18n.Default, "advancement.chest.hint"), Item: "chest"},
	{ID: "stone_axe", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.23"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.24"), Item: "stone_axe"},
	{ID: "fiber", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.25"), Parent: "stone_axe", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.26"), Item: "dragonfactory:bark_fiber", Guide: "stone_tools"},
	{ID: "string", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.27"), Parent: "fiber", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.28"), Item: "string"},
	{ID: "hammer", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.29"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.30"), Item: "dragonfactory:stone_hammer"},
	{ID: "gravel", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.31"), Parent: "hammer", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.32"), Item: "gravel", Guide: "hammer"},
	{ID: "sieve", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.33"), Parent: "string", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.34"), Item: "dragonfactory:sieve"},
	{ID: advSift, Event: true, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.35"), Parent: "sieve", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.36"), Icon: "gravel", Guide: "sieve"},
	{ID: "furnace", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.39"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.40"), Item: "furnace"},
	{ID: "torch", Name: i18n.T(i18n.Default, "advancement.torch.name"), Parent: "furnace", Hint: i18n.T(i18n.Default, "advancement.torch.hint"), Item: "torch"},
	{ID: "composter", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.19"), Parent: "wood_tools", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.20"), Item: "composter"},
	{ID: "barrel", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.21"), Parent: "composter", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.22"), Item: "dragonfactory:barrel_oak"},
	{ID: advGrowTree, Event: true, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.15"), Parent: "wood", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.16"), Icon: "oak_sapling"},
	{ID: "dust", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.37"), Parent: "gravel", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.38"), Item: "dragonfactory:dust", Guide: "hammer"},
	{ID: "clay", Name: i18n.T(i18n.Default, "advancement.clay.name"), Parent: "barrel", Hint: i18n.T(i18n.Default, "advancement.clay.hint"), Item: "clay", Icon: "clay"},
	{ID: "wooden_bucket", Name: i18n.T(i18n.Default, "advancement.wooden_bucket.name"), Parent: "barrel", Hint: i18n.T(i18n.Default, "advancement.wooden_bucket.hint"), Item: "dragonfactory:wooden_bucket"},
	{ID: "glass", Name: i18n.T(i18n.Default, "advancement.glass.name"), Parent: "furnace", Hint: i18n.T(i18n.Default, "advancement.glass.hint"), Item: "glass", Guide: "furnace"},
	{ID: "iron", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.41"), Parent: advSift, Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.42"), Item: "iron_ingot"},
	{ID: "ore_dust", Name: i18n.T(i18n.Default, "advancement.ore_dust.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.ore_dust.hint"), Item: "dragonfactory:metal_dust_iron", Guide: "iron"},
	{ID: "bucket", Name: i18n.T(i18n.Default, "advancement.bucket.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.bucket.hint"), Item: "bucket"},
	{ID: "stone_barrel", Name: i18n.T(i18n.Default, "advancement.stone_barrel.name"), Parent: "bucket", Hint: i18n.T(i18n.Default, "advancement.stone_barrel.hint"), Item: "dragonfactory:barrel_stone", Icon: "dragonfactory:barrel_stone"},
	{ID: "lava", Name: i18n.T(i18n.Default, "advancement.lava.name"), Parent: "stone_barrel", Hint: i18n.T(i18n.Default, "advancement.lava.hint"), Item: "lava_bucket", Icon: "lava_bucket", Guide: "stone_barrel"},
	{ID: "generator", Name: i18n.T(i18n.Default, "advancement.generator.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.generator.hint"), Research: "generator", Icon: "dragonfactory:generator"},
	{ID: advMachine, Event: true, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.43"), Parent: "generator", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.44"), Icon: "dragonfactory:crusher", Guide: "generator"},
	{ID: "laboratory", Name: i18n.T(i18n.Default, "advancement.laboratory.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.laboratory.hint"), Item: "dragonfactory:laboratory"},
	{ID: "lab_engineering", Name: i18n.T(i18n.Default, "advancement.lab_engineering.name"), Parent: "laboratory", Hint: i18n.T(i18n.Default, "advancement.lab_engineering.hint"), Item: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: advEssenceResearch, Event: true, Name: i18n.T(i18n.Default, "advancement.essence_research.name"), Parent: "lab_engineering", Hint: i18n.T(i18n.Default, "advancement.essence_research.hint"), Icon: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: "hopper", Name: i18n.T(i18n.Default, "advancement.hopper.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.hopper.hint"), Research: "hopper", Icon: "hopper"},
	{ID: "energy_conduit", Name: i18n.T(i18n.Default, "advancement.energy_conduit.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.energy_conduit.hint"), Research: "energy_conduit", Icon: "dragonfactory:conduit_energy_1"},
	{ID: "item_conduit", Name: i18n.T(i18n.Default, "advancement.item_conduit.name"), Parent: "hopper", Hint: i18n.T(i18n.Default, "advancement.item_conduit.hint"), Research: "item_conduit", Icon: "dragonfactory:conduit_item_1"},
	{ID: "iron_pickaxe", Name: i18n.T(i18n.Default, "advancement.iron_pickaxe.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.iron_pickaxe.hint"), Item: "iron_pickaxe"},
	{ID: "auto_miner", Name: i18n.T(i18n.Default, "advancement.auto_miner.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.auto_miner.hint"), Research: "auto_miner", Icon: "dragonfactory:auto_miner"},
	{ID: "crusher", Name: i18n.T(i18n.Default, "advancement.crusher.name"), Parent: "auto_miner", Hint: i18n.T(i18n.Default, "advancement.crusher.hint"), Research: "crusher"},
	{ID: "copper", Name: i18n.T(i18n.Default, "advancement.copper.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.copper.hint"), Item: "copper_ingot", Icon: "copper_ingot"},
	{ID: "tin_ingot", Name: i18n.T(i18n.Default, "advancement.tin_ingot.name"), Parent: "copper", Hint: i18n.T(i18n.Default, "advancement.tin_ingot.hint"), Item: "dragonfactory:ingot_tin", Guide: "tin"},
	{ID: "alloy_smelter", Name: i18n.T(i18n.Default, "advancement.alloy_smelter.name"), Parent: "crusher", Hint: i18n.T(i18n.Default, "advancement.alloy_smelter.hint"), Item: "dragonfactory:alloy_smelter"},
	{ID: "bronze_ingot", Name: i18n.T(i18n.Default, "advancement.bronze_ingot.name"), Parent: "tin_ingot", Hint: i18n.T(i18n.Default, "advancement.bronze_ingot.hint"), Item: "dragonfactory:ingot_bronze", Guide: "alloy_smelter"},
	{ID: "lead_ingot", Name: i18n.T(i18n.Default, "advancement.lead_ingot.name"), Parent: "tin_ingot", Hint: i18n.T(i18n.Default, "advancement.lead_ingot.hint"), Item: "dragonfactory:ingot_lead", Guide: "lead"},
	{ID: "energy_cell", Name: i18n.T(i18n.Default, "advancement.energy_cell.name"), Parent: "lead_ingot", Hint: i18n.T(i18n.Default, "advancement.energy_cell.hint"), Item: "dragonfactory:energy_cell"},
	{ID: "gold", Name: i18n.T(i18n.Default, "advancement.gold.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.gold.hint"), Item: "gold_ingot", Icon: "gold_ingot"},
	{ID: "silver_ingot", Name: i18n.T(i18n.Default, "advancement.silver_ingot.name"), Parent: "gold", Hint: i18n.T(i18n.Default, "advancement.silver_ingot.hint"), Item: "dragonfactory:ingot_silver", Guide: "silver"},
	{ID: "electrum_ingot", Name: i18n.T(i18n.Default, "advancement.electrum_ingot.name"), Parent: "silver_ingot", Hint: i18n.T(i18n.Default, "advancement.electrum_ingot.hint"), Item: "dragonfactory:ingot_electrum", Guide: "alloy_smelter"},
	{ID: "nickel_ingot", Name: i18n.T(i18n.Default, "advancement.nickel_ingot.name"), Parent: "gold", Hint: i18n.T(i18n.Default, "advancement.nickel_ingot.hint"), Item: "dragonfactory:ingot_nickel", Guide: "nickel"},
	{ID: "invar_ingot", Name: i18n.T(i18n.Default, "advancement.invar_ingot.name"), Parent: "nickel_ingot", Hint: i18n.T(i18n.Default, "advancement.invar_ingot.hint"), Item: "dragonfactory:ingot_invar", Guide: "alloy_smelter"},
	{ID: "steel_ingot", Name: i18n.T(i18n.Default, "advancement.steel_ingot.name"), Parent: "bronze_ingot", Hint: i18n.T(i18n.Default, "advancement.steel_ingot.hint"), Item: "dragonfactory:ingot_steel", Guide: "alloy_smelter"},
	{ID: "drawer", Name: i18n.T(i18n.Default, "advancement.drawer.name"), Parent: "wood_tools", Hint: i18n.T(i18n.Default, "advancement.drawer.hint"), Research: "drawer", Icon: "dragonfactory:drawer"},
	{ID: "storage_controller", Name: i18n.T(i18n.Default, "advancement.storage_controller.name"), Parent: "drawer", Hint: i18n.T(i18n.Default, "advancement.storage_controller.hint"), Research: "storage_controller", Icon: "dragonfactory:storage_controller"},
	{ID: "storage_link", Name: i18n.T(i18n.Default, "advancement.storage_link.name"), Parent: "storage_controller", Hint: i18n.T(i18n.Default, "advancement.storage_link.hint"), Item: "dragonfactory:storage_link"},
	{ID: "auto_sieve", Name: i18n.T(i18n.Default, "advancement.auto_sieve.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.auto_sieve.hint"), Research: "auto_sieve", Icon: "dragonfactory:auto_sieve"},
	{ID: "electric_furnace", Name: i18n.T(i18n.Default, "advancement.electric_furnace.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.electric_furnace.hint"), Item: "dragonfactory:electric_furnace"},
	{ID: "harvester", Name: i18n.T(i18n.Default, "advancement.harvester.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.harvester.hint"), Research: "harvester", Icon: "dragonfactory:harvester"},
	{ID: "item_vacuum", Name: i18n.T(i18n.Default, "advancement.item_vacuum.name"), Parent: "hopper", Hint: i18n.T(i18n.Default, "advancement.item_vacuum.hint"), Item: "dragonfactory:item_vacuum"},
	{ID: "tree_feller", Name: i18n.T(i18n.Default, "advancement.tree_feller.name"), Parent: "iron_pickaxe", Hint: i18n.T(i18n.Default, "advancement.tree_feller.hint"), Item: "dragonfactory:tree_feller"},
	{ID: "redstone", Name: i18n.T(i18n.Default, "advancement.redstone.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.redstone.hint"), Item: "lever"},
	{ID: "auto_crafter", Name: i18n.T(i18n.Default, "advancement.auto_crafter.name"), Parent: "redstone", Hint: i18n.T(i18n.Default, "advancement.auto_crafter.hint"), Item: "dragonfactory:auto_crafter"},
	{ID: "conduit_upgrade_2", Name: i18n.T(i18n.Default, "advancement.conduit_upgrade_2.name"), Parent: "electrum_ingot", Hint: i18n.T(i18n.Default, "advancement.conduit_upgrade_2.hint"), Item: "dragonfactory:conduit_energy_2"},
	{ID: "conduit_upgrade_3", Name: i18n.T(i18n.Default, "advancement.conduit_upgrade_3.name"), Parent: "steel_ingot", Hint: i18n.T(i18n.Default, "advancement.conduit_upgrade_3.hint"), Item: "dragonfactory:conduit_energy_3"},
	{ID: "mesh_diamond", Name: i18n.T(i18n.Default, "advancement.mesh_diamond.name"), Parent: "gold", Hint: i18n.T(i18n.Default, "advancement.mesh_diamond.hint"), Item: "dragonfactory:mesh_diamond", Icon: "dragonfactory:mesh_diamond"},
	{ID: "essence", Name: i18n.T(i18n.Default, "advancement.essence.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.essence.hint"), Item: "dragonfactory:essence_zombie", Icon: "dragonfactory:essence_zombie", Guide: "iron"},
	{ID: "mob_simulator", Name: i18n.T(i18n.Default, "advancement.mob_simulator.name"), Parent: "essence", Hint: i18n.T(i18n.Default, "advancement.mob_simulator.hint"), Item: "dragonfactory:mob_simulator", Icon: "dragonfactory:mob_simulator"},
	{ID: "ender_pearl", Name: i18n.T(i18n.Default, "advancement.ender_pearl.name"), Parent: "mob_simulator", Hint: i18n.T(i18n.Default, "advancement.ender_pearl.hint"), Item: "ender_pearl", Icon: "ender_pearl", Guide: "mob_simulator"},
	{ID: "blaze_rod", Name: i18n.T(i18n.Default, "advancement.blaze_rod.name"), Parent: "ender_pearl", Hint: i18n.T(i18n.Default, "advancement.blaze_rod.hint"), Item: "blaze_rod", Icon: "blaze_rod", Guide: "mesh_netherite"},
	{ID: "fluid_conduit", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.47"), Parent: "item_conduit", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.48"), Research: "fluid_conduit", Icon: "dragonfactory:conduit_fluid_1"},
	{ID: "fluid_tank", Name: i18n.T(i18n.Default, "advancement.fluid_tank.name"), Parent: "fluid_conduit", Hint: i18n.T(i18n.Default, "advancement.fluid_tank.hint"), Research: "fluid_tank", Icon: "dragonfactory:fluid_tank"},
	{ID: "ore_washer", Name: i18n.T(i18n.Default, "advancement.ore_washer.name"), Parent: "fluid_tank", Hint: i18n.T(i18n.Default, "advancement.ore_washer.hint"), Research: "ore_washer", Icon: "dragonfactory:ore_washer"},
	{ID: "crucible", Name: i18n.T(i18n.Default, "advancement.crucible.name"), Parent: "fluid_tank", Hint: i18n.T(i18n.Default, "advancement.crucible.hint"), Research: "crucible", Icon: "dragonfactory:crucible"},
	{ID: "lava_generator", Name: i18n.T(i18n.Default, "advancement.lava_generator.name"), Parent: "crucible", Hint: i18n.T(i18n.Default, "advancement.lava_generator.hint"), Research: "lava_generator", Icon: "dragonfactory:lava_generator"},
	{ID: "dimension_gate", Name: i18n.T(i18n.Default, "advancement.dimension_gate.name"), Parent: "ender_pearl", Hint: i18n.T(i18n.Default, "advancement.dimension_gate.hint"), Research: "dimension_gate", Icon: "dragonfactory:dimension_gate"},
	{ID: "first_kill", Name: i18n.T(i18n.Default, "advancement.first_kill.name"), Parent: "dimension_gate", Hint: i18n.T(i18n.Default, "advancement.first_kill.hint"), Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: advWave1, Event: true, Name: i18n.T(i18n.Default, "advancement.wave_1.name"), Parent: "dimension_gate", Hint: i18n.T(i18n.Default, "advancement.wave_1.hint"), Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "lab_alchemy", Name: i18n.T(i18n.Default, "advancement.lab_alchemy.name"), Parent: "lab_engineering", Hint: i18n.T(i18n.Default, "advancement.lab_alchemy.hint"), Item: "dragonfactory:research_essence_alchemy", Guide: "laboratory"},
	{ID: "lab_farming", Name: i18n.T(i18n.Default, "advancement.lab_farming.name"), Parent: "lab_engineering", Hint: i18n.T(i18n.Default, "advancement.lab_farming.hint"), Item: "dragonfactory:research_essence_farming", Guide: "laboratory"},
	{ID: "lab_hunting", Name: i18n.T(i18n.Default, "advancement.lab_hunting.name"), Parent: "lab_alchemy", Hint: i18n.T(i18n.Default, "advancement.lab_hunting.hint"), Item: "dragonfactory:research_essence_hunting", Guide: "laboratory"},
	{ID: "lab_magic", Name: i18n.T(i18n.Default, "advancement.lab_magic.name"), Parent: "lab_alchemy", Hint: i18n.T(i18n.Default, "advancement.lab_magic.hint"), Item: "dragonfactory:research_essence_magic", Guide: "laboratory"},
	{ID: "mesh_netherite", Name: i18n.T(i18n.Default, "advancement.mesh_netherite.name"), Parent: "mesh_diamond", Hint: i18n.T(i18n.Default, "advancement.mesh_netherite.hint"), Item: "dragonfactory:mesh_netherite", Icon: "dragonfactory:mesh_netherite"},
	{ID: "magic_sieve", Name: i18n.T(i18n.Default, "advancement.magic_sieve.name"), Parent: "dimension_gate", Hint: i18n.T(i18n.Default, "advancement.magic_sieve.hint"), Research: "magic_sieve", Icon: "dragonfactory:magic_sieve"},
	{ID: advWave5, Event: true, Name: i18n.T(i18n.Default, "advancement.wave_5.name"), Parent: "wave_1", Hint: i18n.T(i18n.Default, "advancement.wave_5.hint"), Icon: "diamond_sword", Guide: "dimension_gate"},
	{ID: "diamond_tools", Name: i18n.T(i18n.Default, "advancement.diamond_tools.name"), Parent: "mesh_diamond", Hint: i18n.T(i18n.Default, "advancement.diamond_tools.hint"), Item: "diamond_pickaxe"},
	{ID: "obsidian", Name: i18n.T(i18n.Default, "advancement.obsidian.name"), Parent: "diamond_tools", Hint: i18n.T(i18n.Default, "advancement.obsidian.hint"), Item: "obsidian", Guide: "diamond_tools"},
	{ID: "tinker_table", Name: i18n.T(i18n.Default, "advancement.tinker_table.name"), Parent: "diamond_tools", Hint: i18n.T(i18n.Default, "advancement.tinker_table.hint"), Research: "tinker_table", Icon: "dragonfactory:tinker_table"},
	{ID: advToolMod, Event: true, Name: i18n.T(i18n.Default, "advancement.tool_modifier.name"), Parent: "tinker_table", Hint: i18n.T(i18n.Default, "advancement.tool_modifier.hint"), Icon: "redstone", Guide: "tinker_table"},
	{ID: advToolLevel, Event: true, Name: i18n.T(i18n.Default, "advancement.tool_level.name"), Parent: advToolMod, Hint: i18n.T(i18n.Default, "advancement.tool_level.hint"), Icon: "experience_bottle", Guide: "tinker_table"},
	{ID: "magic_seed", Name: i18n.T(i18n.Default, "advancement.magic_seed.name"), Parent: advWave1, Hint: i18n.T(i18n.Default, "advancement.magic_seed.hint"), Item: "dragonfactory:seed_flower_red", Guide: "dimension_gate"},
	{ID: advBloom, Event: true, Name: i18n.T(i18n.Default, "advancement.first_bloom.name"), Parent: "magic_seed", Hint: i18n.T(i18n.Default, "advancement.first_bloom.hint"), Icon: "dragonfactory:petal_red"},
	{ID: "ore_bloom", Name: i18n.T(i18n.Default, "advancement.ore_bloom.name"), Parent: advBloom, Hint: i18n.T(i18n.Default, "advancement.ore_bloom.hint"), Item: "dragonfactory:seed_ore_iron"},
	{ID: "gem_bloom", Name: i18n.T(i18n.Default, "advancement.gem_bloom.name"), Parent: "ore_bloom", Hint: i18n.T(i18n.Default, "advancement.gem_bloom.hint"), Item: "dragonfactory:seed_gem_diamond"},
	{ID: "essence_bloom", Name: i18n.T(i18n.Default, "advancement.essence_bloom.name"), Parent: "gem_bloom", Hint: i18n.T(i18n.Default, "advancement.essence_bloom.hint"), Item: "dragonfactory:seed_ess_enderman"},
	{ID: advChunk, Event: true, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.45"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.46"), Icon: "grass_block"},
	{ID: "bread", Name: i18n.T(i18n.Default, "advancement.bread.name"), Parent: "wood_tools", Hint: i18n.T(i18n.Default, "advancement.bread.hint"), Item: "bread"},
	{ID: "shears", Name: i18n.T(i18n.Default, "advancement.shears.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.shears.hint"), Item: "shears"},
	{ID: "iron_armor", Name: i18n.T(i18n.Default, "advancement.iron_armor.name"), Parent: "iron_pickaxe", Hint: i18n.T(i18n.Default, "advancement.iron_armor.hint"), Item: "iron_chestplate"},
	{ID: "lantern", Name: i18n.T(i18n.Default, "advancement.lantern.name"), Parent: "iron_pickaxe", Hint: i18n.T(i18n.Default, "advancement.lantern.hint"), Item: "lantern"},
	{ID: "anvil", Name: i18n.T(i18n.Default, "advancement.anvil.name"), Parent: "lantern", Hint: i18n.T(i18n.Default, "advancement.anvil.hint"), Item: "anvil"},
	{ID: "workbench", Name: i18n.T(i18n.Default, "advancement.workbench.name"), Parent: "iron_pickaxe", Hint: i18n.T(i18n.Default, "advancement.workbench.hint"), Item: "stonecutter_block"},
	{ID: "blast_furnace", Name: i18n.T(i18n.Default, "advancement.blast_furnace.name"), Parent: "iron_pickaxe", Hint: i18n.T(i18n.Default, "advancement.blast_furnace.hint"), Item: "blast_furnace"},
	{ID: "bow", Name: i18n.T(i18n.Default, "advancement.bow.name"), Parent: "string", Hint: i18n.T(i18n.Default, "advancement.bow.hint"), Item: "bow"},
	{ID: "copper_gear", Name: i18n.T(i18n.Default, "advancement.copper_gear.name"), Parent: "copper", Hint: i18n.T(i18n.Default, "advancement.copper_gear.hint"), Item: "copper_pickaxe"},
	{ID: "compass", Name: i18n.T(i18n.Default, "advancement.compass.name"), Parent: "redstone", Hint: i18n.T(i18n.Default, "advancement.compass.hint"), Item: "compass"},
	{ID: "golden_carrot", Name: i18n.T(i18n.Default, "advancement.golden_carrot.name"), Parent: "gold", Hint: i18n.T(i18n.Default, "advancement.golden_carrot.hint"), Item: "golden_carrot"},
	{ID: "diamond_block", Name: i18n.T(i18n.Default, "advancement.diamond_block.name"), Parent: "mesh_diamond", Hint: i18n.T(i18n.Default, "advancement.diamond_block.hint"), Item: "diamond_block"},
	{ID: "sandstone", Name: i18n.T(i18n.Default, "advancement.sandstone.name"), Parent: "furnace", Hint: i18n.T(i18n.Default, "advancement.sandstone.hint"), Item: "sandstone"},
	{ID: "quartz_block", Name: i18n.T(i18n.Default, "advancement.quartz_block.name"), Parent: "furnace", Hint: i18n.T(i18n.Default, "advancement.quartz_block.hint"), Item: "quartz_block"},
	{ID: "amethyst_block", Name: i18n.T(i18n.Default, "advancement.amethyst_block.name"), Parent: "glass", Hint: i18n.T(i18n.Default, "advancement.amethyst_block.hint"), Item: "amethyst_block"},
	{ID: "bone_block", Name: i18n.T(i18n.Default, "advancement.bone_block.name"), Parent: "mob_simulator", Hint: i18n.T(i18n.Default, "advancement.bone_block.hint"), Item: "bone_block"},
	{ID: "slime_block", Name: i18n.T(i18n.Default, "advancement.slime_block.name"), Parent: "mob_simulator", Hint: i18n.T(i18n.Default, "advancement.slime_block.hint"), Item: "slime"},
	{ID: "blaze_powder", Name: i18n.T(i18n.Default, "advancement.blaze_powder.name"), Parent: "blaze_rod", Hint: i18n.T(i18n.Default, "advancement.blaze_powder.hint"), Item: "blaze_powder"},
	{ID: "ender_eye", Name: i18n.T(i18n.Default, "advancement.ender_eye.name"), Parent: "blaze_powder", Hint: i18n.T(i18n.Default, "advancement.ender_eye.hint"), Item: "ender_eye"},
	{ID: "ender_chest", Name: i18n.T(i18n.Default, "advancement.ender_chest.name"), Parent: "ender_eye", Hint: i18n.T(i18n.Default, "advancement.ender_chest.hint"), Item: "ender_chest"},
	{ID: "diamond_armor", Name: i18n.T(i18n.Default, "advancement.diamond_armor.name"), Parent: "diamond_tools", Hint: i18n.T(i18n.Default, "advancement.diamond_armor.hint"), Item: "diamond_chestplate"},
	{ID: "dye", Name: i18n.T(i18n.Default, "advancement.dye.name"), Parent: "first_bloom", Hint: i18n.T(i18n.Default, "advancement.dye.hint"), Item: "red_dye"},
	{ID: "stained_glass", Name: i18n.T(i18n.Default, "advancement.stained_glass.name"), Parent: "dye", Hint: i18n.T(i18n.Default, "advancement.stained_glass.hint"), Item: "blue_stained_glass"},
	{ID: "wave_3", Name: i18n.T(i18n.Default, "advancement.wave_3.name"), Parent: "wave_1", Hint: i18n.T(i18n.Default, "advancement.wave_3.hint"), Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "wave_8", Name: i18n.T(i18n.Default, "advancement.wave_8.name"), Parent: "wave_5", Hint: i18n.T(i18n.Default, "advancement.wave_8.hint"), Event: true, Icon: "netherite_sword", Guide: "dimension_gate"},
	{ID: "tool_made", Name: i18n.T(i18n.Default, "advancement.tool_made.name"), Parent: "tinker_table", Hint: i18n.T(i18n.Default, "advancement.tool_made.hint"), Event: true, Icon: "iron_pickaxe", Guide: "tinker_table"},
	{ID: "tool_level_5", Name: i18n.T(i18n.Default, "advancement.tool_level_5.name"), Parent: "tool_level", Hint: i18n.T(i18n.Default, "advancement.tool_level_5.hint"), Event: true, Icon: "diamond_pickaxe", Guide: "tinker_table"},
	{ID: "ore_harvest", Name: i18n.T(i18n.Default, "advancement.ore_harvest.name"), Parent: "ore_bloom", Hint: i18n.T(i18n.Default, "advancement.ore_harvest.hint"), Event: true, Icon: "iron_ingot", Guide: "magic_botany"},
	{ID: "gem_harvest", Name: i18n.T(i18n.Default, "advancement.gem_harvest.name"), Parent: "gem_bloom", Hint: i18n.T(i18n.Default, "advancement.gem_harvest.hint"), Event: true, Icon: "diamond", Guide: "magic_gems"},
	{ID: "ess_harvest", Name: i18n.T(i18n.Default, "advancement.ess_harvest.name"), Parent: "essence_bloom", Hint: i18n.T(i18n.Default, "advancement.ess_harvest.hint"), Event: true, Icon: "ender_pearl", Guide: "magic_essence"},
	{ID: "chunks_8", Name: i18n.T(i18n.Default, "advancement.chunks_8.name"), Parent: "chunk", Hint: i18n.T(i18n.Default, "advancement.chunks_8.hint"), Event: true, Icon: "grass_block"},
	{ID: "chunks_16", Name: i18n.T(i18n.Default, "advancement.chunks_16.name"), Parent: "chunks_8", Hint: i18n.T(i18n.Default, "advancement.chunks_16.hint"), Event: true, Icon: "grass_block"},
	{ID: "project_industry", Name: i18n.T(i18n.Default, "advancement.project_industry.name"), Parent: "conduit_upgrade_3", Hint: i18n.T(i18n.Default, "advancement.project_industry.hint"), Research: "project_industry", Icon: "dragonfactory:storage_controller"},
	{ID: "project_hunter", Name: i18n.T(i18n.Default, "advancement.project_hunter.name"), Parent: "project_industry", Hint: i18n.T(i18n.Default, "advancement.project_hunter.hint"), Research: "project_hunter", Icon: "diamond_sword"},
	{ID: "project_botanist", Name: i18n.T(i18n.Default, "advancement.project_botanist.name"), Parent: "project_hunter", Hint: i18n.T(i18n.Default, "advancement.project_botanist.hint"), Research: "project_botanist", Icon: "dragonfactory:petal_red"},
	{ID: "project_dragon", Name: i18n.T(i18n.Default, "advancement.project_dragon.name"), Parent: "project_botanist", Hint: i18n.T(i18n.Default, "advancement.project_dragon.hint"), Research: "project_dragon", Icon: "ender_eye"},
	{ID: advDragon, Name: i18n.T(i18n.Default, "advancement.dragon_egg.name"), Parent: "project_dragon", Hint: i18n.T(i18n.Default, "advancement.dragon_egg.hint"), Event: true, Icon: "dragon_egg", Guide: "project_dragon"},
}

// advancementByID returns an advancement of the tree.
func advancementByID(id string) (*Advancement, bool) {
	for i := range advancementTree {
		if advancementTree[i].ID == id {
			return &advancementTree[i], true
		}
	}
	return nil, false
}

// advanced reports whether the save has done an advancement.
func (s *session) advanced(id string) bool { return s.state.Advancements[id] }

// advancementOpen reports whether an advancement is not done yet and its
// parent is.
func (s *session) advancementOpen(a *Advancement) bool {
	return !s.advanced(a.ID) && (a.Parent == "" || s.advanced(a.Parent))
}

// nextAdvancement is the tracked advancement while it is open, else the
// first open one in tree order.
func (s *session) nextAdvancement() (*Advancement, bool) {
	if a, ok := advancementByID(s.state.Tracked); ok && s.advancementOpen(a) {
		return a, true
	}
	for i := range advancementTree {
		if a := &advancementTree[i]; s.advancementOpen(a) {
			return a, true
		}
	}
	return nil, false
}

// track makes the quest panel follow an open advancement; tracking the one
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

// checkAdvancements completes the item and research advancements met by
// the players in tx. It runs every second.
func (s *session) checkAdvancements(tx *world.Tx) {
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
				have = heldItemNames(tx)
			}
			if have[unlockName(a.Item)] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		s.completeAdvancements(tx, done)
	}
}

// heldItemNames are the names of the items in the inventories of the
// players in tx.
func heldItemNames(tx *world.Tx) map[string]bool {
	names := map[string]bool{}
	for e := range tx.Players() {
		for _, st := range e.(*player.Player).Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			names[name] = true
		}
	}
	return names
}

// completeAdvancements marks advancements done, with the ones before them
// (an item used up within a second would otherwise leave a gap), and tells
// everyone on the island what they did .
func (s *session) completeAdvancements(tx *world.Tx, as []*Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var names, newly []string
	for _, a := range as {
		for cur, ok := a, true; ok && !s.advanced(cur.ID); cur, ok = advancementByID(cur.Parent) {
			s.state.Advancements[cur.ID] = true
			newly = append(newly, cur.ID)
		}
		names = append(names, a.Name)
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
	msg := i18n.T(i18n.Default, "dragonfactory.advancement.completeAdvancements.1") + strings.Join(names, ", ")
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.SendToast(i18n.T(i18n.Of(p), "dragonfactory.advancement.completeAdvancements.3"), strings.Join(names, ", "))
		p.Message(msg)
		p.PlaySound(sound.LevelUp{})
	}
}

// flatHint is an advancement's hint on one line.
func flatHint(a *Advancement) string { return strings.ReplaceAll(a.Hint, "\n", " ") }

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
