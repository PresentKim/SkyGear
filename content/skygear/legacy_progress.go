package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
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

// A joining player is sent the recipes they know; the recipe filter, which
// belongs to this island, is cleared on leave.
func init() {
	core.Join.On(60, "progress.sendRecipes", func(e core.Presence) {
		progress.SendRecipesTo(e.S, e.P)
	})
	core.Leave.On(70, "progress.recipes", func(e core.Presence) {
		// Locked recipes belong to this island (recipes.go).
		e.P.SetRecipeFilter(nil)
	})
}

// Placing the dragon egg finishes the game.
func init() {
	core.BlockPlace.On(100, "progress.dragonEgg", func(e core.BlockPlaceEvent) {
		if _, ok := e.Block.(block.DragonEgg); ok {
			completeDragon(e.S, e.P, e.Tx, e.Pos)
		}
	})
}
