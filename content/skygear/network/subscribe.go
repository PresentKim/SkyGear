package network

import (
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/core"
)

// fastCount counts the Fast calls (an item step is every second one,
// conduit.go).
var fastCount = core.NewPart[int](nil)

// FastCount is the number of Fast calls the session has had.
func FastCount(s *core.Session) int { return *fastCount.Of(s) }

// subscribe registers the conduit step on the fast tick.
func subscribe() {
	subscribeFast()
}

// The conduits move on every fast tick.
func subscribeFast() {
	core.Fast.On(core.Normal, "network.step", func(e core.Tick) {
		// Energy moves every FastInterval (four ticks) over the conduits.
		Energy(e.S).Step(e.Tx, 4)
		// Items move every itemnet.StepTicks ticks and fluids every
		// fluidnet.StepTicks ticks (a Fast call is four ticks).
		n := fastCount.Of(e.S)
		*n++
		if *n%(itemnet.StepTicks/4) == 0 {
			core.Mark("network.items")
			Items(e.S).Step(e.Tx)
		}
		if *n%(fluidnet.StepTicks/4) == 0 {
			core.Mark("network.fluids")
			Fluids(e.S).Step(e.Tx)
		}
	})
}
