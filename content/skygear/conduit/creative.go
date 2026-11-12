package conduit

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// conduits of every tier and the network conduit.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	construction := creative.ConstructionCategory()
	for t := 1; t <= Tiers; t++ {
		out = append(out,
			core.CreativeItem{Slot: 40, Category: construction, Item: EnergyConduit{Tier: t}},
			core.CreativeItem{Slot: 40, Category: construction, Item: ItemConduit{Tier: t}},
			core.CreativeItem{Slot: 40, Category: construction, Item: FluidConduit{Tier: t}},
		)
	}
	return append(out, core.CreativeItem{Slot: 40, Category: construction, Item: NetConduit{}})
}
