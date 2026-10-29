package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

func init() {
	world.RegisterItem(sgitem.WoodenBucket{})
	world.RegisterItem(sgitem.WoodenBucket{Water: true})
}
