package skygear

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
	"github.com/presentkim/skygear/internal/skygear/tinker"
)

// Slot decoration: the rarity and the tier of an item
// show as a coloured slot background and tier pips on its slot. The marks are
// part of the item's name: texts/*.lang for items that have no custom name of
// their own (tools/slotmarks writes them from ItemDecor), the custom name for
// stacks that have one (tinker.DescribeTool).

// ItemDecor returns the rarity and the tier badge of an item type: Common and
// 0 for the many items that have none. Rarity grows with how far into the
// game the item is.
func ItemDecor(it world.Item) (slotdeco.Rarity, int) {
	switch it := it.(type) {
	case sgitem.TierAlloy:
		for i, k := range sgitem.TierAlloys {
			if k == it.Kind {
				return slotdeco.Rarity(i + 1), i + 1
			}
		}
	case sgitem.TierKit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Epic))), it.Level
	case sgitem.Circuit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Legendary))), it.Level
	case sgitem.Mesh:
		switch it.Tier {
		case "flint", "iron":
			return slotdeco.Uncommon, 0
		case "gold":
			return slotdeco.Rare, 0
		case "diamond":
			return slotdeco.Epic, 0
		case "netherite":
			return slotdeco.Legendary, 0
		}
	case sgitem.Ingot:
		switch it.Metal {
		case sgitem.MetalBronze, sgitem.MetalSteel:
			return slotdeco.Uncommon, 0
		case sgitem.MetalElectrum, sgitem.MetalInvar:
			return slotdeco.Rare, 0
		}
	case sgitem.ResearchEssence:
		return slotdeco.Uncommon, 0
	case sgitem.Essence:
		if sgitem.MobDifficulty[it.Mob] >= 4 {
			return slotdeco.Epic, 0
		}
		return slotdeco.Rare, 0
	case sgitem.FairyEmerald, FairyBoard, sgitem.FairyHourglass:
		return slotdeco.Rare, 0
	}
	return slotdeco.Common, 0
}

// tipName is the name of a stack for the first line of a screen slot's
// tooltip: the slot marks in front and after (common/ui draws the rarity's
// frame behind the slot from them), the name in the rarity colour.
func tipName(lang string, st item.Stack) string {
	if n := st.CustomName(); n != "" {
		return n // stacks with a name of their own carry their marks already
	}
	r, tier := ItemDecor(st.Item())
	return slotdeco.Decorate(r, tier, tinker.RarityColour(r)+naming.UIItemName(lang, st))
}
