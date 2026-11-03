package session

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var testRecipesRegistered bool

func registerTestRecipes() {
	if testRecipesRegistered {
		return
	}
	testRecipesRegistered = true
	world.DefaultBlockRegistry.Finalize()
	c, s := item.NewStack(block.Cobblestone{}, 1), item.NewStack(item.Stick{}, 1)
	e := item.Stack{}
	recipe.Register(recipe.NewShaped([]recipe.Item{c, c, e, c, s, e, e, s, e}, item.NewStack(item.Stick{}, 2), recipe.NewShape(3, 3), "crafting_table"))
	recipe.Register(recipe.NewShapeless([]recipe.Item{c, s}, item.NewStack(item.Stick{}, 1), "crafting_table"))
	recipe.Register(recipe.NewShaped([]recipe.Item{s, s}, item.NewStack(block.Cobblestone{}, 1), recipe.NewShape(1, 2), "crafting_table").WithoutMirror())
	recipe.Register(recipe.NewShapeless([]recipe.Item{s}, item.NewStack(block.Cobblestone{}, 3), "crafting_table"))
}

// legacyCraftingData is the pre-cache construction, with recipe IDs blanked.
func legacyCraftingData(br world.BlockRegistry, filter func(recipe.Recipe) bool) (*packet.CraftingData, map[uint32]recipe.Recipe) {
	pk := &packet.CraftingData{ClearRecipes: true}
	accepted := map[uint32]recipe.Recipe{}
	fit := creativeBlockFitter(br)
	for index, r := range recipe.Recipes() {
		networkID := uint32(index) + 1
		if filter != nil && !filter(r) {
			continue
		}
		accepted[networkID] = r
		switch i := r.(type) {
		case recipe.Shapeless:
			pk.ShapelessRecipes = append(pk.ShapelessRecipes, protocol.ShapelessRecipe{
				Priority: int32(i.Priority()), Input: stacksToIngredientItems(br, i.Input()),
				Output: fit(stacksToRecipeStacks(br, i.Output())), Block: i.Block(), RecipeNetworkID: networkID,
			})
		case recipe.Shaped:
			pk.ShapedRecipes = append(pk.ShapedRecipes, protocol.ShapedRecipe{
				Priority: int32(i.Priority()), Width: int32(i.Shape().Width()), Height: int32(i.Shape().Height()),
				Input:  stacksToIngredientItems(br, i.Input()),
				Output: fit(stacksToRecipeStacks(br, i.Output())), Block: i.Block(),
				AssumeSymmetry: !i.NoMirror(), RecipeNetworkID: networkID,
			})
		}
	}
	return pk, accepted
}

func blankIDs(pk *packet.CraftingData) {
	for i := range pk.ShapedRecipes {
		pk.ShapedRecipes[i].RecipeID = ""
	}
	for i := range pk.ShapelessRecipes {
		pk.ShapelessRecipes[i].RecipeID = ""
	}
}

// TestRecipeFilterSameEntries: the cached build gives the old output for no
// filter and for a filter (same network IDs, same recipes).
func TestRecipeFilterSameEntries(t *testing.T) {
	registerTestRecipes()
	br := world.DefaultBlockRegistry
	all := recipe.Recipes()
	if len(all) < 4 {
		t.Fatal("no recipes registered")
	}
	odd := func(r recipe.Recipe) bool {
		return r.Block() == "crafting_table" && len(r.Output()) > 0 && r.Output()[0].Count()%2 == 1
	}
	for name, f := range map[string]func(recipe.Recipe) bool{"nil": nil, "odd": odd} {
		want, wantAcc := legacyCraftingData(br, f)
		acc := map[uint32]recipe.Recipe{}
		got := craftingData(br, f, acc)
		for _, r := range got.ShapedRecipes {
			if r.RecipeID == "" {
				t.Errorf("%s: empty recipe ID", name)
			}
		}
		blankIDs(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: crafting data differs", name)
		}
		if !reflect.DeepEqual(acc, wantAcc) {
			t.Errorf("%s: accepted recipes differ", name)
		}
		again := craftingData(br, f, map[uint32]recipe.Recipe{})
		ids := func(p *packet.CraftingData) (n []string) {
			for _, r := range p.ShapedRecipes {
				n = append(n, r.RecipeID)
			}
			return
		}
		blankedAgain := ids(again)
		if len(blankedAgain) > 0 && blankedAgain[0] == "" {
			t.Errorf("%s: id lost", name)
		}
	}
	if odd(all[0]) == odd(all[1]) && odd(all[1]) == odd(all[2]) && odd(all[2]) == odd(all[3]) {
		t.Error("filter does not discriminate")
	}
}

func BenchmarkSendRecipes(b *testing.B) {
	registerTestRecipes()
	br := world.DefaultBlockRegistry
	acc := map[uint32]recipe.Recipe{}
	b.ReportAllocs()
	for b.Loop() {
		craftingData(br, nil, acc)
	}
}

func BenchmarkSendRecipesLegacy(b *testing.B) {
	registerTestRecipes()
	b.ReportAllocs()
	for b.Loop() {
		legacyCraftingData(world.DefaultBlockRegistry, nil)
	}
}
