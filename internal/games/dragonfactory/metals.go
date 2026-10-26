package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	for _, m := range dfitem.DustMetals {
		world.RegisterItem(dfitem.MetalDust{Metal: m})
	}
	for _, m := range dfitem.IngotMetals {
		world.RegisterItem(dfitem.Ingot{Metal: m})
	}
}
