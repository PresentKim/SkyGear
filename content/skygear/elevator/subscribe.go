package elevator

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the elevator rides on the jump and the sneak.
func subscribe() {
	subscribeJump()
	subscribeSneak()
}

// A jump on an elevator rides it up.
func subscribeJump() {
	core.Jump.On(core.Normal, "elevator.up", func(e core.PlayerEvent) {
		Up(e.S, e.P, e.Tx)
	})
}

// Sneaking on an elevator rides it down. The sneak is not cancelled: the
// player goes on sneaking and the other subscribers of the sneak run. A
// player a takeover holds (island expansion mode, a hunt) does not ride: that
// mode has the player's body.
func subscribeSneak() {
	core.Sneak.On(core.Normal, "elevator.down", func(e core.PlayerEvent) {
		if core.Takeover(e.S, e.P) == "" {
			Down(e.S, e.P, e.Tx)
		}
	})
}
