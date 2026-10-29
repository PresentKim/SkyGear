package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

func init() {
	for _, metal := range sgitem.NewOreMetals {
		world.RegisterBlock(sgitem.OreBlock{Metal: metal})
		world.RegisterItem(sgitem.OreBlock{Metal: metal})
		world.RegisterItem(sgitem.RawOre{Metal: metal})
	}
}
