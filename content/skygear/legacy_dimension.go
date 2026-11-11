package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
)

// The second dimension's reaction to a death and a respawn.
func init() {
	core.Death.On(20, "dimension.died", func(e core.PlayerEvent) {
		dimension.Died(e.S, e.P.Tx(), e.P)
	})
	core.Respawn.On(10, "dimension.respawned", func(e core.RespawnEvent) {
		dimension.Respawned(e.S, e.P, e.Pos)
	})
}
