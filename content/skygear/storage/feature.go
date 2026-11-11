package storage

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Feature is the storage feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "storage", Entities: []world.EntityType{HolderType}}
