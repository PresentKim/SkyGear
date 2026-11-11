package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
)

// A movement updates where the player looks.
func init() {
	core.Move.On(10, "look.update", func(e core.MoveEvent) {
		look.Update(e.S, e.P, e.Tx, e.Pos, e.Rot, false)
	})
}

// Every player's panel updates on every fast tick.
func init() {
	core.Fast.On(30, "look.updateAll", func(e core.Tick) {
		look.UpdateAll(e.S, e.Tx)
	})
}
