package waila

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// toolKind is a tool type and the panel icons of its tiers (one icon for
// untiered tools).
type toolKind struct {
	typ   item.ToolType
	make  func(item.ToolTier) item.Tool
	icons []Tool
}

var tiers = []item.ToolTier{item.ToolTierWood, item.ToolTierStone, item.ToolTierIron, item.ToolTierDiamond}

// toolKinds in the order the panel prefers them.
var toolKinds = []toolKind{
	{item.TypePickaxe, func(t item.ToolTier) item.Tool { return item.Pickaxe{Tier: t} }, []Tool{PickaxeWood, PickaxeStone, PickaxeIron, PickaxeDiamond}},
	{item.TypeAxe, func(t item.ToolTier) item.Tool { return item.Axe{Tier: t} }, []Tool{AxeWood, AxeStone, AxeIron, AxeDiamond}},
	{item.TypeShovel, func(t item.ToolTier) item.Tool { return item.Shovel{Tier: t} }, []Tool{Shovel}},
	{item.TypeHoe, func(t item.ToolTier) item.Tool { return item.Hoe{Tier: t} }, []Tool{Hoe}},
	{item.TypeShears, func(item.ToolTier) item.Tool { return item.Shears{} }, []Tool{Shears}},
	{item.TypeSword, func(t item.ToolTier) item.Tool { return item.Sword{Tier: t} }, []Tool{Sword}},
}

// ToolsFor returns up to two tool icons for mining b: the tools that mine
// it faster than a hand, with the lowest tier that still gets its drops.
// The mark tells whether held is such a tool (and good enough), when the
// block needs one.
func ToolsFor(b world.Block, held item.Stack) []ToolIcon {
	br, ok := b.(block.Breakable)
	if !ok {
		return nil
	}
	info := br.BreakInfo()
	if info.Effective == nil || info.Harvestable == nil {
		return nil
	}
	heldTool, _ := held.Item().(item.Tool)
	needsTool := !info.Harvestable(item.ToolNone{})
	var out []ToolIcon
	for _, k := range toolKinds {
		if !info.Effective(k.make(item.ToolTierDiamond)) {
			continue
		}
		icon := ToolNone
		for i, t := range tiers {
			if info.Harvestable(k.make(t)) {
				icon = k.icons[min(i, len(k.icons)-1)]
				break
			}
		}
		if icon == ToolNone {
			continue
		}
		mark := MarkNone
		switch {
		case heldTool != nil && heldTool.ToolType() == k.typ && info.Harvestable(heldTool):
			mark = MarkRight
		case needsTool:
			mark = MarkWrong
		}
		out = append(out, ToolIcon{icon, mark})
		if len(out) == 2 {
			break
		}
	}
	return out
}
