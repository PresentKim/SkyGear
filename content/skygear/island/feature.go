package island

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the island feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "island", Setup: subscribe}
