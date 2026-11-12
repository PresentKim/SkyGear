package decor_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestItemDecor: an item says its own slot decoration (slotdeco.Decorated);
// an item that says nothing is common and has no tier.
func TestItemDecor(t *testing.T) {
	for _, c := range []struct {
		it   world.Item
		r    slotdeco.Rarity
		tier int
	}{
		{item.Stick{}, slotdeco.Common, 0},
		{sgitem.Hammer{}, slotdeco.Common, 0},
		{sgitem.Mesh{Tier: "string"}, slotdeco.Common, 0},
		{sgitem.Mesh{Tier: "gold"}, slotdeco.Rare, 0},
		{sgitem.TierKit{Level: 2}, slotdeco.Rare, 2},
		{sgitem.Circuit{Level: 4}, slotdeco.Legendary, 4},
		{sgitem.TierAlloy{Kind: sgitem.TierAlloys[0]}, slotdeco.Uncommon, 1},
		{sgitem.Ingot{Metal: sgitem.MetalSteel}, slotdeco.Uncommon, 0},
		{sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, slotdeco.Uncommon, 0},
		{sgitem.Relic{Kind: sgitem.RelicWeapons[0]}, slotdeco.Epic, 0},
		{sgitem.FairyEmerald{}, slotdeco.Rare, 0},
		{sgitem.DimensionStone{}, slotdeco.Rare, 0},
		{fairy.Board{}, slotdeco.Rare, 0},
	} {
		if r, tier := decor.ItemDecor(c.it); r != c.r || tier != c.tier {
			t.Errorf("%T%+v: rarity %d tier %d, want %d and %d", c.it, c.it, r, tier, c.r, c.tier)
		}
	}
}

// TestTipNameColour: the name on a slot's tooltip is in the colour of the
// item's rarity, between the rarity's marks.
func TestTipNameColour(t *testing.T) {
	for _, c := range []struct {
		it     world.Item
		colour string
	}{
		{item.Stick{}, "§f"},
		{sgitem.Ingot{Metal: sgitem.MetalSteel}, "§a"},
		{sgitem.Mesh{Tier: "gold"}, "§b"},
		{sgitem.Relic{Kind: sgitem.RelicWeapons[0]}, "§d"},
		{sgitem.Circuit{Level: 4}, "§6"},
	} {
		st := item.NewStack(c.it, 1)
		r, tier := decor.ItemDecor(c.it)
		want := slotdeco.Decorate(r, tier, c.colour+naming.UIItemName("ko", st))
		if got := decor.TipName("ko", st); got != want {
			t.Errorf("%T%+v: tip name %q, want %q", c.it, c.it, got, want)
		}
	}
}
