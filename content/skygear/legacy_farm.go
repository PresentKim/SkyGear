package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
)

// Sneaking grows saplings nearby by chance.
func init() {
	core.Sneak.On(40, "farm.crouchGrow", func(e core.PlayerEvent) {
		farm.CrouchGrow(e.S, e.P, e.Tx)
	})
}
