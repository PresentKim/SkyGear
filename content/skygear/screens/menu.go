package screens

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
)

const menuItemKey = "skygear_menu"

// MenuItem opens the game menu when used. Its name is written on it in lang
// (a name of a stack is text, not a translation key); renameMenuItems names
// the menu items a player has in the player's language again.
func MenuItem(lang string) item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(lang, "menu.item.name")).WithValue(menuItemKey, true)
}

// RenameMenuItems names the menu items in p's inventory in p's language, as
// p may have come with another client language than the one they got them
// in.
func RenameMenuItems(p *player.Player) {
	inv := p.Inventory()
	name := i18n.T(i18n.Of(p), "menu.item.name")
	for slot, it := range inv.Slots() {
		if IsMenuItem(it) && it.CustomName() != name {
			_ = inv.SetItem(slot, it.WithCustomName(name))
		}
	}
}

func IsMenuItem(s item.Stack) bool {
	_, ok := s.Value(menuItemKey)
	return ok
}
