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
}
