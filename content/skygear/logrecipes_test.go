package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/progress"
)

func logRecipeFor(out world.Item, count int) (recipe.Shaped, bool) {
	for _, r := range recipe.Recipes() {
		sh, ok := r.(recipe.Shaped)
		if !ok || len(r.Output()) != 1 || progress.RecipeKey(r.Output()[0].Item()) != progress.RecipeKey(out) || r.Output()[0].Count() != count {
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
			t.Errorf("no log recipe for %d %s", c.count, progress.RecipeKey(c.out))
			continue
		}
		s := testSession(t)
		progress.StateOf(s.Session).Researched = map[string]bool{}
		if progress.RecipeFilter(s.Session)(r) {
			t.Errorf("%s open before %s", progress.RecipeKey(c.out), c.research)
		}
		progress.StateOf(s.Session).Researched[c.research] = true
		if !progress.RecipeFilter(s.Session)(r) {
			t.Errorf("%s not opened by %s", progress.RecipeKey(c.out), c.research)
		}
	}
	if !recipe.NewItemTag("minecraft:logs", 1).Contains("minecraft:stripped_oak_log") {
		t.Error("stripped logs are not logs")
	}
}
