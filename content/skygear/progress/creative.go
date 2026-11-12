package progress

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory.
func creativeItems() []core.CreativeItem {
	return []core.CreativeItem{{Slot: 10, Category: creative.ConstructionCategory(), Item: ResearchTable{}}}
}
