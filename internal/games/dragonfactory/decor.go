package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/tinker"
)

// Slot decoration (docs/tech/slotdeco.md): the rarity and the tier of an item
// show as a coloured slot background and tier pips on its slot. The marks are
// part of the item's name: texts/*.lang for items that have no custom name of
// their own (tools/slotmarks writes them from ItemDecor), the custom name for
// stacks that have one (describeTool).

// ItemDecor returns the rarity and the tier badge of an item type: Common and
// 0 for the many items that have none. Rarity grows with how far into the
// game the item is (docs/design/open-questions.md).
func ItemDecor(it world.Item) (slotdeco.Rarity, int) {
	switch it := it.(type) {
	case dfitem.TierAlloy:
		for i, k := range dfitem.TierAlloys {
			if k == it.Kind {
				return slotdeco.Rarity(i + 1), i + 1
			}
		}
	case dfitem.TierKit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Epic))), it.Level
	case dfitem.Circuit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Legendary))), it.Level
	case dfitem.Mesh:
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
	case dfitem.Ingot:
		switch it.Metal {
		case dfitem.MetalBronze, dfitem.MetalSteel:
			return slotdeco.Uncommon, 0
		case dfitem.MetalElectrum, dfitem.MetalInvar:
			return slotdeco.Rare, 0
		}
	case dfitem.ResearchEssence:
		return slotdeco.Uncommon, 0
	case dfitem.Essence:
		if dfitem.MobDifficulty[it.Mob] >= 4 {
			return slotdeco.Epic, 0
		}
		return slotdeco.Rare, 0
	case dfitem.FairyEmerald, FairyBoard:
		return slotdeco.Rare, 0
	}
	return slotdeco.Common, 0
}

// tipName is the name of a stack for the first line of a form slot's
// tooltip: the slot marks in front and after (the pack draws a frame and
// badge from them, df_form.json gslot_deco), the name in the rarity colour.
func tipName(lang string, st item.Stack) string {
	if n := st.CustomName(); n != "" {
		return n // stacks with a name of their own carry their marks already
	}
	r, tier := ItemDecor(st.Item())
	return slotdeco.Decorate(r, tier, tinker.RarityColour(r)+naming.ItemName(lang, st))
}
