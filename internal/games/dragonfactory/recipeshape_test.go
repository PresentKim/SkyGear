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
	// An empty cell is an empty stack, never a nil input (the recipe list
	// sent to the client crashed the server on a nil).
	for _, r := range recipe.Recipes() {
		for i, in := range r.Input() {
			if in == nil {
				t.Fatalf("recipe of %v has a nil input at %d", r.Output(), i)
			}
		}
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

// TestConduitRecipes: the binder mix is a sand and gravel checkerboard, the
// basic conduits are binders above and below a row of redstone or iron
// nugget, chest, iron nugget; the mix bakes into the binder; the better tiers
// have no recipe yet.
func TestConduitRecipes(t *testing.T) {
	cells := func(it world.Item) (string, int, []string) {
		for _, r := range recipe.Recipes() {
			for _, out := range r.Output() {
				if recipeKey(out.Item()) != recipeKey(it) {
					continue
				}
				var names []string
				for _, in := range r.Input() {
					st, ok := in.(item.Stack)
					if !ok || st.Empty() {
						names = append(names, ".")
						continue
					}
					names = append(names, recipeKey(st.Item()))
				}
				return "found", out.Count(), names
			}
		}
		return "", 0, nil
	}
	join := func(n []string) string {
		s := ""
		for _, c := range n {
			s += c + " "
		}
		return s
	}
	_, count, in := cells(ConduitBinderMix{})
	want := "minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand "
	if count != 6 || join(in) != want {
		t.Errorf("mix: %d from %s", count, join(in))
	}
	b := "dragonfactory:conduit_binder"
	_, count, in = cells(EnergyConduit{Tier: 1})
	want = b + " " + b + " " + b + " minecraft:redstone minecraft:redstone minecraft:redstone " + b + " " + b + " " + b + " "
	if count != 8 || join(in) != want {
		t.Errorf("energy conduit: %d from %s", count, join(in))
	}
	_, count, in = cells(ItemConduit{Tier: 1})
	want = b + " " + b + " " + b + " minecraft:iron_nugget minecraft:chest minecraft:iron_nugget " + b + " " + b + " " + b + " "
	if count != 8 || join(in) != want {
		t.Errorf("item conduit: %d from %s", count, join(in))
	}
	for t2 := 2; t2 <= 3; t2++ {
		if found, _, _ := cells(EnergyConduit{Tier: t2}); found != "" {
			t.Errorf("energy conduit tier %d has a recipe", t2)
		}
		if found, _, _ := cells(ItemConduit{Tier: t2}); found != "" {
			t.Errorf("item conduit tier %d has a recipe", t2)
		}
	}
	if info := (ConduitBinderMix{}).SmeltInfo(); info.Product.Item() != (ConduitBinder{}) {
		t.Errorf("the mix bakes into %v", info.Product.Item())
	}
}
