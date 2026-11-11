package network

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the network feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "network", Setup: subscribe}
