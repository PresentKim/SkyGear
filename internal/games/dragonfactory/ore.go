package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	for _, metal := range dfitem.NewOreMetals {
		world.RegisterBlock(dfitem.OreBlock{Metal: metal})
		world.RegisterItem(dfitem.OreBlock{Metal: metal})
		world.RegisterItem(dfitem.RawOre{Metal: metal})
	}
}
