// Package machinerecipe holds the recipes of the alloy smelter, the
// laboratory and the auto crafter, and how inputs match them.
package machinerecipe

import (
	"reflect"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Alloy is one recipe of the alloy smelter.
type Alloy struct {
	A, B           world.Item
	CountA, CountB int
	Out            world.Item
	OutCount       int
}

// Alloys are the recipes, drafts.
var Alloys = []Alloy{
	{A: item.CopperIngot{}, CountA: 3, B: sgitem.Ingot{Metal: sgitem.MetalTin}, CountB: 1, Out: sgitem.Ingot{Metal: sgitem.MetalBronze}, OutCount: 4},
	{A: item.GoldIngot{}, CountA: 1, B: sgitem.Ingot{Metal: sgitem.MetalSilver}, CountB: 1, Out: sgitem.Ingot{Metal: sgitem.MetalElectrum}, OutCount: 2},
	{A: item.IronIngot{}, CountA: 2, B: sgitem.Ingot{Metal: sgitem.MetalNickel}, CountB: 1, Out: sgitem.Ingot{Metal: sgitem.MetalInvar}, OutCount: 3},
	{A: item.IronIngot{}, CountA: 1, B: item.Coal{}, CountB: 4, Out: sgitem.Ingot{Metal: sgitem.MetalSteel}, OutCount: 1},
	// the alloys of the machine tiers (tier.go)
	{A: item.IronIngot{}, CountA: 1, B: item.RedstoneWire{}, CountB: 4, Out: sgitem.TierAlloy{Kind: "infused"}, OutCount: 2},
	{A: sgitem.TierAlloy{Kind: "infused"}, CountA: 2, B: item.Diamond{}, CountB: 1, Out: sgitem.TierAlloy{Kind: "reinforced"}, OutCount: 2},
	{A: sgitem.TierAlloy{Kind: "reinforced"}, CountA: 2, B: block.Obsidian{}, CountB: 2, Out: sgitem.TierAlloy{Kind: "atomic"}, OutCount: 2},
}

// itemID is the name and metadata an item encodes as.
type itemID struct {
	name string
	meta int16
}

func idOf(it world.Item) itemID {
	name, meta := it.EncodeItem()
	return itemID{name, meta}
}

// alloyIDs are the encoded names of the recipes' items, in Alloys order.
var alloyIDs = func() [][2]itemID {
	ids := make([][2]itemID, len(Alloys))
	for i, r := range Alloys {
		ids[i] = [2]itemID{idOf(r.A), idOf(r.B)}
	}
	return ids
}()

// AlloyAccepts reports whether an item is an ingredient of some recipe.
func AlloyAccepts(it item.Stack) bool {
	k := idOf(it.Item())
	for _, r := range alloyIDs {
		if r[0] == k || r[1] == k {
			return true
		}
	}
	return false
}

// hasItem reports whether have, held n or more times by a stack, is want.
// Item types name themselves from their fields, so two values of one
// comparable type are the same item exactly when they are equal; that check
// does not build the encoded name (which allocates for metals). Other types
// are told apart by type, and only uncomparable ones by their encoded name.
func hasItem(have world.Item, count int, want world.Item, n int) bool {
	if have == nil || count < n {
		return false
	}
	t := reflect.TypeOf(have)
	if t != reflect.TypeOf(want) {
		return false
	}
	if t.Comparable() {
		return have == want
	}
	return idOf(have) == idOf(want)
}

// MatchAlloy returns the recipe the two slots make (in either order), if any.
func MatchAlloy(a, b item.Stack) (Alloy, bool) {
	// Item is read once per slot: it checks Empty, which encodes the name
	ia, ib := a.Item(), b.Item()
	ca, cb := a.Count(), b.Count()
	for _, r := range Alloys {
		if (hasItem(ia, ca, r.A, r.CountA) && hasItem(ib, cb, r.B, r.CountB)) || (hasItem(ia, ca, r.B, r.CountB) && hasItem(ib, cb, r.A, r.CountA)) {
			return r, true
		}
	}
	return Alloy{}, false
}
