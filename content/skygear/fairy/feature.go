package fairy

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Feature is the fairy feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "fairy", Entities: []world.EntityType{EntityType}}
