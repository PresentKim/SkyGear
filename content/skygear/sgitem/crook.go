package sgitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// The crook, after the crook of Ex Nihilo Fabrico (SirLyle, MIT): a hooked
// stick that pulls leaves apart. Leaves broken with it drop saplings more
// often than by hand and now and then a silkworm (farm.CrookLeafDrops);
// fully infested leaves drop more string (farm.InfestedLeaves). It breaks
// leaves quickly and wears a point a leaf. The pictures are Fabrico's.

// Crook tiers.
const (
	CrookWood    = "wooden"
	CrookStone   = "stone"
	CrookIron    = "iron"
	CrookDiamond = "diamond"
)

// CrookTier is what a crook of one material does. Balancing drafts
// (docs/design/open-questions.md).
type CrookTier struct {
	Name string
	// Tier is the tool tier of the material: durability and mining speed.
	Tier item.ToolTier
	// Sapling is the chance a leaf drops a sapling (jungle leaves: half),
	// Silkworm the chance it drops a silkworm.
	Sapling, Silkworm float64
}

// CrookTiers are the crooks from the wooden one up.
var CrookTiers = []CrookTier{
	{CrookWood, item.ToolTierWood, 0.3, 0.03},
	{CrookStone, item.ToolTierStone, 0.35, 0.04},
	{CrookIron, item.ToolTierIron, 0.4, 0.05},
	{CrookDiamond, item.ToolTierDiamond, 0.45, 0.06},
}

// Crook is a crook of a tier (the zero value is the wooden one).
type Crook struct{ Tier string }

// Stats returns the tier of the crook.
func (c Crook) Stats() CrookTier {
	for _, t := range CrookTiers {
		if t.Name == c.Tier {
			return t
		}
	}
	return CrookTiers[0]
}

func (c Crook) EncodeItem() (string, int16) { return "skygear:" + c.Stats().Name + "_crook", 0 }

func (c Crook) Name() string { return i18n.T(i18n.Default, "name.Crook."+c.Stats().Name) }

func (Crook) Category() category.Category { return category.Equipment() }

func (Crook) MaxCount() int { return 1 }

func (Crook) HandEquipped() bool { return true }

func (c Crook) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability: c.Stats().Tier.Durability, BrokenItem: func() item.Stack { return item.Stack{} },
		AttackDurability: 2, BreakDurability: 1,
	}
}

// ToolType makes the crook a hoe for the server: leaves count it as their
// tool and break fast. The hoe's own use (tilling) is item.Hoe's, which the
// crook is not.
func (Crook) ToolType() item.ToolType { return item.TypeHoe }

func (c Crook) HarvestLevel() int { return c.Stats().Tier.HarvestLevel }

func (c Crook) BaseMiningEfficiency(b world.Block) float64 {
	if IsCrookBlock(b) {
		return c.Stats().Tier.BaseMiningEfficiency
	}
	return 1
}

// DestroySpeeds implements item.Digger: leaves (and the infested leaves) at
// the material's speed, so the client breaks them as fast as the server.
func (c Crook) DestroySpeeds() map[string]int {
	out := map[string]int{}
	speed := int(c.Stats().Tier.BaseMiningEfficiency)
	for _, t := range block.LeavesTypes() {
		out["minecraft:"+t.String()] = speed
	}
	for _, name := range CrookBlockNames {
		out[name] = speed
	}
	return out
}

// CrookBlockNames are the custom blocks the crook is made for besides the
// leaves (the farm package's infested leaves add theirs).
var CrookBlockNames []string

// IsCrookBlock reports whether b is leaves or a block the crook is made for.
func IsCrookBlock(b world.Block) bool {
	if _, ok := b.(block.Leaves); ok {
		return true
	}
	if it, ok := b.(world.Item); ok {
		name, _ := it.EncodeItem()
		for _, n := range CrookBlockNames {
			if n == name {
				return true
			}
		}
	}
	return false
}

// CrookOf returns the crook held in st.
func CrookOf(st item.Stack) (Crook, bool) {
	c, ok := st.Item().(Crook)
	return c, ok
}
