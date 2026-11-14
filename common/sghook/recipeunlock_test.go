package sghook

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func set(ids ...string) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func unlocks(pks []packet.Packet) map[uint32][]string {
	m := map[uint32][]string{}
	for _, pk := range pks {
		u := pk.(*packet.UnlockedRecipes)
		m[u.UnlockType] = u.Recipes
	}
	return m
}

func TestUnlockDiff(t *testing.T) {
	first := unlocks(unlockDiff(nil, set("a", "b"), true))
	if len(first) != 1 || len(first[packet.UnlockedRecipesTypeInitiallyUnlocked]) != 2 {
		t.Fatalf("first %v", first)
	}
	// The list was replaced, so the kept recipes are listed again and
	// the new one is newly unlocked.
	next := unlocks(unlockDiff(set("a", "b"), set("a", "b", "c"), false))
	if got := next[packet.UnlockedRecipesTypeInitiallyUnlocked]; len(got) != 2 {
		t.Fatalf("kept %v", got)
	}
	if got := next[packet.UnlockedRecipesTypeNewlyUnlocked]; len(got) != 1 || got[0] != "c" {
		t.Fatalf("added %v", got)
	}
	if got := unlockDiff(set("a"), set("a"), false); len(got) != 1 || got[0].(*packet.UnlockedRecipes).UnlockType != packet.UnlockedRecipesTypeInitiallyUnlocked {
		t.Fatalf("unchanged list is sent again %v", got)
	}
}

func TestNeedUnlock(t *testing.T) {
	pk := &packet.CraftingData{
		ShapedRecipes:    []protocol.ShapedRecipe{{RecipeID: "a", RecipeNetworkID: 1}},
		ShapelessRecipes: []protocol.ShapelessRecipe{{RecipeID: "b", RecipeNetworkID: 2}},
	}
	needUnlock(pk)
	for _, r := range []protocol.Optional[protocol.RecipeUnlockRequirement]{pk.ShapedRecipes[0].UnlockRequirement, pk.ShapelessRecipes[0].UnlockRequirement} {
		if u, ok := r.Value(); !ok || u.Context != protocol.RecipeUnlockContextNone || len(u.Ingredients) != 0 {
			t.Fatalf("requirement %+v", r)
		}
	}
	if ids := recipeIDs(pk); !ids["a"] || !ids["b"] || len(ids) != 2 {
		t.Fatalf("ids %v", ids)
	}
}
