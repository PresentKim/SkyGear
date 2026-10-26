package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	for _, k := range dfitem.CrushedKinds {
		world.RegisterBlock(dfitem.CrushedBlock{Kind: k})
		world.RegisterItem(dfitem.CrushedBlock{Kind: k})
	}
}
