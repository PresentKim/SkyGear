package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
)

// A player that falls below the island returns from the void.
func init() {
	core.Move.On(40, "island.void", func(e core.MoveEvent) {
		if e.Pos.Y() < island.VoidY {
			e.Cancel()
			island.ReturnFromVoid(e.S, e.P, e.Tx)
		}
	})
}
