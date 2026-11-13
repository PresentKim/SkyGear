package sghook

// Recipe unlocking test (platform.toml [Experimental] RecipeUnlock): the
// client's recipe book shows only the recipes the server says are unlocked
// (UnlockedRecipes). Dragonfly sends every recipe without an unlock
// requirement, so they all show; Sky Gear instead sends only the recipes the
// island has researched (Session.SetRecipeFilter) and resends the list on
// every research.
//
// "toast" keeps that list and adds the unlock packets: recipes need to be
// unlocked (UnlockRequirement with no ingredients), the list sent is
// unlocked, and recipes added since the last list are "newly unlocked",
// which the client shows as a notice.
//
// "book" sends the full list once and from then on only the unlock packets:
// the recipes left out by the filter are locked instead of missing. The
// server still refuses to craft them. It tests whether the client keeps a
// locked recipe out of the crafting grid too.

import (
	"maps"
	"slices"
	"sync"
	_ "unsafe" // go:linkname

	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The modes of RecipeUnlock.
const (
	RecipeUnlockToast = "toast"
	RecipeUnlockBook  = "book"
)

// RecipeUnlock is the recipe unlocking test mode: "" (off), "toast" or
// "book". Set it once at startup.
var RecipeUnlock string

// recipeUnlockRule is the game rule that makes the recipe book show only
// unlocked recipes.
const recipeUnlockRule = "recipesunlock"

type recipeState struct {
	mu       sync.Mutex
	unlocked map[string]bool // nil before the first list
	full     bool            // the full list was sent (book)
}

var recipeStates sync.Map // *Conn -> *recipeState

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		cd, ok := pk.(*packet.CraftingData)
		if !ok || (RecipeUnlock != RecipeUnlockToast && RecipeUnlock != RecipeUnlockBook) {
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
	switch {
	case RecipeUnlock == RecipeUnlockBook && st.full:
		// The client has every recipe: only the unlocks change.
	case RecipeUnlock == RecipeUnlockBook:
		// Before the session is known (the first list at spawn) the
		// filtered list goes as it is, and the full one with the next.
		full := fullCraftingData(c)
		st.full = full != nil
		if full == nil {
			full = pk
		}
		needUnlock(full)
		out = append(out, full)
	default:
		needUnlock(pk)
		out = append(out, pk)
	}
	out = append(out, unlockDiff(st.unlocked, ids, first, RecipeUnlock == RecipeUnlockToast)...)
	st.unlocked = ids
	for _, p := range out {
		_ = c.Conn.WritePacket(p)
	}
	return true
}

// unlockDiff is what tells the client that now (not before) unlocks:
// relist sends the recipes still unlocked again (the client's list was
// replaced); otherwise the removed ones are locked.
func unlockDiff(before, now map[string]bool, first, relist bool) []packet.Packet {
	var kept, added, removed []string
	for id := range now {
		if before[id] {
			kept = append(kept, id)
		} else {
			added = append(added, id)
		}
	}
	for id := range before {
		if !now[id] {
			removed = append(removed, id)
		}
	}
	slices.Sort(kept)
	slices.Sort(added)
	slices.Sort(removed)
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
	if relist {
		add(packet.UnlockedRecipesTypeInitiallyUnlocked, kept)
	} else {
		add(packet.UnlockedRecipesTypeRemoveUnlocked, removed)
	}
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

// fullCraftingData is the list of every recipe for c's session, or nil if c
// has no session yet.
func fullCraftingData(c *Conn) *packet.CraftingData {
	s := c.Session()
	if s == nil {
		return nil
	}
	return session_craftingData(*field[world.BlockRegistry](s, "br"), nil, map[uint32]recipe.Recipe{})
}

//go:linkname session_craftingData github.com/df-mc/dragonfly/server/session.craftingData
func session_craftingData(world.BlockRegistry, func(recipe.Recipe) bool, map[uint32]recipe.Recipe) *packet.CraftingData
