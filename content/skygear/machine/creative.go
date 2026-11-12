package machine

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: every
// machine kind in Order.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, k := range Order {
		out = append(out, core.CreativeItem{Slot: 50, Category: creative.ConstructionCategory(), Item: Machine{Kind: k}})
	}
	return out
}
