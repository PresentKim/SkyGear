package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The conduit binder (after EnderIO's): sand and gravel in a checkerboard
// make the binder mix, which a furnace bakes into the binder, the base every
// conduit is made of.

func init() {
	world.RegisterItem(sgitem.ConduitBinderMix{})
	world.RegisterItem(sgitem.ConduitBinder{})
}
