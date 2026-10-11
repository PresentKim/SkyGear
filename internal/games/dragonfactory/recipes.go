package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Recipe locking (spike for the research system, 13th playtest request):
// while an island's recipes are locked, its players only get the recipes
// whose result is unlocked (Player.SetRecipeFilter, Dragonfly patch); the
// list is sent again at once when one is unlocked. Kept in memory only:
// /dfadmin recipes <lock|unlock|all> tries it in game before the research
// design is settled (docs/design/open-questions.md).

// baseRecipes are the results always open while recipes are locked.
var baseRecipes = []world.Item{block.Planks{Wood: block.OakWood()}, block.CraftingTable{}}

// recipeKey is the key of an item for unlocking: its name, so all variants
// (e.g. every colour) open together.
func recipeKey(it world.Item) string {
	name, _ := it.EncodeItem()
	return name
}

// recipeOpen reports whether a recipe is open on a locked island: one of
// its results is unlocked.
func (s *session) recipeOpen(r recipe.Recipe) bool {
	return slices.ContainsFunc(r.Output(), func(st item.Stack) bool {
		return !st.Empty() && s.unlocked[recipeKey(st.Item())]
	})
}

// lockRecipes locks the island's recipes but the base ones.
func (s *session) lockRecipes(tx *world.Tx) {
	s.unlocked = map[string]bool{}
	for _, it := range baseRecipes {
		s.unlocked[recipeKey(it)] = true
	}
	s.sendRecipes(tx)
}

// unlockRecipe opens the recipes making it and sends the new lists.
func (s *session) unlockRecipe(tx *world.Tx, it world.Item) {
	if s.unlocked == nil {
		return
	}
	s.unlocked[recipeKey(it)] = true
	s.sendRecipes(tx)
}

// openRecipes opens every recipe again.
func (s *session) openRecipes(tx *world.Tx) {
	s.unlocked = nil
	s.sendRecipes(tx)
}

// sendRecipes sends every player on the island its recipe list.
func (s *session) sendRecipes(tx *world.Tx) {
	for e := range tx.Players() {
		s.sendRecipesTo(e.(*player.Player))
	}
}

// sendRecipesTo sends p the island's recipe list.
func (s *session) sendRecipesTo(p *player.Player) {
	if s.unlocked == nil {
		p.SetRecipeFilter(nil)
		return
	}
	p.SetRecipeFilter(s.recipeOpen)
}
