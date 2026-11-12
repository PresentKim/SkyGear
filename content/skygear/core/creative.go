package core

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
)

// CreativeItem is one item of the game's part of the creative inventory.
type CreativeItem struct {
	// Slot is the item's place in the list: items sort by Slot, and the items
	// of one Slot keep the order their owner gave them in. A Slot belongs to
	// one owner.
	Slot int
	// Category is the creative tab the item is in.
	Category creative.Category
	// Item is the item, listed as a stack of one.
	Item world.Item
}

// CreativeItems returns own, the items of no feature, and the Creative items
// of every installed feature in one list ordered by Slot. It panics when two
// owners use the same Slot.
func CreativeItems(own []CreativeItem) []CreativeItem {
	owner := map[int]string{}
	var out []CreativeItem
	add := func(name string, items []CreativeItem) {
		mine := map[int]bool{}
		for _, it := range items {
			if o, taken := owner[it.Slot]; taken && !mine[it.Slot] {
				panic(fmt.Sprintf("core: creative slot %d is used by %s and by %s", it.Slot, o, name))
			}
			owner[it.Slot], mine[it.Slot] = name, true
		}
		out = append(out, items...)
	}
	add("the content list", own)
	for _, f := range Features() {
		if f.Creative != nil {
			add("feature "+f.Name, f.Creative())
		}
	}
	slices.SortStableFunc(out, func(a, b CreativeItem) int { return a.Slot - b.Slot })
	return out
}
