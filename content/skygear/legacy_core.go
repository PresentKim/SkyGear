package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
)

// Shapes are sent when a player joins and cleared, client-side, before the
// player leaves the instance world; the core session drops the player's
// per-player parts.
func init() {
	core.Join.On(40, "core.shapes", func(e core.Presence) {
		e.S.SendShapes(e.P)
	})
	// The kernel broadcasts the new day right after this hook; the step only
	// names that broadcast in the trace.
	core.NewDay.On(20, "game.newDay", func(core.DayEvent) {})
	core.Leave.On(20, "core.parts", func(e core.Presence) {
		e.S.Leave(e.P)
	})
	core.Leave.On(50, "core.shapes", func(e core.Presence) {
		e.P.RemoveAllDebugShapes()
	})
}
