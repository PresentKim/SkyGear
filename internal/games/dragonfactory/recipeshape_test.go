package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
)

// TestToolAndMachineRecipesAreShaped: the hammer, the sieve and the drawer
// are crafted in one arrangement only, not mirrored either; the sieve takes
// no string (its mesh is fitted separately).
func TestToolAndMachineRecipesAreShaped(t *testing.T) {
	find := func(it world.Item) recipe.Shaped {
		for _, r := range recipe.Recipes() {
			for _, out := range r.Output() {
				if recipeKey(out.Item()) != recipeKey(it) {
					continue
				}
				sh, ok := r.(recipe.Shaped)
				if !ok {
					t.Fatalf("the recipe of %s is not shaped", recipeKey(it))
				}
				return sh
			}
		}
		t.Fatalf("no recipe of %s", recipeKey(it))
		return recipe.Shaped{}
	}
	name := func(in recipe.Item) string {
		st, ok := in.(item.Stack)
		if !ok || st.Empty() {
			return "."
		}
		return recipeKey(st.Item())
	}
	hammer := find(Hammer{})
	if hammer.Shape() != recipe.NewShape(3, 3) || !hammer.NoMirror() {
		t.Errorf("hammer shape %v, no mirror %v", hammer.Shape(), hammer.NoMirror())
	}
	want := []string{".", "minecraft:cobblestone", ".", ".", "minecraft:stick", "minecraft:cobblestone", "minecraft:stick", ".", "."}
	for i, in := range hammer.Input() {
		if name(in) != want[i] {
			t.Errorf("hammer cell %d is %s, want %s", i, name(in), want[i])
		}
	}
	sieve := find(Machine{Kind: KindSieve})
	if sieve.Shape() != recipe.NewShape(3, 2) || !sieve.NoMirror() {
		t.Errorf("sieve shape %v", sieve.Shape())
	}
	for _, in := range sieve.Input() {
		if name(in) == "minecraft:string" {
			t.Error("the sieve takes string")
		}
	}
	if drawer := find(Drawer{}); drawer.Shape() != recipe.NewShape(3, 3) || !drawer.NoMirror() || name(drawer.Input()[4]) != "minecraft:chest" {
		t.Errorf("drawer shape %v centre %s", drawer.Shape(), name(drawer.Input()[4]))
	}
}
