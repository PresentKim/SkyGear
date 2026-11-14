package sghook

// Recipe unlocking: the client's recipe book shows only the recipes the
// server says are unlocked (UnlockedRecipes). Dragonfly sends every recipe
// without an unlock requirement, so they all show; Sky Gear instead sends
// only the recipes the island has researched (Session.SetRecipeFilter) and
// resends the list on every research.
//
// Every recipe needs to be unlocked (UnlockRequirement with no ingredients),
// the list sent is unlocked, and recipes added since the last list are
// "newly unlocked", which the client shows as a notice.

import (
	"maps"
	"slices"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// recipeUnlockRule is the game rule that makes the recipe book show only
// unlocked recipes.
const recipeUnlockRule = "recipesunlock"

type recipeState struct {
	mu       sync.Mutex
	unlocked map[string]bool // nil before the first list
}

var recipeStates sync.Map // *Conn -> *recipeState

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		cd, ok := pk.(*packet.CraftingData)
		if !ok {
			return false
		}
		v, _ := recipeStates.LoadOrStore(c, &recipeState{})
		return v.(*recipeState).crafting(c, cd)
	})
	OnClose(func(c *Conn) { recipeStates.Delete(c) })
}

// crafting handles a recipe list on its way to c. It returns true if it
// sent what the client gets itself.
func (st *recipeState) crafting(c *Conn, pk *packet.CraftingData) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := recipeIDs(pk)
	first := st.unlocked == nil
	var out []packet.Packet
	if first {
		out = append(out, &packet.GameRulesChanged{GameRules: []protocol.GameRule{{Name: recipeUnlockRule, Value: true}}})
	}
	needUnlock(pk)
	out = append(out, pk)
	out = append(out, unlockDiff(st.unlocked, ids, first)...)
	st.unlocked = ids
	for _, p := range out {
		_ = c.Conn.WritePacket(p)
	}
	return true
}

// unlockDiff is what tells the client that now (not before) unlocks: the
// recipes still unlocked again (the client's list was replaced) and the
// added ones as newly unlocked.
func unlockDiff(before, now map[string]bool, first bool) []packet.Packet {
	var kept, added []string
	for id := range now {
		if before[id] {
			kept = append(kept, id)
		} else {
			added = append(added, id)
		}
	}
	slices.Sort(kept)
	slices.Sort(added)
	var out []packet.Packet
	add := func(t uint32, ids []string) {
		if len(ids) > 0 {
			out = append(out, &packet.UnlockedRecipes{UnlockType: t, Recipes: ids})
		}
	}
	if first {
		add(packet.UnlockedRecipesTypeInitiallyUnlocked, slices.Sorted(maps.Keys(now)))
		return out
	}
	add(packet.UnlockedRecipesTypeInitiallyUnlocked, kept)
	add(packet.UnlockedRecipesTypeNewlyUnlocked, added)
	return out
}

// recipeIDs are the names of the crafting recipes in pk.
func recipeIDs(pk *packet.CraftingData) map[string]bool {
	ids := map[string]bool{}
	for _, r := range pk.ShapedRecipes {
		ids[r.RecipeID] = true
	}
	for _, r := range pk.ShapelessRecipes {
		ids[r.RecipeID] = true
	}
	for _, r := range pk.UserDataShapelessRecipes {
		ids[r.RecipeID] = true
	}
	return ids
}

// needUnlock makes every crafting recipe in pk need an unlock: no
// ingredients unlock it, only UnlockedRecipes.
func needUnlock(pk *packet.CraftingData) {
	locked := protocol.Option(protocol.RecipeUnlockRequirement{Context: protocol.RecipeUnlockContextNone})
	for i := range pk.ShapedRecipes {
		pk.ShapedRecipes[i].UnlockRequirement = locked
	}
	for i := range pk.ShapelessRecipes {
		pk.ShapelessRecipes[i].UnlockRequirement = locked
	}
	for i := range pk.UserDataShapelessRecipes {
		pk.UserDataShapelessRecipes[i].UnlockRequirement = locked
	}
}
