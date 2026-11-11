package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// keptFood is the food level each player died with, given back when it
// respawns: dying does not fill the food bar (it would make vein mining's
// food cost free).
var keptFood = core.NewPlayers[int]()

func init() {
	// The iron hammer keeps whole: its vein mining costs food instead (vein.go).
	core.ItemDamage.On(10, "tinker.ironHammer", func(e core.ItemDamageEvent) {
		if sgitem.IsIronHammer(e.Item) {
			*e.Damage = 0
		}
	})
	core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
		keptFood.Of(e.S)[e.P.H()] = e.P.Food()
	})
	core.Respawn.On(20, "tinker.restoreFood", func(e core.RespawnEvent) {
		if food, ok := keptFood.Of(e.S)[e.P.H()]; ok {
			delete(keptFood.Of(e.S), e.P.H())
			e.P.SetFood(food)
		}
	})
}
