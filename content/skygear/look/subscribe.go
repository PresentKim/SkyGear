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

// A movement updates where the player looks.
func subscribeMove() {
	core.Move.On(10, "look.update", func(e core.MoveEvent) {
		Update(e.S, e.P, e.Tx, e.Pos, e.Rot, false)
	})
}

// Every player's panel updates on every fast tick.
func subscribeFast() {
	core.Fast.On(30, "look.updateAll", func(e core.Tick) {
		UpdateAll(e.S, e.Tx)
	})
}

// What the player looked at is cleared when the player leaves.
func subscribeLeave() {
	core.Leave.On(40, "look.clear", func(e core.Presence) {
		waila.Clear(e.P)
	})
}
