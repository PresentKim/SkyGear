package farm

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the farm feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "farm", Creative: creativeItems, Commands: commands, Setup: subscribe}
