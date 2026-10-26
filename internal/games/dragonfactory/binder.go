package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// The conduit binder (EnderIO's conduit binder, docs/design/06-power-and-conduits.md
// §5.1): sand and gravel in a checkerboard make the binder mix, which a
// furnace bakes into the binder, the base every conduit is made of.

func init() {
	world.RegisterItem(dfitem.ConduitBinderMix{})
	world.RegisterItem(dfitem.ConduitBinder{})
}
