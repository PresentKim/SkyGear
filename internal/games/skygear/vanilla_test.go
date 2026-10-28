package skygear

import (
	"sync"
	_ "unsafe" // go:linkname

	_ "github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/internal/games/skygear/machinerecipe"
)

// registerVanillaRecipes registers the game's own recipes as the server does
// on start (server.Config.New), for tests that read them. The crafter list,
// worked out once on a server after that, is worked out again.
//
//go:linkname recipeRegisterVanilla github.com/df-mc/dragonfly/server/item/recipe.registerVanilla
func recipeRegisterVanilla()

var vanillaOnce sync.Once

func registerVanillaRecipes() {
	vanillaOnce.Do(func() {
		recipeRegisterVanilla()
		machinerecipe.ResetCrafters()
	})
}
