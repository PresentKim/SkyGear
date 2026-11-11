package dimension

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Feature is the dimension feature, listed in the game's feature list.
var Feature = &core.Feature{
	Name:     "dimension",
	Setup:    subscribe,
	Entities: append([]world.EntityType{GatePortalType, FxDemoType, GemType, PopupType}, MonsterTypes()...),
}
