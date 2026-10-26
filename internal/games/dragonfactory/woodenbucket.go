package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	world.RegisterItem(dfitem.WoodenBucket{})
	world.RegisterItem(dfitem.WoodenBucket{Water: true})
}
