package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

func init() {
	world.RegisterItem(sgitem.Hammer{})
	world.RegisterItem(sgitem.Hammer{Stone: true})
}
