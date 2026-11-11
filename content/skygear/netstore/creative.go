package netstore

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory: the
// disks, the pattern and the upgrades.
func creativeItems() []core.CreativeItem {
	var out []core.CreativeItem
	items := creative.ItemsCategory()
	for t := range DiskCaps {
		out = append(out, core.CreativeItem{Slot: 110, Category: items, Item: Disk{Tier: t}})
	}
	return append(out,
		core.CreativeItem{Slot: 110, Category: items, Item: Pattern{}},
		core.CreativeItem{Slot: 110, Category: items, Item: Upgrade{}},
		core.CreativeItem{Slot: 110, Category: items, Item: Upgrade{Stack: true}},
	)
}
