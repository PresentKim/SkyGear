package machine

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the steps of machine on the kernel's hooks.
func subscribe() {
	subscribeMonitor()
	subscribeBlocks()
	registerLook()
}

// The monitor reads its machines every second.
func subscribeMonitor() {
	core.Second.On(core.Normal, "machine.monitor", func(e core.Tick) {
		TickMonitor(e.S, e.Tx)
	})
}

// A placed machine is set up. A broken one takes itself down: its BreakInfo
// has the handler (Break), which Dragonfly calls for every break.
func subscribeBlocks() {
	core.BlockPlace.On(core.Normal, "machine.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(Machine); ok {
			Place(e.S, e.P, e.Tx, e.Pos, b)
		}
	})
}
