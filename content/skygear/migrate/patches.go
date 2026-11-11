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
	}},
}
