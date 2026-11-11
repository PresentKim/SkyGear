package progress

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the progress feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "progress", Setup: subscribe}
