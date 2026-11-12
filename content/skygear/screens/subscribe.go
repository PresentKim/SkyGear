package screens

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
)

// subscribe registers the steps of screens on the kernel's hooks.
func subscribe() {
	subscribeBoards()
	subscribePresence()
	subscribeUse()
}

// The scoreboards update every second.
func subscribeBoards() {
	core.Second.On(100, "screens.boards", func(e core.Tick) {
		UpdateBoards(e.S, e.Tx)
	})
}

// The menu item and the scoreboard are given on join; the scoreboard goes on
// leave.
func subscribePresence() {
	core.Join.On(10, "screens.menuItem", func(e core.Presence) {
		if e.First {
			// Bare hands: only the menu item.
			_ = e.P.Inventory().SetItem(8, MenuItem(i18n.Of(e.P)))
		}
		RenameMenuItems(e.P)
	})
	core.Join.On(70, "screens.board", func(e core.Presence) {
		UpdateBoard(e.S, e.P)
	})
	core.Leave.On(60, "screens.board", func(e core.Presence) {
		e.P.RemoveScoreboard()
	})
}

// A held menu item opens the menu on a block and in the air. A hunter's click
// on a block is the hunting ground's to refuse (dimension.hunting), whatever
// the hunter holds; in the air the menu item is the hunter's way out.
func subscribeUse() {
	core.UseOnBlock.Handle(20, "screens.menuItem", func(e core.UseOnBlockEvent) bool {
		if IsMenuItem(e.Held) && !dimension.Hunting(e.S, e.P) {
			e.Cancel()
			useHeldScreen(e.S, e.P)
			return true
		}
		return false
	})
	core.Use.Handle(20, "screens.menuItem", func(e core.UseEvent) bool {
		if IsMenuItem(e.Held) {
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
	OpenMenu(s, p)
}
