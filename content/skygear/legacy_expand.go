package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
)

// A player in island expansion mode stays in place and sees the expansion
// screen again when sneaking.
func init() {
	core.Move.Handle(20, "expand.hold", func(e core.MoveEvent) bool {
		if _, expanding := expand.Of(e.S, e.P); expanding {
			e.Cancel()
			return true
		}
		return false
	})
	core.Sneak.Handle(10, "expand.show", func(e core.PlayerEvent) bool {
		if _, ok := expand.Of(e.S, e.P); ok {
			expand.Show(e.S, e.P)
			return true
		}
		return false
	})
}

// Island expansion mode ends when the player leaves.
func init() {
	core.Leave.On(30, "expand.stop", func(e core.Presence) {
		expand.Stop(e.S, e.P, e.Tx)
	})
}
