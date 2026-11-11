package skygear

import (
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
)

// fastCount counts the Fast calls (an item step is every second one,
// conduit.go).
var fastCount = core.NewPart[int](nil)

// The conduits move on every fast tick.
func init() {
	core.Fast.On(10, "network.step", func(e core.Tick) {
		// Energy moves every FastInterval (four ticks) over the conduits.
		network.Energy(e.S).Step(e.Tx, 4)
		// Items move every itemnet.StepTicks ticks and fluids every
		// fluidnet.StepTicks ticks (a Fast call is four ticks).
		n := fastCount.Of(e.S)
		*n++
		if *n%(itemnet.StepTicks/4) == 0 {
			core.Mark("network.items")
			network.Items(e.S).Step(e.Tx)
		}
		if *n%(fluidnet.StepTicks/4) == 0 {
			core.Mark("network.fluids")
			network.Fluids(e.S).Step(e.Tx)
		}
	})
}
