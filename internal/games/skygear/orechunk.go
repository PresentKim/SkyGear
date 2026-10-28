package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
)

func init() {
	for _, k := range sgitem.OreKinds {
		world.RegisterItem(sgitem.OreChunk{Ore: k.Ore})
	}
	sgitem.OreChunkRecipes()
}
