package netstore

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the netstore feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "netstore", Creative: creativeItems, Setup: subscribe}
