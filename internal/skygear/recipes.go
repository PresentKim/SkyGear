package skygear

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// Recipe locking (research.go): players on the island
// only get the recipes whose result is open: the base results and those of
// the done researches (Player.SetRecipeFilter, Dragonfly patch). The list
// is sent again at once when a research is done.

// recipeKey is the key of an item for opening recipes: its name, so all
// variants (e.g. every colour) open together.
func recipeKey(it world.Item) string {
	name, _ := it.EncodeItem()
	return name
}

// openResults are the result names whose recipes are open on the island.
func (s *session) openResults() map[string]bool {
	open := map[string]bool{}
	for _, n := range baseUnlocks {
		open[naming.UnlockName(n)] = true
	}
	for i := range researchTree {
		if r := &researchTree[i]; s.researched(r.ID) {
			for _, n := range r.Unlocks {
				open[naming.UnlockName(n)] = true
			}
		}
	}
	return open
}

// recipeFilter returns the filter of the island's recipes: open while one
// of a recipe's results is open.
func (s *session) recipeFilter() func(recipe.Recipe) bool {
	open := s.openResults()
	return func(r recipe.Recipe) bool {
		return slices.ContainsFunc(r.Output(), func(st item.Stack) bool {
			return !st.Empty() && open[recipeKey(st.Item())]
		})
	}
}

// sendRecipes sends every player on the island its recipe list.
func (s *session) sendRecipes(tx *world.Tx) {
	filter := s.islandRecipeFilter()
	for e := range tx.Players() {
		e.(*player.Player).SetRecipeFilter(filter)
	}
}

// sendRecipesTo sends p the island's recipe list.
func (s *session) sendRecipesTo(p *player.Player) {
	p.SetRecipeFilter(s.islandRecipeFilter())
}

// islandRecipeFilter is the filter players on the island get: none while an
// operator opened every recipe with /sgadmin research free.
func (s *session) islandRecipeFilter() func(recipe.Recipe) bool {
	if s.recipesFree {
		return nil
	}
	return s.recipeFilter()
}
