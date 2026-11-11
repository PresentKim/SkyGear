package elevator

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the elevator rides on the jump and the sneak.
func subscribe() {
	subscribeJump()
	subscribeSneak()
}

// A jump on an elevator rides it up.
func subscribeJump() {
	core.Jump.On(20, "elevator.up", func(e core.PlayerEvent) {
		Up(e.S, e.P, e.Tx)
	})
}

// Sneaking on an elevator rides it down.
func subscribeSneak() {
	core.Sneak.Handle(30, "elevator.down", func(e core.PlayerEvent) bool {
		return Down(e.S, e.P, e.Tx)
	})
}
