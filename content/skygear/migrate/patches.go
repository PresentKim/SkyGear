package migrate

// Patches are the changes to saved data, oldest first.
var Patches = []Patch{
	{Version: "26.11.3", Name: "drawer controller and trim", Rename: map[string]string{
		"skygear:storage_controller":     "skygear:drawer_controller",
		"skygear:storage_link":           "skygear:drawer_trim",
		"place_storage_controller":       "place_drawer_controller",
		"place_storage_link":             "place_drawer_trim",
		"sg_place_storage_controller_1":  "sg_place_drawer_controller_1",
		"sg_place_storage_controller_10": "sg_place_drawer_controller_10",
		"sg_place_storage_link_1":        "sg_place_drawer_trim_1",
		"sg_place_storage_link_10":       "sg_place_drawer_trim_10",
	}},
	{Version: "26.11.9", Name: "retired blocks and items", Rename: map[string]string{
		// Each name below has its own reason.
		// skygear:sell_box: the block is a chest now. The sell box had 27
		// slots like a chest (its stacks were in 9..17), so the old block
		// entity's Items by Slot are read by the chest as they are.
		"skygear:sell_box": "minecraft:chest",
		// ess_<mob>: the essence blooms were redstone blooms in all but
		// name (the game turned them into one on the next tick). The kind
		// is saved as a string: in the state of the block skygear:magic_plant
		// (ess_<mob>_0 is a bud, ess_<mob>_1 a bloom, as for any kind) and
		// in the plant records of the save. The soil (dirt, grass, farmland
		// or podzol) is the same for every kind, so nothing pops off; the
		// plant record keeps its ready time. No other saved string is one
		// of these.
		"ess_zombie":     "gem_redstone",
		"ess_zombie_0":   "gem_redstone_0",
		"ess_zombie_1":   "gem_redstone_1",
		"ess_skeleton":   "gem_redstone",
		"ess_skeleton_0": "gem_redstone_0",
		"ess_skeleton_1": "gem_redstone_1",
		"ess_creeper":    "gem_redstone",
		"ess_creeper_0":  "gem_redstone_0",
		"ess_creeper_1":  "gem_redstone_1",
		"ess_spider":     "gem_redstone",
		"ess_spider_0":   "gem_redstone_0",
		"ess_spider_1":   "gem_redstone_1",
		"ess_slime":      "gem_redstone",
		"ess_slime_0":    "gem_redstone_0",
		"ess_slime_1":    "gem_redstone_1",
		"ess_enderman":   "gem_redstone",
		"ess_enderman_0": "gem_redstone_0",
		"ess_enderman_1": "gem_redstone_1",
		"ess_blaze":      "gem_redstone",
		"ess_blaze_0":    "gem_redstone_0",
		"ess_blaze_1":    "gem_redstone_1",
	},
		// Plain: the stacks of the retired items become plain stacks of what they
		// are renamed to, with no old name, lore, durability or values.
		Plain: map[string]string{
			// skygear:dimension_gate: the retired gate (a block and an item)
			// was made of steel, obsidian and ender pearls; of those the
			// obsidian is the one that is a block, so the placed block (see
			// Blocks) and the item are obsidian. A placed gate had a block
			// entity the obsidian cannot read; Blocks drops it. A gate item
			// that was carried becomes plain obsidian.
			"skygear:dimension_gate": "minecraft:obsidian",
			// skygear:tool_sword_<head>_<handle>: the tinker sword is gone
			// (the second dimension's relics fight). Each becomes one ingot
			// of its head material, the head being what the sword was worth.
			// Its XP, modifiers, durability, custom name and lore go with it:
			// the result is a plain ingot that stacks with any other.
			"skygear:tool_sword_flint_wood":     "minecraft:flint",
			"skygear:tool_sword_flint_bone":     "minecraft:flint",
			"skygear:tool_sword_flint_blaze":    "minecraft:flint",
			"skygear:tool_sword_iron_wood":      "minecraft:iron_ingot",
			"skygear:tool_sword_iron_bone":      "minecraft:iron_ingot",
			"skygear:tool_sword_iron_blaze":     "minecraft:iron_ingot",
			"skygear:tool_sword_diamond_wood":   "minecraft:diamond",
			"skygear:tool_sword_diamond_bone":   "minecraft:diamond",
			"skygear:tool_sword_diamond_blaze":  "minecraft:diamond",
			"skygear:tool_sword_bronze_wood":    "skygear:ingot_bronze",
			"skygear:tool_sword_bronze_bone":    "skygear:ingot_bronze",
			"skygear:tool_sword_bronze_blaze":   "skygear:ingot_bronze",
			"skygear:tool_sword_electrum_wood":  "skygear:ingot_electrum",
			"skygear:tool_sword_electrum_bone":  "skygear:ingot_electrum",
			"skygear:tool_sword_electrum_blaze": "skygear:ingot_electrum",
			"skygear:tool_sword_invar_wood":     "skygear:ingot_invar",
			"skygear:tool_sword_invar_bone":     "skygear:ingot_invar",
			"skygear:tool_sword_invar_blaze":    "skygear:ingot_invar",
			"skygear:tool_sword_steel_wood":     "skygear:ingot_steel",
			"skygear:tool_sword_steel_bone":     "skygear:ingot_steel",
			"skygear:tool_sword_steel_blaze":    "skygear:ingot_steel",
			// skygear:research_note: it has only said it is no longer used
			// since research moved to the research table; paper is what a
			// note is.
			"skygear:research_note": "minecraft:paper",
			// skygear:seed_ess_<mob>: the essence bloom seeds are redstone
			// bloom seeds now (the retired kind's own replacement, the same
			// one the game has made of them while it still knew them). A
			// seed carries nothing, so it becomes a plain seed; the disks
			// of the storage network add their keys up.
			"skygear:seed_ess_zombie":   "skygear:seed_gem_redstone",
			"skygear:seed_ess_skeleton": "skygear:seed_gem_redstone",
			"skygear:seed_ess_creeper":  "skygear:seed_gem_redstone",
			"skygear:seed_ess_spider":   "skygear:seed_gem_redstone",
			"skygear:seed_ess_slime":    "skygear:seed_gem_redstone",
			"skygear:seed_ess_enderman": "skygear:seed_gem_redstone",
			"skygear:seed_ess_blaze":    "skygear:seed_gem_redstone",
			// skygear:infested_leaves: the leaves of 26.11.7 are the
			// silkworm leaves; the item is the leaves that can be placed.
			// The placed block is replaced by Blocks.
			"skygear:infested_leaves": "skygear:infesting_leaves",
		},
		// A placed gate is obsidian. A gate had a block entity; the world
		// would log an error for it at every load, so it is dropped.
		Blocks: map[string]Block{
			"skygear:dimension_gate": {Name: "minecraft:obsidian", DropEntities: true},
			// The leaves of 26.11.7 were full silkworm leaves that turned
			// into a full cocoon (10) of leaves that are not persistent on
			// their next tick or update; they are that from the start.
			"skygear:infested_leaves": {Name: "skygear:infesting_leaves", States: map[string]any{"skygear:infest_stage": "10"}},
		}},
}
