// Package decor is the slot decoration of items: the rarity and the tier
// of an item show as a coloured slot background and tier pips on its slot.
// The marks are part of the item's name: texts/*.lang for items that have no
// custom name of their own (tools/slotmarks writes them from ItemDecor), the
// custom name for stacks that have one (tinker.DescribeTool).
package decor

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// ItemDecor returns the rarity and the tier badge of an item type: Common and
// 0 for the many items that have none. Rarity grows with how far into the
// game the item is.
func ItemDecor(it world.Item) (slotdeco.Rarity, int) {
	if d, ok := it.(slotdeco.Decorated); ok {
		return d.Decor()
	}
	return slotdeco.Common, 0
}

// TipName is the name of a stack for the first line of a screen slot's
// tooltip: the slot marks in front and after (common/ui draws the rarity's
// frame behind the slot from them), the name in the rarity colour.
func TipName(lang string, st item.Stack) string {
	if n := st.CustomName(); n != "" {
		return n // stacks with a name of their own carry their marks already
	}
	r, tier := ItemDecor(st.Item())
	return slotdeco.Decorate(r, tier, tinker.RarityColour(r)+naming.UIItemName(lang, st))
}
