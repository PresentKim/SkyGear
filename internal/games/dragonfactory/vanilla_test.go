package dragonfactory

import (
	"sync"
	_ "unsafe" // go:linkname

	_ "github.com/df-mc/dragonfly/server/item/recipe"
)

// registerVanillaRecipes registers the game's own recipes as the server does
// on start (server.Config.New), for tests that read them.
//
//go:linkname recipeRegisterVanilla github.com/df-mc/dragonfly/server/item/recipe.registerVanilla
func recipeRegisterVanilla()

var vanillaOnce sync.Once

func registerVanillaRecipes() { vanillaOnce.Do(recipeRegisterVanilla) }
