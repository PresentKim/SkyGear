package machine

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Feature is the machine feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "machine", Entities: []world.EntityType{RangeBoxType}, Creative: creativeItems, Setup: subscribe}
