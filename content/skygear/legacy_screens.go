package skygear

import (
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/screens"
)

// The scoreboards update every second.
func init() {
	core.Second.On(100, "screens.boards", func(e core.Tick) {
		screens.UpdateBoards(e.S, e.Tx)
	})
}

// The menu item and the scoreboard are given on join; the scoreboard goes on
// leave.
func init() {
	core.Join.On(10, "screens.menuItem", func(e core.Presence) {
		if e.First {
			// Bare hands: only the menu item.
			_ = e.P.Inventory().SetItem(8, screens.MenuItem(i18n.Of(e.P)))
		}
		screens.RenameMenuItems(e.P)
	})
	core.Join.On(70, "screens.board", func(e core.Presence) {
		screens.UpdateBoard(e.S, e.P)
	})
	core.Leave.On(60, "screens.board", func(e core.Presence) {
		e.P.RemoveScoreboard()
	})
}
