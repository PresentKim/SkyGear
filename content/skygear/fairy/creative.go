package fairy

import (
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/presentkim/skygear/content/skygear/core"
)

// creativeItems are the feature's items in the creative inventory.
func creativeItems() []core.CreativeItem {
	return []core.CreativeItem{{Slot: 130, Category: creative.EquipmentCategory(), Item: Board{}}}
}
