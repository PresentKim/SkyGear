package sgitem

import "github.com/df-mc/dragonfly/server/item"

// MenuItemKey is the value the menu item carries: the game menu's item is a
// plain item marked with it (screens.MenuItem).
const MenuItemKey = "skygear_menu"

// IsMenuItem reports whether s is the menu item.
func IsMenuItem(s item.Stack) bool {
	_, ok := s.Value(MenuItemKey)
	return ok
}

// OpensScreen reports whether s is an item that opens its screen wherever it
// is used: the menu item or the dimension stone. A step that refuses a click
// because of where the player is leaves the click of such an item alone.
func OpensScreen(s item.Stack) bool {
	if IsMenuItem(s) {
		return true
	}
	_, stone := s.Item().(DimensionStone)
	return stone
}
