package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// TestMagicSeedsAndRecipes: the flower seed makes a random colour, every better seed
// has a recipe, and the researches unlock them in order.
func TestMagicSeedsAndRecipes(t *testing.T) {
	if len(farm.PlantKinds) != 16+7+6 {
		t.Fatalf("%d kinds", len(farm.PlantKinds))
	}
	have := map[string]bool{}
	for _, r := range recipe.Recipes() {
		for _, o := range r.Output() {
			if seed, ok := o.Item().(farm.PlantSeed); ok {
				have[seed.Seed] = true
			}
		}
	}
	for _, sd := range farm.PlantSeeds {
		if !strings.HasPrefix(sd, "flower_") && !have[sd] {
			t.Errorf("no recipe makes the seed %s", sd)
		}
	}
	unlocked := map[string]string{}
	for _, r := range progress.Tree {
		for _, u := range r.Unlocks {
			unlocked[u] = r.ID
		}
	}
	for _, sd := range farm.PlantSeeds {
		if !strings.HasPrefix(sd, "flower_") && unlocked["skygear:seed_"+sd] == "" {
			t.Errorf("no research unlocks the seed %s", sd)
		}
	}
	for _, k := range farm.PlantKinds {
		if len(k.Yield) == 0 || k.Grow <= 0 {
			t.Errorf("%s: incomplete", k.ID)
		}
	}
}
