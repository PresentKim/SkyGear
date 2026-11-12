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

// Sneaking on an elevator rides it down. A ride cancels the event for the
// later tiers: the sneak was used. A player a takeover holds (island
// expansion mode, a hunt) does not ride: that mode has the player's body.
func subscribeSneak() {
	core.Sneak.Handle(core.Normal, "elevator.down", func(e core.PlayerEvent) bool {
		return core.Takeover(e.S, e.P) == "" && Down(e.S, e.P, e.Tx)
	})
}
