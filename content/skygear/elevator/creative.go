package elevator

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// elevator in every colour.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, c := range item.Colours() {
		out = append(out, core.CreativeItem{Slot: 70, Category: creative.ConstructionCategory(), Item: Elevator{Colour: c}})
	}
	return out
}
