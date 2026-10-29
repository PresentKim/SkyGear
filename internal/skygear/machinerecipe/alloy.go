// Package machinerecipe holds the recipes of the alloy smelter, the
// laboratory and the auto crafter, and how inputs match them.
package machinerecipe

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
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

// AlloyAccepts reports whether an item is an ingredient of some recipe.
func AlloyAccepts(it item.Stack) bool {
	k := naming.ItemKey(it.Item())
	for _, r := range Alloys {
		if naming.ItemKey(r.A) == k || naming.ItemKey(r.B) == k {
			return true
		}
	}
	return false
}

// MatchAlloy returns the recipe the two slots make (in either order), if any.
func MatchAlloy(a, b item.Stack) (Alloy, bool) {
	has := func(it item.Stack, want world.Item, n int) bool {
		return !it.Empty() && naming.ItemKey(it.Item()) == naming.ItemKey(want) && it.Count() >= n
	}
	for _, r := range Alloys {
		if (has(a, r.A, r.CountA) && has(b, r.B, r.CountB)) || (has(a, r.B, r.CountB) && has(b, r.A, r.CountA)) {
			return r, true
		}
	}
	return Alloy{}, false
}
