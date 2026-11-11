package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
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

// TestNoEssenceBloomRecipes: no recipe makes essence bloom seeds, flower
// seeds come from essences and fairy dust, and no sieve finds monster
// essences.
func TestNoEssenceBloomRecipes(t *testing.T) {
	flower := false
	for _, r := range recipe.Recipes() {
		for _, o := range r.Output() {
			seed, ok := o.Item().(farm.PlantSeed)
			if !ok {
				continue
			}
			if strings.HasPrefix(seed.Seed, "ess_") {
				t.Errorf("a recipe makes %s", seed.Seed)
			}
			if seed.Seed == "flower_red" {
				for _, in := range r.Input() {
					if st, ok := in.(item.Stack); ok {
						if _, ess := st.Item().(sgitem.Essence); ess {
							flower = true
						}
					}
				}
			}
		}
	}
	if !flower {
		t.Error("no recipe makes a red flower seed from essences")
	}
	for tier, table := range sievetable.Tables {
		for in, drops := range table {
			for _, d := range drops {
				if _, ok := d.Item.Item().(sgitem.Essence); ok {
					t.Errorf("the %s mesh finds an essence in %s", tier, in)
				}
			}
		}
	}
}
