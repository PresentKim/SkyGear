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

// The crops, barrels, composters and plants grow every second.
func init() {
	core.Second.On(10, "farm.growCrops", func(e core.Tick) {
		farm.GrowCrops(e.S, e.Tx)
	})
	core.Second.On(30, "farm.barrels", func(e core.Tick) {
		farm.TickBarrels(e.S, e.Tx)
		farm.TickComposters(e.S, e.Tx)
	})
	core.Second.On(90, "farm.plants", func(e core.Tick) {
		farm.TickPlants(e.S, e.Tx)
	})
}
