package skygear

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
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

// A held menu item opens the menu on a block and in the air.
func init() {
	core.UseOnBlock.Handle(20, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
		if screens.IsMenuItem(e.Held) {
			e.Cancel()
			useHeldScreen(e.S, e.P)
			return true
		}
		return false
	})
	core.Use.Handle(20, "screens.menuItem", func(e core.UseEvent) bool {
		if screens.IsMenuItem(e.Held) {
			e.Cancel()
			useHeldScreen(e.S, e.P)
			return true
		}
		return false
	})
}

// useHeldScreen opens the menu held by p. The client repeats use packets
// while the button is held, so repeats are ignored.
func useHeldScreen(s *core.Session, p *player.Player) {
	if s.Debounced(p) {
		return
	}
	if dimension.Hunting(s, p) {
		dimension.ShowRunMenu(s, p)
		return
	}
	if dimension.Watching(s, p) {
		dimension.ShowWatchMenu(s, p)
		return
	}
	screens.OpenMenu(s, p)
}
