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
	ID string
	// NameKey and HintKey are the catalog keys of its name and of the hint
	// that says how to do it, one HUD row per line (Title, HintText).
	NameKey, HintKey string
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
	{ID: "wood", NameKey: "dragonfactory.advancement.init.1", HintKey: "dragonfactory.advancement.init.2", Item: "oak_log"},
	{ID: "planks", NameKey: "dragonfactory.advancement.init.3", Parent: "wood", HintKey: "dragonfactory.advancement.init.4", Item: "oak_planks"},
	{ID: "crafting_table", NameKey: "dragonfactory.advancement.init.5", Parent: "planks", HintKey: "dragonfactory.advancement.init.6", Item: "crafting_table"},
	{ID: "research_table", NameKey: "dragonfactory.advancement.init.7", Parent: "crafting_table", HintKey: "dragonfactory.advancement.init.8", Item: "dragonfactory:research_table"},
	{ID: "first_research", NameKey: "dragonfactory.advancement.init.9", Parent: "research_table", HintKey: "dragonfactory.advancement.init.10", Research: "stick", Icon: "stick"},
	{ID: "pebble", NameKey: "dragonfactory.advancement.init.11", Parent: "first_research", HintKey: "dragonfactory.advancement.init.12", Item: "dragonfactory:pebble_stone", Guide: "stick"},
	{ID: "cobble", NameKey: "dragonfactory.advancement.init.13", Parent: "pebble", HintKey: "dragonfactory.advancement.init.14", Item: "cobblestone"},
	{ID: "wood_tools", NameKey: "dragonfactory.advancement.init.17", Parent: "first_research", HintKey: "dragonfactory.advancement.init.18", Item: "wooden_pickaxe"},
	{ID: "chest", NameKey: "advancement.chest.name", Parent: "first_research", HintKey: "advancement.chest.hint", Item: "chest"},
	{ID: "stone_axe", NameKey: "dragonfactory.advancement.init.23", Parent: "cobble", HintKey: "dragonfactory.advancement.init.24", Item: "stone_axe"},
	{ID: "fiber", NameKey: "dragonfactory.advancement.init.25", Parent: "stone_axe", HintKey: "dragonfactory.advancement.init.26", Item: "dragonfactory:bark_fiber", Guide: "stone_tools"},
	{ID: "string", NameKey: "dragonfactory.advancement.init.27", Parent: "fiber", HintKey: "dragonfactory.advancement.init.28", Item: "string"},
	{ID: "hammer", NameKey: "dragonfactory.advancement.init.29", Parent: "cobble", HintKey: "dragonfactory.advancement.init.30", Item: "dragonfactory:stone_hammer"},
	{ID: "gravel", NameKey: "dragonfactory.advancement.init.31", Parent: "hammer", HintKey: "dragonfactory.advancement.init.32", Item: "gravel", Guide: "hammer"},
	{ID: "sieve", NameKey: "dragonfactory.advancement.init.33", Parent: "string", HintKey: "dragonfactory.advancement.init.34", Item: "dragonfactory:sieve"},
	{ID: advSift, Event: true, NameKey: "dragonfactory.advancement.init.35", Parent: "sieve", HintKey: "dragonfactory.advancement.init.36", Icon: "gravel", Guide: "sieve"},
	{ID: "furnace", NameKey: "dragonfactory.advancement.init.39", Parent: "cobble", HintKey: "dragonfactory.advancement.init.40", Item: "furnace"},
	{ID: "torch", NameKey: "advancement.torch.name", Parent: "furnace", HintKey: "advancement.torch.hint", Item: "torch"},
	{ID: "composter", NameKey: "dragonfactory.advancement.init.19", Parent: "wood_tools", HintKey: "dragonfactory.advancement.init.20", Item: "composter"},
	{ID: "barrel", NameKey: "dragonfactory.advancement.init.21", Parent: "composter", HintKey: "dragonfactory.advancement.init.22", Item: "dragonfactory:barrel_oak"},
	{ID: advGrowTree, Event: true, NameKey: "dragonfactory.advancement.init.15", Parent: "wood", HintKey: "dragonfactory.advancement.init.16", Icon: "oak_sapling"},
	{ID: "dust", NameKey: "dragonfactory.advancement.init.37", Parent: "gravel", HintKey: "dragonfactory.advancement.init.38", Item: "dragonfactory:dust", Guide: "hammer"},
	{ID: "clay", NameKey: "advancement.clay.name", Parent: "barrel", HintKey: "advancement.clay.hint", Item: "clay", Icon: "clay"},
	{ID: "wooden_bucket", NameKey: "advancement.wooden_bucket.name", Parent: "barrel", HintKey: "advancement.wooden_bucket.hint", Item: "dragonfactory:wooden_bucket"},
	{ID: "glass", NameKey: "advancement.glass.name", Parent: "furnace", HintKey: "advancement.glass.hint", Item: "glass", Guide: "furnace"},
	{ID: "iron", NameKey: "dragonfactory.advancement.init.41", Parent: advSift, HintKey: "dragonfactory.advancement.init.42", Item: "iron_ingot"},
	{ID: "ore_dust", NameKey: "advancement.ore_dust.name", Parent: "iron", HintKey: "advancement.ore_dust.hint", Item: "dragonfactory:metal_dust_iron", Guide: "iron"},
	{ID: "bucket", NameKey: "advancement.bucket.name", Parent: "iron", HintKey: "advancement.bucket.hint", Item: "bucket"},
	{ID: "stone_barrel", NameKey: "advancement.stone_barrel.name", Parent: "bucket", HintKey: "advancement.stone_barrel.hint", Item: "dragonfactory:barrel_stone", Icon: "dragonfactory:barrel_stone"},
	{ID: "lava", NameKey: "advancement.lava.name", Parent: "stone_barrel", HintKey: "advancement.lava.hint", Item: "lava_bucket", Icon: "lava_bucket", Guide: "stone_barrel"},
	{ID: "generator", NameKey: "advancement.generator.name", Parent: "iron", HintKey: "advancement.generator.hint", Research: "generator", Icon: "dragonfactory:generator"},
	{ID: advMachine, Event: true, NameKey: "dragonfactory.advancement.init.43", Parent: "generator", HintKey: "dragonfactory.advancement.init.44", Icon: "dragonfactory:crusher", Guide: "generator"},
	{ID: "laboratory", NameKey: "advancement.laboratory.name", Parent: "generator", HintKey: "advancement.laboratory.hint", Item: "dragonfactory:laboratory"},
	{ID: "lab_engineering", NameKey: "advancement.lab_engineering.name", Parent: "laboratory", HintKey: "advancement.lab_engineering.hint", Item: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: advEssenceResearch, Event: true, NameKey: "advancement.essence_research.name", Parent: "lab_engineering", HintKey: "advancement.essence_research.hint", Icon: "dragonfactory:research_essence_engineering", Guide: "laboratory"},
	{ID: "hopper", NameKey: "advancement.hopper.name", Parent: "iron", HintKey: "advancement.hopper.hint", Research: "hopper", Icon: "hopper"},
	{ID: "energy_conduit", NameKey: "advancement.energy_conduit.name", Parent: "generator", HintKey: "advancement.energy_conduit.hint", Research: "energy_conduit", Icon: "dragonfactory:conduit_energy_1"},
	{ID: "item_conduit", NameKey: "advancement.item_conduit.name", Parent: "hopper", HintKey: "advancement.item_conduit.hint", Research: "item_conduit", Icon: "dragonfactory:conduit_item_1"},
	{ID: "iron_pickaxe", NameKey: "advancement.iron_pickaxe.name", Parent: "iron", HintKey: "advancement.iron_pickaxe.hint", Item: "iron_pickaxe"},
	{ID: "auto_miner", NameKey: "advancement.auto_miner.name", Parent: "generator", HintKey: "advancement.auto_miner.hint", Research: "auto_miner", Icon: "dragonfactory:auto_miner"},
	{ID: "crusher", NameKey: "advancement.crusher.name", Parent: "auto_miner", HintKey: "advancement.crusher.hint", Research: "crusher"},
	{ID: "copper", NameKey: "advancement.copper.name", Parent: "iron", HintKey: "advancement.copper.hint", Item: "copper_ingot", Icon: "copper_ingot"},
	{ID: "tin_ingot", NameKey: "advancement.tin_ingot.name", Parent: "copper", HintKey: "advancement.tin_ingot.hint", Item: "dragonfactory:ingot_tin", Guide: "tin"},
	{ID: "alloy_smelter", NameKey: "advancement.alloy_smelter.name", Parent: "crusher", HintKey: "advancement.alloy_smelter.hint", Item: "dragonfactory:alloy_smelter"},
	{ID: "bronze_ingot", NameKey: "advancement.bronze_ingot.name", Parent: "tin_ingot", HintKey: "advancement.bronze_ingot.hint", Item: "dragonfactory:ingot_bronze", Guide: "alloy_smelter"},
	{ID: "lead_ingot", NameKey: "advancement.lead_ingot.name", Parent: "tin_ingot", HintKey: "advancement.lead_ingot.hint", Item: "dragonfactory:ingot_lead", Guide: "lead"},
	{ID: "energy_cell", NameKey: "advancement.energy_cell.name", Parent: "lead_ingot", HintKey: "advancement.energy_cell.hint", Item: "dragonfactory:energy_cell"},
	{ID: "gold", NameKey: "advancement.gold.name", Parent: "iron", HintKey: "advancement.gold.hint", Item: "gold_ingot", Icon: "gold_ingot"},
	{ID: "silver_ingot", NameKey: "advancement.silver_ingot.name", Parent: "gold", HintKey: "advancement.silver_ingot.hint", Item: "dragonfactory:ingot_silver", Guide: "silver"},
	{ID: "electrum_ingot", NameKey: "advancement.electrum_ingot.name", Parent: "silver_ingot", HintKey: "advancement.electrum_ingot.hint", Item: "dragonfactory:ingot_electrum", Guide: "alloy_smelter"},
	{ID: "nickel_ingot", NameKey: "advancement.nickel_ingot.name", Parent: "gold", HintKey: "advancement.nickel_ingot.hint", Item: "dragonfactory:ingot_nickel", Guide: "nickel"},
	{ID: "invar_ingot", NameKey: "advancement.invar_ingot.name", Parent: "nickel_ingot", HintKey: "advancement.invar_ingot.hint", Item: "dragonfactory:ingot_invar", Guide: "alloy_smelter"},
	{ID: "steel_ingot", NameKey: "advancement.steel_ingot.name", Parent: "bronze_ingot", HintKey: "advancement.steel_ingot.hint", Item: "dragonfactory:ingot_steel", Guide: "alloy_smelter"},
	{ID: "drawer", NameKey: "advancement.drawer.name", Parent: "wood_tools", HintKey: "advancement.drawer.hint", Research: "drawer", Icon: "dragonfactory:drawer"},
	{ID: "storage_controller", NameKey: "advancement.storage_controller.name", Parent: "drawer", HintKey: "advancement.storage_controller.hint", Research: "storage_controller", Icon: "dragonfactory:storage_controller"},
	{ID: "storage_link", NameKey: "advancement.storage_link.name", Parent: "storage_controller", HintKey: "advancement.storage_link.hint", Item: "dragonfactory:storage_link"},
	{ID: "auto_sieve", NameKey: "advancement.auto_sieve.name", Parent: "generator", HintKey: "advancement.auto_sieve.hint", Research: "auto_sieve", Icon: "dragonfactory:auto_sieve"},
	{ID: "electric_furnace", NameKey: "advancement.electric_furnace.name", Parent: "generator", HintKey: "advancement.electric_furnace.hint", Item: "dragonfactory:electric_furnace"},
	{ID: "harvester", NameKey: "advancement.harvester.name", Parent: "generator", HintKey: "advancement.harvester.hint", Research: "harvester", Icon: "dragonfactory:harvester"},
	{ID: "item_vacuum", NameKey: "advancement.item_vacuum.name", Parent: "hopper", HintKey: "advancement.item_vacuum.hint", Item: "dragonfactory:item_vacuum"},
	{ID: "tree_feller", NameKey: "advancement.tree_feller.name", Parent: "iron_pickaxe", HintKey: "advancement.tree_feller.hint", Item: "dragonfactory:tree_feller"},
	{ID: "redstone", NameKey: "advancement.redstone.name", Parent: "generator", HintKey: "advancement.redstone.hint", Item: "lever"},
	{ID: "auto_crafter", NameKey: "advancement.auto_crafter.name", Parent: "redstone", HintKey: "advancement.auto_crafter.hint", Item: "dragonfactory:auto_crafter"},
	{ID: "conduit_upgrade_2", NameKey: "advancement.conduit_upgrade_2.name", Parent: "electrum_ingot", HintKey: "advancement.conduit_upgrade_2.hint", Item: "dragonfactory:conduit_energy_2"},
	{ID: "conduit_upgrade_3", NameKey: "advancement.conduit_upgrade_3.name", Parent: "steel_ingot", HintKey: "advancement.conduit_upgrade_3.hint", Item: "dragonfactory:conduit_energy_3"},
	{ID: "mesh_diamond", NameKey: "advancement.mesh_diamond.name", Parent: "gold", HintKey: "advancement.mesh_diamond.hint", Item: "dragonfactory:mesh_diamond", Icon: "dragonfactory:mesh_diamond"},
	{ID: "essence", NameKey: "advancement.essence.name", Parent: "iron", HintKey: "advancement.essence.hint", Item: "dragonfactory:essence_zombie", Icon: "dragonfactory:essence_zombie", Guide: "iron"},
	{ID: "mob_simulator", NameKey: "advancement.mob_simulator.name", Parent: "essence", HintKey: "advancement.mob_simulator.hint", Item: "dragonfactory:mob_simulator", Icon: "dragonfactory:mob_simulator"},
	{ID: "ender_pearl", NameKey: "advancement.ender_pearl.name", Parent: "mob_simulator", HintKey: "advancement.ender_pearl.hint", Item: "ender_pearl", Icon: "ender_pearl", Guide: "mob_simulator"},
	{ID: "blaze_rod", NameKey: "advancement.blaze_rod.name", Parent: "ender_pearl", HintKey: "advancement.blaze_rod.hint", Item: "blaze_rod", Icon: "blaze_rod", Guide: "mesh_netherite"},
	{ID: "fluid_conduit", NameKey: "dragonfactory.advancement.init.47", Parent: "item_conduit", HintKey: "dragonfactory.advancement.init.48", Research: "fluid_conduit", Icon: "dragonfactory:conduit_fluid_1"},
	{ID: "fluid_tank", NameKey: "advancement.fluid_tank.name", Parent: "fluid_conduit", HintKey: "advancement.fluid_tank.hint", Research: "fluid_tank", Icon: "dragonfactory:fluid_tank"},
	{ID: "ore_washer", NameKey: "advancement.ore_washer.name", Parent: "fluid_tank", HintKey: "advancement.ore_washer.hint", Research: "ore_washer", Icon: "dragonfactory:ore_washer"},
	{ID: "crucible", NameKey: "advancement.crucible.name", Parent: "fluid_tank", HintKey: "advancement.crucible.hint", Research: "crucible", Icon: "dragonfactory:crucible"},
	{ID: "lava_generator", NameKey: "advancement.lava_generator.name", Parent: "crucible", HintKey: "advancement.lava_generator.hint", Research: "lava_generator", Icon: "dragonfactory:lava_generator"},
	{ID: "dimension_gate", NameKey: "advancement.dimension_gate.name", Parent: "ender_pearl", HintKey: "advancement.dimension_gate.hint", Research: "dimension_gate", Icon: "dragonfactory:dimension_gate"},
	{ID: "first_kill", NameKey: "advancement.first_kill.name", Parent: "dimension_gate", HintKey: "advancement.first_kill.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: advWave1, Event: true, NameKey: "advancement.wave_1.name", Parent: "dimension_gate", HintKey: "advancement.wave_1.hint", Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "lab_alchemy", NameKey: "advancement.lab_alchemy.name", Parent: "lab_engineering", HintKey: "advancement.lab_alchemy.hint", Item: "dragonfactory:research_essence_alchemy", Guide: "laboratory"},
	{ID: "lab_farming", NameKey: "advancement.lab_farming.name", Parent: "lab_engineering", HintKey: "advancement.lab_farming.hint", Item: "dragonfactory:research_essence_farming", Guide: "laboratory"},
	{ID: "lab_hunting", NameKey: "advancement.lab_hunting.name", Parent: "lab_alchemy", HintKey: "advancement.lab_hunting.hint", Item: "dragonfactory:research_essence_hunting", Guide: "laboratory"},
	{ID: "lab_magic", NameKey: "advancement.lab_magic.name", Parent: "lab_alchemy", HintKey: "advancement.lab_magic.hint", Item: "dragonfactory:research_essence_magic", Guide: "laboratory"},
	{ID: "mesh_netherite", NameKey: "advancement.mesh_netherite.name", Parent: "mesh_diamond", HintKey: "advancement.mesh_netherite.hint", Item: "dragonfactory:mesh_netherite", Icon: "dragonfactory:mesh_netherite"},
	{ID: "magic_sieve", NameKey: "advancement.magic_sieve.name", Parent: "dimension_gate", HintKey: "advancement.magic_sieve.hint", Research: "magic_sieve", Icon: "dragonfactory:magic_sieve"},
	{ID: advWave5, Event: true, NameKey: "advancement.wave_5.name", Parent: "wave_1", HintKey: "advancement.wave_5.hint", Icon: "diamond_sword", Guide: "dimension_gate"},
	{ID: "diamond_tools", NameKey: "advancement.diamond_tools.name", Parent: "mesh_diamond", HintKey: "advancement.diamond_tools.hint", Item: "diamond_pickaxe"},
	{ID: "obsidian", NameKey: "advancement.obsidian.name", Parent: "diamond_tools", HintKey: "advancement.obsidian.hint", Item: "obsidian", Guide: "diamond_tools"},
	{ID: "tinker_table", NameKey: "advancement.tinker_table.name", Parent: "diamond_tools", HintKey: "advancement.tinker_table.hint", Research: "tinker_table", Icon: "dragonfactory:tinker_table"},
	{ID: advToolMod, Event: true, NameKey: "advancement.tool_modifier.name", Parent: "tinker_table", HintKey: "advancement.tool_modifier.hint", Icon: "redstone", Guide: "tinker_table"},
	{ID: advToolLevel, Event: true, NameKey: "advancement.tool_level.name", Parent: advToolMod, HintKey: "advancement.tool_level.hint", Icon: "experience_bottle", Guide: "tinker_table"},
	{ID: "magic_seed", NameKey: "advancement.magic_seed.name", Parent: advWave1, HintKey: "advancement.magic_seed.hint", Item: "dragonfactory:seed_flower_red", Guide: "dimension_gate"},
	{ID: advBloom, Event: true, NameKey: "advancement.first_bloom.name", Parent: "magic_seed", HintKey: "advancement.first_bloom.hint", Icon: "dragonfactory:petal_red"},
	{ID: "ore_bloom", NameKey: "advancement.ore_bloom.name", Parent: advBloom, HintKey: "advancement.ore_bloom.hint", Item: "dragonfactory:seed_ore_iron"},
	{ID: "gem_bloom", NameKey: "advancement.gem_bloom.name", Parent: "ore_bloom", HintKey: "advancement.gem_bloom.hint", Item: "dragonfactory:seed_gem_diamond"},
	{ID: "essence_bloom", NameKey: "advancement.essence_bloom.name", Parent: "gem_bloom", HintKey: "advancement.essence_bloom.hint", Item: "dragonfactory:seed_ess_enderman"},
	{ID: advChunk, Event: true, NameKey: "dragonfactory.advancement.init.45", Parent: "cobble", HintKey: "dragonfactory.advancement.init.46", Icon: "grass_block"},
	{ID: "seeds", NameKey: "advancement.seeds.name", Parent: advSift, HintKey: "advancement.seeds.hint", Item: "wheat_seeds", Guide: "sieve"},
	{ID: "wheat", NameKey: "advancement.wheat.name", Parent: "seeds", HintKey: "advancement.wheat.hint", Item: "wheat", Guide: "wood_tools"},
	{ID: "bread", NameKey: "advancement.bread.name", Parent: "wheat", HintKey: "advancement.bread.hint", Item: "bread"},
	{ID: "shears", NameKey: "advancement.shears.name", Parent: "iron", HintKey: "advancement.shears.hint", Item: "shears"},
	{ID: "iron_armor", NameKey: "advancement.iron_armor.name", Parent: "iron_pickaxe", HintKey: "advancement.iron_armor.hint", Item: "iron_chestplate"},
	{ID: "lantern", NameKey: "advancement.lantern.name", Parent: "iron_pickaxe", HintKey: "advancement.lantern.hint", Item: "lantern"},
	{ID: "anvil", NameKey: "advancement.anvil.name", Parent: "lantern", HintKey: "advancement.anvil.hint", Item: "anvil"},
	{ID: "workbench", NameKey: "advancement.workbench.name", Parent: "iron_pickaxe", HintKey: "advancement.workbench.hint", Item: "stonecutter_block"},
	{ID: "blast_furnace", NameKey: "advancement.blast_furnace.name", Parent: "iron_pickaxe", HintKey: "advancement.blast_furnace.hint", Item: "blast_furnace"},
	{ID: "bow", NameKey: "advancement.bow.name", Parent: "string", HintKey: "advancement.bow.hint", Item: "bow"},
	{ID: "copper_gear", NameKey: "advancement.copper_gear.name", Parent: "copper", HintKey: "advancement.copper_gear.hint", Item: "copper_pickaxe"},
	{ID: "compass", NameKey: "advancement.compass.name", Parent: "redstone", HintKey: "advancement.compass.hint", Item: "compass"},
	{ID: "golden_carrot", NameKey: "advancement.golden_carrot.name", Parent: "gold", HintKey: "advancement.golden_carrot.hint", Item: "golden_carrot"},
	{ID: "diamond_block", NameKey: "advancement.diamond_block.name", Parent: "mesh_diamond", HintKey: "advancement.diamond_block.hint", Item: "diamond_block"},
	{ID: "sandstone", NameKey: "advancement.sandstone.name", Parent: "furnace", HintKey: "advancement.sandstone.hint", Item: "sandstone"},
	{ID: "quartz_block", NameKey: "advancement.quartz_block.name", Parent: "furnace", HintKey: "advancement.quartz_block.hint", Item: "quartz_block"},
	{ID: "amethyst_block", NameKey: "advancement.amethyst_block.name", Parent: "glass", HintKey: "advancement.amethyst_block.hint", Item: "amethyst_block"},
	{ID: "bone_block", NameKey: "advancement.bone_block.name", Parent: "mob_simulator", HintKey: "advancement.bone_block.hint", Item: "bone_block"},
	{ID: "slime_block", NameKey: "advancement.slime_block.name", Parent: "mob_simulator", HintKey: "advancement.slime_block.hint", Item: "slime"},
	{ID: "blaze_powder", NameKey: "advancement.blaze_powder.name", Parent: "blaze_rod", HintKey: "advancement.blaze_powder.hint", Item: "blaze_powder"},
	{ID: "ender_eye", NameKey: "advancement.ender_eye.name", Parent: "blaze_powder", HintKey: "advancement.ender_eye.hint", Item: "ender_eye"},
	{ID: "ender_chest", NameKey: "advancement.ender_chest.name", Parent: "ender_eye", HintKey: "advancement.ender_chest.hint", Item: "ender_chest"},
	{ID: "diamond_armor", NameKey: "advancement.diamond_armor.name", Parent: "diamond_tools", HintKey: "advancement.diamond_armor.hint", Item: "diamond_chestplate"},
	{ID: "dye", NameKey: "advancement.dye.name", Parent: "first_bloom", HintKey: "advancement.dye.hint", Item: "red_dye"},
	{ID: "stained_glass", NameKey: "advancement.stained_glass.name", Parent: "dye", HintKey: "advancement.stained_glass.hint", Item: "blue_stained_glass"},
	{ID: "wave_3", NameKey: "advancement.wave_3.name", Parent: "wave_1", HintKey: "advancement.wave_3.hint", Event: true, Icon: "iron_sword", Guide: "dimension_gate"},
	{ID: "wave_8", NameKey: "advancement.wave_8.name", Parent: "wave_5", HintKey: "advancement.wave_8.hint", Event: true, Icon: "netherite_sword", Guide: "dimension_gate"},
	{ID: "tool_made", NameKey: "advancement.tool_made.name", Parent: "tinker_table", HintKey: "advancement.tool_made.hint", Event: true, Icon: "iron_pickaxe", Guide: "tinker_table"},
	{ID: "tool_level_5", NameKey: "advancement.tool_level_5.name", Parent: "tool_level", HintKey: "advancement.tool_level_5.hint", Event: true, Icon: "diamond_pickaxe", Guide: "tinker_table"},
	{ID: "ore_harvest", NameKey: "advancement.ore_harvest.name", Parent: "ore_bloom", HintKey: "advancement.ore_harvest.hint", Event: true, Icon: "iron_ingot", Guide: "magic_botany"},
	{ID: "gem_harvest", NameKey: "advancement.gem_harvest.name", Parent: "gem_bloom", HintKey: "advancement.gem_harvest.hint", Event: true, Icon: "diamond", Guide: "magic_gems"},
	{ID: "ess_harvest", NameKey: "advancement.ess_harvest.name", Parent: "essence_bloom", HintKey: "advancement.ess_harvest.hint", Event: true, Icon: "ender_pearl", Guide: "magic_essence"},
	{ID: "chunks_8", NameKey: "advancement.chunks_8.name", Parent: "chunk", HintKey: "advancement.chunks_8.hint", Event: true, Icon: "grass_block"},
	{ID: "chunks_16", NameKey: "advancement.chunks_16.name", Parent: "chunks_8", HintKey: "advancement.chunks_16.hint", Event: true, Icon: "grass_block"},
	{ID: "project_industry", NameKey: "advancement.project_industry.name", Parent: "conduit_upgrade_3", HintKey: "advancement.project_industry.hint", Research: "project_industry", Icon: "dragonfactory:storage_controller"},
	{ID: "project_hunter", NameKey: "advancement.project_hunter.name", Parent: "project_industry", HintKey: "advancement.project_hunter.hint", Research: "project_hunter", Icon: "diamond_sword"},
	{ID: "project_botanist", NameKey: "advancement.project_botanist.name", Parent: "project_hunter", HintKey: "advancement.project_botanist.hint", Research: "project_botanist", Icon: "dragonfactory:petal_red"},
	{ID: "project_dragon", NameKey: "advancement.project_dragon.name", Parent: "project_botanist", HintKey: "advancement.project_dragon.hint", Research: "project_dragon", Icon: "ender_eye"},
	{ID: advDragon, NameKey: "advancement.dragon_egg.name", Parent: "project_dragon", HintKey: "advancement.dragon_egg.hint", Event: true, Icon: "dragon_egg", Guide: "project_dragon"},
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

// completeAdvancements marks advancements done, with the ones before them
// (an item used up within a second would otherwise leave a gap), and tells
// everyone on the island what they did, as each one's settings say.
func (s *session) completeAdvancements(tx *world.Tx, as []*Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var newly []string
	for _, a := range as {
		for cur, ok := a, true; ok && !s.advanced(cur.ID); cur, ok = advancementByID(cur.Parent) {
			s.state.Advancements[cur.ID] = true
			newly = append(newly, cur.ID)
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
