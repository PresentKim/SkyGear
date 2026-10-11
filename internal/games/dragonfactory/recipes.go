package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Recipe locking (research.go, docs/tech/recipes.md): players on the island
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
		open[unlockName(n)] = true
	}
	for i := range researchTree {
		if r := &researchTree[i]; s.researched(r.ID) {
			for _, n := range r.Unlocks {
				open[unlockName(n)] = true
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
	for e := range tx.Players() {
		s.sendRecipesTo(e.(*player.Player))
	}
}

// sendRecipesTo sends p the island's recipe list (every recipe while an
// operator opened them all with /dfadmin research free).
func (s *session) sendRecipesTo(p *player.Player) {
	if s.recipesFree {
		p.SetRecipeFilter(nil)
		return
	}
	p.SetRecipeFilter(s.recipeFilter())
}
