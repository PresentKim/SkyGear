package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// recipeFor returns a crafting recipe whose result is it. (Vanilla
// recipes are registered with the server config, not in tests.)
func recipeFor(it world.Item) recipe.Recipe {
	return recipe.NewShapeless([]recipe.Item{item.NewStack(block.Planks{Wood: block.OakWood()}, 1)}, item.NewStack(it, 1), "crafting_table")
}

func TestRecipeLock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		planks, table, stick := recipeFor(block.Planks{Wood: block.OakWood()}), recipeFor(block.CraftingTable{}), recipeFor(item.Stick{})
		s.lockRecipes(tx)
		if !s.recipeOpen(planks) || !s.recipeOpen(table) || s.recipeOpen(stick) {
			t.Fatal("locked: planks and the crafting table should be the only ones open")
		}
		s.unlockRecipe(tx, item.Stick{})
		if !s.recipeOpen(stick) {
			t.Fatal("stick not open after unlocking")
		}
		s.openRecipes(tx)
		if s.unlocked != nil {
			t.Fatal("still locked")
		}
	})
}
