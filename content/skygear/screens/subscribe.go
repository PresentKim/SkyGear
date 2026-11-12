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

// The scoreboards update every second, when the second's work is done: they
// show the result (Monitor).
func subscribeBoards() {
	core.Second.On(core.Monitor, "screens.boards", func(e core.Tick) {
		UpdateBoards(e.S, e.Tx)
	})
}

// The menu item and the scoreboard are given on join. The menu item sets the
// fixed slot of a new player's inventory before any join step hands out
// items, which would otherwise land in that slot: Lowest. The scoreboard
// shows what the other join steps made: Monitor. The scoreboard goes on
// leave, after the features' own leave steps, which may draw it again (island
// expansion mode redraws it when it ends): the last clean-up, Highest.
func subscribePresence() {
	core.Join.On(core.Lowest, "screens.menuItem", func(e core.Presence) {
		if e.First {
			// Bare hands: only the menu item.
			_ = e.P.Inventory().SetItem(8, MenuItem(i18n.Of(e.P)))
		}
		RenameMenuItems(e.P)
	})
	core.Join.On(core.Monitor, "screens.board", func(e core.Presence) {
		UpdateBoard(e.S, e.P)
	})
	core.Leave.On(core.Highest, "screens.board", func(e core.Presence) {
		e.P.RemoveScoreboard()
	})
}

// A held menu item opens the menu on a block and in the air. A hunter's click
// on a block is the second dimension's to refuse (dimension.hunting), whatever
// the hunter holds; in the air the menu item is the hunter's way out. The menu
// item takes its click before anything else looks at it: Lowest.
func subscribeUse() {
	core.UseOnBlock.On(core.Lowest, "screens.menuItem", func(e core.UseOnBlockEvent) {
		if IsMenuItem(e.Held) && !dimension.Hunting(e.S, e.P) {
			e.Cancel()
			useHeldScreen(e.S, e.P)
		}
	})
	core.Use.On(core.Lowest, "screens.menuItem", func(e core.UseEvent) {
		if IsMenuItem(e.Held) {
			e.Cancel()
			useHeldScreen(e.S, e.P)
		}
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
