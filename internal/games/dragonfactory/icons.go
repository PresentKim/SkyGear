package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Form button icons of menu entries that are no item: vanilla texture paths
// the client already has, and the game menu's own pictures
// (textures/ui/df/menu_*, apart from the lobby's icons). Items use iconOf.

const (
	iconMission  = "textures/items/book_normal"
	iconDone     = "textures/ui/check"
	iconIsland   = "textures/blocks/grass_side_carried"
	iconLost     = "textures/ui/df/menu_lost"
	iconLobby    = "textures/items/compass_item"
	iconBack     = "textures/items/arrow"
	iconLocked   = "textures/blocks/barrier"
	iconTitle    = "textures/ui/df/menu_title"
	iconFriends  = "textures/ui/df/menu_friends"
	iconSettings = "textures/ui/df/menu_settings"
	iconAdvances = "textures/ui/df/menu_advancements"
	iconStats    = "textures/ui/df/menu_stats"
	iconCodex    = "textures/ui/df/menu_codex"
	iconManage   = "textures/ui/df/menu_island"
	iconExpand   = "textures/ui/df/menu_expand"
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
