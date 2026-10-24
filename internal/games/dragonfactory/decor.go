package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/slotdeco"
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
	case TierAlloy:
		for i, k := range tierAlloys {
			if k == it.Kind {
				return slotdeco.Rarity(i + 1), i + 1
			}
		}
	case TierKit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Epic))), it.Level
	case Circuit:
		return slotdeco.Rarity(min(it.Level, int(slotdeco.Legendary))), it.Level
	case Mesh:
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
	case Ingot:
		switch it.Metal {
		case MetalBronze, MetalSteel:
			return slotdeco.Uncommon, 0
		case MetalElectrum, MetalInvar:
			return slotdeco.Rare, 0
		}
	case ResearchEssence:
		return slotdeco.Uncommon, 0
	case Essence:
		if mobDifficulty[it.Mob] >= 4 {
			return slotdeco.Epic, 0
		}
		return slotdeco.Rare, 0
	}
	return slotdeco.Common, 0
}

// toolRarity is the rarity of a tool of the level: a tool that has been used
// a lot looks worth more.
func toolRarity(level int) slotdeco.Rarity {
	switch {
	case level >= 10:
		return slotdeco.Legendary
	case level >= 8:
		return slotdeco.Epic
	case level >= 5:
		return slotdeco.Rare
	case level >= 3:
		return slotdeco.Uncommon
	}
	return slotdeco.Common
}

// rarityColour is the colour code of a rarity's name ("§f" for Common).
func rarityColour(r slotdeco.Rarity) string {
	switch r {
	case slotdeco.Uncommon:
		return "§a"
	case slotdeco.Rare:
		return "§b"
	case slotdeco.Epic:
		return "§d"
	case slotdeco.Legendary:
		return "§6"
	}
	return "§f"
}

// tipName is the name of a stack for the first line of a form slot's
// tooltip: the slot marks in front and after (the pack draws a frame and
// badge from them, df_form.json gslot_deco), the name in the rarity colour.
func tipName(lang string, st item.Stack) string {
	if n := st.CustomName(); n != "" {
		return n // stacks with a name of their own carry their marks already
	}
	r, tier := ItemDecor(st.Item())
	return slotdeco.Decorate(r, tier, rarityColour(r)+itemName(lang, st))
}
