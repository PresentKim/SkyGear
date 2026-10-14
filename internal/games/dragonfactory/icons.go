package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Form button icons of menu entries that are no item: vanilla texture paths
// the client already has. Items use iconOf.

const (
	iconMission = "textures/items/book_normal"
	iconIsland  = "textures/blocks/grass_side_carried"
	iconLost    = "textures/blocks/chest_front"
	iconMenu    = "textures/items/nether_star"
	iconLobby   = "textures/items/compass_item"
	iconBack    = "textures/items/arrow"
	iconLocked  = "textures/blocks/barrier"
	iconAll     = "textures/blocks/hopper_top"
	iconTitle   = "textures/items/name_tag"
)

// iconOf returns the form icon of an item: the item drawn by the client's
// item renderer (ui.ItemIcon), so every item and block shows as in the
// inventory without an icon texture.
func iconOf(it item.Stack) string {
	if it.Empty() {
		return ""
	}
	return ui.ItemIcon(it.Item())
}
