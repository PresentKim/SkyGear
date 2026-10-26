// Package formicon names the form button icons of the game's menus.
package formicon

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Form button icons of menu entries that are no item: vanilla texture paths
// the client already has, and the game menu's own pictures
// (textures/ui/df/menu_*, apart from the lobby's icons). Items use Of.

const (
	Mission  = "textures/items/book_normal"
	Done     = "textures/ui/check"
	Island   = "textures/blocks/grass_side_carried"
	Lost     = "textures/ui/df/menu_lost"
	Lobby    = "textures/items/compass_item"
	Back     = "textures/items/arrow"
	Locked   = "textures/blocks/barrier"
	Title    = "textures/ui/df/menu_title"
	Friends  = "textures/ui/df/menu_friends"
	Settings = "textures/ui/df/menu_settings"
	Advances = "textures/ui/df/menu_advancements"
	Stats    = "textures/ui/df/menu_stats"
	Codex    = "textures/ui/df/menu_codex"
	Manage   = "textures/ui/df/menu_island"
	Expand   = "textures/ui/df/menu_expand"
)

// Of returns the form icon of an item: the item drawn by the client's
// item renderer (ui.ItemIcon), so every item and block shows as in the
// inventory without an icon texture.
func Of(it item.Stack) string {
	if it.Empty() {
		return ""
	}
	return ui.ItemIcon(it.Item())
}
