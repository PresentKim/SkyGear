package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/hopper"
)

func logRecipeFor(out world.Item, count int) (recipe.Shaped, bool) {
	for _, r := range recipe.Recipes() {
		sh, ok := r.(recipe.Shaped)
		if !ok || len(r.Output()) != 1 || recipeKey(r.Output()[0].Item()) != recipeKey(out) || r.Output()[0].Count() != count {
			continue
		}
		for _, in := range r.Input() {
			if tag, ok := in.(recipe.ItemTag); ok && tag.Tag() == "minecraft:logs" {
				return sh, true
			}
		}
	}
	return recipe.Shaped{}, false
}

func TestLogRecipesOpenWithTheirResults(t *testing.T) {
	cases := []struct {
		out      world.Item
		count    int
		research string
	}{
		{stick, 16, "stick"},
		{block.NewChest(), 4, "chest"},
		{hopper.NewFlowHopper("wood_hopper", cube.FaceDown), 1, "wood_hopper"},
		{block.NewHopper(), 1, "hopper"},
	}
	for _, c := range cases {
		r, ok := logRecipeFor(c.out, c.count)
		if !ok {
			t.Errorf("no log recipe for %d %s", c.count, recipeKey(c.out))
			continue
		}
		s := testSession(t)
		s.state.Researched = map[string]bool{}
		if s.recipeFilter()(r) {
			t.Errorf("%s open before %s", recipeKey(c.out), c.research)
		}
		s.state.Researched[c.research] = true
		if !s.recipeFilter()(r) {
			t.Errorf("%s not opened by %s", recipeKey(c.out), c.research)
		}
	}
	if !recipe.NewItemTag("minecraft:logs", 1).Contains("minecraft:stripped_oak_log") {
		t.Error("stripped logs are not logs")
	}
}
