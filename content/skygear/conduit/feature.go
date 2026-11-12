package conduit

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Feature is the conduit feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "conduit", Entities: []world.EntityType{VisualType, PlateType}, Creative: creativeItems, Setup: subscribe}
