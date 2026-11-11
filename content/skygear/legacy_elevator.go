package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/elevator"
)

// A jump on an elevator rides it up.
func init() {
	core.Jump.On(20, "elevator.up", func(e core.PlayerEvent) {
		elevator.Up(e.S, e.P, e.Tx)
	})
}

// Sneaking on an elevator rides it down.
func init() {
	core.Sneak.Handle(30, "elevator.down", func(e core.PlayerEvent) bool {
		return elevator.Down(e.S, e.P, e.Tx)
	})
}
