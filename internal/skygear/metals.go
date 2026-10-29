package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

func init() {
	for _, m := range sgitem.DustMetals {
		world.RegisterItem(sgitem.MetalDust{Metal: m})
	}
	for _, m := range sgitem.IngotMetals {
		world.RegisterItem(sgitem.Ingot{Metal: m})
	}
}
