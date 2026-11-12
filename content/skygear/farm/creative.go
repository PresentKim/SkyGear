package farm

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// barrels, the silkworms and the seeds of the magic plants.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	for _, w := range BarrelWoods {
		out = append(out, core.CreativeItem{Slot: 30, Category: creative.ConstructionCategory(), Item: Barrel{Wood: w}})
	}
	out = append(out,
		core.CreativeItem{Slot: 90, Category: creative.ItemsCategory(), Item: Silkworm{}},
		core.CreativeItem{Slot: 90, Category: creative.ItemsCategory(), Item: CookedSilkworm{}},
	)
	for _, sd := range PlantSeeds {
		out = append(out, core.CreativeItem{Slot: 150, Category: creative.ItemsCategory(), Item: PlantSeed{Seed: sd}})
	}
	return out
}
