package progress

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// RecipeKey is the key of an item for opening recipes: its name, so all
// variants (e.g. every colour) open together.
func RecipeKey(it world.Item) string {
	name, _ := it.EncodeItem()
	return name
}

// OpenResults are the result names whose recipes are open on the island.
func OpenResults(s *core.Session) map[string]bool {
	open := map[string]bool{}
	for _, n := range BaseUnlocks {
		open[naming.UnlockName(n)] = true
	}
	for i := range Tree {
		if r := &Tree[i]; Researched(s, r.ID) {
			for _, n := range r.Unlocks {
				open[naming.UnlockName(n)] = true
			}
		}
	}
	return open
}

// RecipeFilter returns the filter of the island's recipes: open while one
// of a recipe's results is open.
func RecipeFilter(s *core.Session) func(recipe.Recipe) bool {
	open := OpenResults(s)
	return func(r recipe.Recipe) bool {
		return slices.ContainsFunc(r.Output(), func(st item.Stack) bool {
			return !st.Empty() && open[RecipeKey(st.Item())]
		})
	}
}

// SendRecipes sends every player on the island its recipe list.
func SendRecipes(s *core.Session, tx *world.Tx) {
	filter := IslandRecipeFilter(s)
	for e := range tx.Players() {
		e.(*player.Player).SetRecipeFilter(filter)
	}
}

// SendRecipesTo sends p the island's recipe list.
func SendRecipesTo(s *core.Session, p *player.Player) {
	p.SetRecipeFilter(IslandRecipeFilter(s))
}

// IslandRecipeFilter is the filter players on the island get: none while an
// operator opened every recipe with /sgadmin research free.
func IslandRecipeFilter(s *core.Session) func(recipe.Recipe) bool {
	if LocalOf(s).RecipesFree {
		return nil
	}
	return RecipeFilter(s)
}

// BaseUnlocks are the results open without research.
var BaseUnlocks = []string{"oak_planks", "crafting_table", "skygear:research_table"}
