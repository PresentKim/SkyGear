package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestResearchTreeIsValid: IDs are unique, needs come earlier in the tree,
// every result is a known item and every name is Korean.
func TestResearchTreeIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range researchTree {
		if seen[r.ID] || len(r.Cost) == 0 || len(r.Unlocks) == 0 {
			t.Errorf("research %q is invalid", r.ID)
		}
		for _, n := range r.Needs {
			if !seen[n] {
				t.Errorf("research %q needs %q, which is not before it", r.ID, n)
			}
		}
		seen[r.ID] = true
		for _, n := range r.Unlocks {
			it, ok := world.ItemByName(unlockName(n), 0)
			if !ok {
				t.Errorf("research %q opens unknown item %q", r.ID, n)
				continue
			}
			if got := recipeKey(it); got != unlockName(n) {
				t.Errorf("research %q: item %q encodes as %q", r.ID, n, got)
			}
			if name := itemName(item.NewStack(it, 1)); name == "" || name[0] < 0x80 {
				t.Errorf("research %q: no Korean name for %q (%q)", r.ID, n, name)
			}
		}
		for _, c := range r.Cost {
			if name := itemName(item.NewStack(c.Item, 1)); name == "" || name[0] < 0x80 {
				t.Errorf("research %q: no Korean name for cost %v (%q)", r.ID, c.Item, name)
			}
		}
	}
}

// recipeFor returns a crafting recipe whose result is it. (Vanilla
// recipes are registered with the server config, not in tests.)
func recipeFor(it world.Item) recipe.Recipe {
	return recipe.NewShapeless([]recipe.Item{item.NewStack(block.Planks{Wood: block.OakWood()}, 1)}, item.NewStack(it, 1), "crafting_table")
}

// TestResearchOpensRecipes: only the base recipes are open at first; a
// research paid in parts opens its results when the last part is paid,
// and needs the researches before it.
func TestResearchOpensRecipes(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		open := s.recipeFilter()
		for _, it := range []world.Item{oakPlanks, block.CraftingTable{}, ResearchTable{}} {
			if !open(recipeFor(it)) {
				t.Errorf("base recipe of %v is locked", it)
			}
		}
		if open(recipeFor(stick)) || open(recipeFor(block.NewChest())) {
			t.Fatal("a recipe is open before research")
		}
		chest, _ := researchByID("chest")
		if s.payResearch(p, tx, chest) != 0 {
			t.Fatal("paid for a research whose needs are not done")
		}
		sticks, _ := researchByID("stick")
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 4))
		if n := s.payResearch(p, tx, sticks); n != 4 || !s.researched("stick") {
			t.Fatalf("stick research: took %d, done %v", n, s.researched("stick"))
		}
		// The chest research costs more and is paid in parts.
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 10))
		if n := s.payResearch(p, tx, chest); n != 10 || s.researched("chest") {
			t.Fatalf("first part: took %d, done %v", n, s.researched("chest"))
		}
		_ = p.Inventory().SetItem(0, item.NewStack(oakPlanks, 64))
		if n := s.payResearch(p, tx, chest); n != 54 || !s.researched("chest") {
			t.Fatalf("second part: took %d, done %v", n, s.researched("chest"))
		}
		if st, _ := p.Inventory().Item(0); st.Count() != 10 {
			t.Fatalf("%d planks left, want 10", st.Count())
		}
		if !s.recipeFilter()(recipeFor(stick)) || !s.recipeFilter()(recipeFor(block.NewChest())) {
			t.Fatal("researches did not open sticks and the chest")
		}
	})
}

// TestResearchTableRecipe: the research table is crafted from two crafting
// tables.
func TestResearchTableRecipe(t *testing.T) {
	for _, r := range recipe.Recipes() {
		for _, out := range r.Output() {
			if _, ok := out.Item().(ResearchTable); ok && len(r.Input()) == 2 {
				return
			}
		}
	}
	t.Fatal("no research table recipe")
}
