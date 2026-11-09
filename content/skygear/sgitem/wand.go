package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// The construction wands (from the ConstructionWand mod, MIT): a click on a
// block places copies of it on the face clicked, across the whole face of
// the building. The stone, iron and diamond wands wear one point a block;
// the infinity wand never wears.

// Wand tiers.
const (
	WandStone    = "stone"
	WandIron     = "iron"
	WandDiamond  = "diamond"
	WandInfinity = "infinity"
)

// WandTiers are the wand tiers from the weakest.
var WandTiers = []string{WandStone, WandIron, WandDiamond, WandInfinity}

// wandStats are each tier's block limit and durability (0: never wears),
// those of the mod.
var wandStats = map[string]struct{ limit, durability int }{
	WandStone:    {9, 131},
	WandIron:     {27, 250},
	WandDiamond:  {128, 1561},
	WandInfinity: {1024, 0},
}

// Wand is a wand that wears: stone, iron or diamond.
type Wand struct{ Tier string }

// InfinityWand is the wand that never wears.
type InfinityWand struct{}

// WandItem is a construction wand of either kind.
type WandItem interface {
	world.Item
	// WandTier is the tier of the wand.
	WandTier() string
}

// WandOf returns the wand st is, if it is one.
func WandOf(st item.Stack) (WandItem, bool) {
	w, ok := st.Item().(WandItem)
	return w, ok
}

// IsWand reports whether st is a construction wand.
func IsWand(st item.Stack) bool {
	_, ok := WandOf(st)
	return ok
}

// WandLimit is how many blocks a wand of the tier places at most at once.
func WandLimit(tier string) int { return wandStats[tier].limit }

// NewWand returns the wand of a tier.
func NewWand(tier string) WandItem {
	if tier == WandInfinity {
		return InfinityWand{}
	}
	return Wand{Tier: tier}
}

func (w Wand) EncodeItem() (string, int16) { return "skygear:" + w.Tier + "_wand", 0 }

func (w Wand) Name() string { return i18n.T(i18n.Default, "name.Wand."+w.Tier) }

func (Wand) Category() category.Category { return category.Equipment() }

func (Wand) MaxCount() int { return 1 }

func (Wand) HandEquipped() bool { return true }

func (w Wand) WandTier() string { return w.Tier }

func (w Wand) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{MaxDurability: wandStats[w.Tier].durability, BrokenItem: func() item.Stack { return item.Stack{} }}
}

func (InfinityWand) EncodeItem() (string, int16) { return "skygear:infinity_wand", 0 }

func (InfinityWand) Name() string { return i18n.T(i18n.Default, "name.Wand.infinity") }

func (InfinityWand) Category() category.Category { return category.Equipment() }

func (InfinityWand) MaxCount() int { return 1 }

func (InfinityWand) HandEquipped() bool { return true }

func (InfinityWand) WandTier() string { return WandInfinity }

func init() {
	for _, t := range WandTiers {
		world.RegisterItem(NewWand(t))
	}
}
