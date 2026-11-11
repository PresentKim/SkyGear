package skygear

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// held is what each player holds, taken by progress.checkHeld before
// dimension.second retires items and handed to progress.collection.
var held = core.NewPart[[]advancement.PlayerItems](nil)

// The advancements and the collection read the items players hold.
func init() {
	core.Second.On(20, "progress.checkHeld", func(e core.Tick) {
		names := advancement.InventoryNames(e.Tx)
		*held.Of(e.S) = names
		progress.CheckHeld(e.S, e.Tx, names)
	})
	core.Second.On(120, "progress.collection", func(e core.Tick) {
		achievement.GrantCollection(*held.Of(e.S))
	})
}
