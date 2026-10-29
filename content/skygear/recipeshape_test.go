package skygear

import (
	"github.com/presentkim/skygear/content/skygear/storage"
	"testing"

	"github.com/presentkim/skygear/content/skygear/conduit"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestToolAndMachineRecipesAreShaped: the hammer, the sieve and the drawer
// are crafted in one arrangement only, not mirrored either; the sieve takes
// no string (its mesh is fitted separately).
func TestToolAndMachineRecipesAreShaped(t *testing.T) {
	find := func(it world.Item) recipe.Shaped {
		for _, r := range recipe.Recipes() {
			for _, out := range r.Output() {
				if progress.RecipeKey(out.Item()) != progress.RecipeKey(it) {
					continue
				}
				sh, ok := r.(recipe.Shaped)
				if !ok {
					t.Fatalf("the recipe of %s is not shaped", progress.RecipeKey(it))
				}
				return sh
			}
		}
		t.Fatalf("no recipe of %s", progress.RecipeKey(it))
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
		return progress.RecipeKey(st.Item())
	}
	hammer := find(sgitem.Hammer{Stone: true})
	if hammer.Shape() != recipe.NewShape(3, 3) || !hammer.NoMirror() {
		t.Errorf("hammer shape %v, no mirror %v", hammer.Shape(), hammer.NoMirror())
	}
	want := []string{".", "minecraft:cobblestone", ".", ".", "minecraft:stick", "minecraft:cobblestone", "minecraft:stick", ".", "."}
	for i, in := range hammer.Input() {
		if name(in) != want[i] {
			t.Errorf("hammer cell %d is %s, want %s", i, name(in), want[i])
		}
	}
	sieve := find(machine.Machine{Kind: machine.KindSieve})
	if sieve.Shape() != recipe.NewShape(3, 2) || !sieve.NoMirror() {
		t.Errorf("sieve shape %v", sieve.Shape())
	}
	for _, in := range sieve.Input() {
		if name(in) == "minecraft:string" {
			t.Error("the sieve takes string")
		}
	}
	if drawer := find(storage.Drawer{}); drawer.Shape() != recipe.NewShape(3, 3) || !drawer.NoMirror() || name(drawer.Input()[4]) != "minecraft:chest" {
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
				if progress.RecipeKey(out.Item()) != progress.RecipeKey(it) {
					continue
				}
				var names []string
				for _, in := range r.Input() {
					st, ok := in.(item.Stack)
					if !ok || st.Empty() {
						names = append(names, ".")
						continue
					}
					names = append(names, progress.RecipeKey(st.Item()))
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
	_, count, in := cells(sgitem.ConduitBinderMix{})
	want := "minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand minecraft:gravel minecraft:sand "
	if count != 6 || join(in) != want {
		t.Errorf("mix: %d from %s", count, join(in))
	}
	b := "skygear:conduit_binder"
	_, count, in = cells(conduit.EnergyConduit{Tier: 1})
	want = b + " " + b + " " + b + " minecraft:redstone minecraft:redstone minecraft:redstone " + b + " " + b + " " + b + " "
	if count != 8 || join(in) != want {
		t.Errorf("energy conduit: %d from %s", count, join(in))
	}
	_, count, in = cells(conduit.ItemConduit{Tier: 1})
	want = b + " " + b + " " + b + " minecraft:iron_nugget minecraft:chest minecraft:iron_nugget " + b + " " + b + " " + b + " "
	if count != 8 || join(in) != want {
		t.Errorf("item conduit: %d from %s", count, join(in))
	}
	_, count, in = cells(conduit.FluidConduit{Tier: 1})
	want = b + " " + b + " " + b + " minecraft:glass minecraft:glass minecraft:glass " + b + " " + b + " " + b + " "
	if count != 8 || join(in) != want {
		t.Errorf("fluid conduit: %d from %s", count, join(in))
	}
	// the better tiers are made from the tier below: electrum for the second,
	// steel and a diamond for the third
	for _, kind := range []string{"energy", "item", "fluid"} {
		_, count, in = cells(conduit.Of(kind, 2))
		if count != 1 || join(in) != "skygear:conduit_"+kind+"_1 skygear:ingot_electrum " {
			t.Errorf("%s conduit tier 2: %d from %s", kind, count, join(in))
		}
		_, count, in = cells(conduit.Of(kind, 3))
		if count != 1 || join(in) != "skygear:conduit_"+kind+"_2 skygear:ingot_steel minecraft:diamond " {
			t.Errorf("%s conduit tier 3: %d from %s", kind, count, join(in))
		}
	}
	if info := (sgitem.ConduitBinderMix{}).SmeltInfo(); info.Product.Item() != (sgitem.ConduitBinder{}) {
		t.Errorf("the mix bakes into %v", info.Product.Item())
	}
}

// TestFluidConduitRecipe: binders above and below a row of glass make eight
// basic fluid conduits; the research that unlocks it opens after the item conduit
// and glasswork researches.
func TestFluidConduitRecipe(t *testing.T) {
	var r *progress.Research
	for i := range progress.Tree {
		if progress.Tree[i].ID == "fluid_conduit" {
			r = &progress.Tree[i]
		}
	}
	if r == nil || len(r.Unlocks) == 0 || r.Unlocks[0] != "skygear:conduit_fluid_1" {
		t.Fatalf("fluid conduit research: %+v", r)
	}
	if a, ok := advancement.ByID("fluid_conduit"); !ok || a.Research != "fluid_conduit" {
		t.Errorf("fluid conduit advancement: %+v", a)
	}
}

// TestAutoSieveRecipeTakesAnySieve: the auto sieve's recipe names the sieve with
// the wildcard meta (the sieve has many block states), and a plain sieve matches.
func TestAutoSieveRecipeTakesAnySieve(t *testing.T) {
	var found bool
	for _, r := range recipe.Recipes() {
		out := r.Output()
		if len(out) != 1 || out[0].Item() != (machine.Machine{Kind: machine.KindAutoSieve}) {
			continue
		}
		for _, in := range r.Input() {
			if s, ok := in.(item.Stack); ok && s.Item() == (machine.Machine{Kind: machine.KindSieve}) {
				_, any := s.Value("variants")
				found = any
			}
		}
	}
	if !found {
		t.Error("the auto sieve recipe does not take the sieve with the wildcard meta")
	}
}

// TestShapelessRecipesFitTheGrid: a shapeless recipe puts one item in each
// cell of the 3×3 grid, so it cannot have more than nine ingredients.
func TestShapelessRecipesFitTheGrid(t *testing.T) {
	for _, r := range recipe.Recipes() {
		if _, ok := r.(recipe.Shapeless); !ok {
			continue
		}
		if n := len(r.Input()); n > 9 {
			name, _ := r.Output()[0].Item().EncodeItem()
			t.Errorf("shapeless recipe for %s has %d ingredients", name, n)
		}
	}
}
