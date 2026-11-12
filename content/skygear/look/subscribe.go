package look

import (
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/core"
)

// subscribe registers the look panel on the move, the fast tick and the leave.
func subscribe() {
	subscribeMove()
	subscribeFast()
	subscribeLeave()
}

// A movement updates where the player looks. It is called for every move,
// also a cancelled one (island expansion mode holds the player in place, the
// island brings a player back from the void): Update then clears the hidden
// panel.
func subscribeMove() {
	core.Move.On(core.Monitor, "look.update", func(e core.MoveEvent) {
		Update(e.S, e.P, e.Tx, e.Pos, e.Rot, false)
	}, core.HandleCancelled)
}

// Every player's panel updates on every fast tick, when the tick's work is
// done: it shows the result (Monitor).
func subscribeFast() {
	core.Fast.On(core.Monitor, "look.updateAll", func(e core.Tick) {
		UpdateAll(e.S, e.Tx)
	})
}

// What the player looked at is cleared when the player leaves, after the
// features' own leave steps: the last clean-up, Highest.
func subscribeLeave() {
	core.Leave.On(core.Highest, "look.clear", func(e core.Presence) {
		waila.Clear(e.P)
	})
}
