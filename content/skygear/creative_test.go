package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/content/skygear/catalog"
)

// TestEveryGameRecipeResultIsCreative: the client lists a recipe in its
// recipe book only if the result is in the creative inventory, so every
// result of the game's own items must be registered there (the hammer and
// the sieve were missing from the crafting list).
func TestEveryGameRecipeResultIsCreative(t *testing.T) {
	catalog.RegisterCreativeItems()
	have := map[string]bool{}
	for _, c := range creative.Items() {
		name, _ := c.Stack.Item().EncodeItem()
		have[name] = true
	}
	for _, r := range recipe.Recipes() {
		for _, out := range r.Output() {
			name, _ := out.Item().EncodeItem()
			if strings.HasPrefix(name, "skygear:") && !have[name] {
				t.Errorf("recipe result %s is not in the creative inventory", name)
			}
		}
	}
	for _, c := range creative.Items() {
		if name, _ := c.Stack.Item().EncodeItem(); strings.HasPrefix(name, "skygear:") {
			found := false
			for _, g := range creative.Groups() {
				found = found || g.Name == c.Group
			}
			if !found {
				t.Errorf("%s: group %q is not registered", name, c.Group)
			}
		}
	}
}
