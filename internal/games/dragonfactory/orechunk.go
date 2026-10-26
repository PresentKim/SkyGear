package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	for _, k := range dfitem.OreKinds {
		world.RegisterItem(dfitem.OreChunk{Ore: k.Ore})
	}
	dfitem.OreChunkRecipes()
}
