package sgitem

import "github.com/presentkim/skygear/common/slotdeco"

// The slot decoration of the shared items (slotdeco.Decorated): the rarity
// grows with how far into the game the item is, and the tier shows as pips.

func (a TierAlloy) Decor() (slotdeco.Rarity, int) {
	for i, k := range TierAlloys {
		if k == a.Kind {
			return slotdeco.Rarity(i + 1), i + 1
		}
	}
	return slotdeco.Common, 0
}

func (k TierKit) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(min(k.Level, int(slotdeco.Epic))), k.Level
}

func (c Circuit) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(min(c.Level, int(slotdeco.Legendary))), c.Level
}

func (m Mesh) Decor() (slotdeco.Rarity, int) {
	switch m.Tier {
	case "flint", "iron":
		return slotdeco.Uncommon, 0
	case "gold":
		return slotdeco.Rare, 0
	case "diamond":
		return slotdeco.Epic, 0
	case "netherite":
		return slotdeco.Legendary, 0
	}
	return slotdeco.Common, 0
}

func (i Ingot) Decor() (slotdeco.Rarity, int) {
	switch i.Metal {
	case MetalBronze, MetalSteel:
		return slotdeco.Uncommon, 0
	case MetalElectrum, MetalInvar:
		return slotdeco.Rare, 0
	}
	return slotdeco.Common, 0
}

// The level shows as pips; the higher levels in rarer colours.
func (c CompactBlock) Decor() (slotdeco.Rarity, int) {
	return slotdeco.Rarity(c.Level - 1), c.Level
}

func (ResearchEssence) Decor() (slotdeco.Rarity, int) { return slotdeco.Uncommon, 0 }

func (e Essence) Decor() (slotdeco.Rarity, int) {
	if MobDifficulty[e.Mob] >= 4 {
		return slotdeco.Epic, 0
	}
	return slotdeco.Rare, 0
}

func (Relic) Decor() (slotdeco.Rarity, int) { return slotdeco.Epic, 0 }

func (FairyEmerald) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }

func (FairyHourglass) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }

func (DimensionStone) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }
