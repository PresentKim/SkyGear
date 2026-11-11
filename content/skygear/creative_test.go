package skygear

import (
	"go/parser"
	"go/token"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/content/skygear/catalog"
	"github.com/presentkim/skygear/content/skygear/core"
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

// TestCreativeListNamesNoFeature: the content list's creative file lists the
// items of the shared item packages only; a feature lists its own items in
// its descriptor (core.Feature.Creative).
func TestCreativeListNamesNoFeature(t *testing.T) {
	const prefix = "github.com/presentkim/skygear/content/skygear/"
	f, err := parser.ParseFile(token.NewFileSet(), "catalog/creative.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	allowed := []string{"core", "hopper", "machine", "sgitem"}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if dep, ok := strings.CutPrefix(path, prefix); ok && !slices.Contains(allowed, dep) {
			t.Errorf("catalog/creative.go imports %s: the feature lists its creative items itself", dep)
		}
	}
	slots := map[int]string{}
	for _, f := range core.Features() {
		if f.Creative == nil {
			continue
		}
		for _, it := range f.Creative() {
			if o, taken := slots[it.Slot]; taken && o != f.Name {
				t.Errorf("creative slot %d is used by %s and %s", it.Slot, o, f.Name)
			}
			slots[it.Slot] = f.Name
		}
	}
	want := map[int]string{10: "progress", 20: "storage", 30: "farm", 40: "conduit", 50: "machine", 70: "elevator", 90: "farm", 110: "netstore", 130: "fairy", 150: "farm"}
	if !maps.Equal(slots, want) {
		t.Errorf("creative slots of the features %v, want %v", slots, want)
	}
}
