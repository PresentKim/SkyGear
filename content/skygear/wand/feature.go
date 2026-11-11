package wand

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the wand feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "wand", Setup: subscribe}
