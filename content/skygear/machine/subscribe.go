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

// A machine breaking or placed is taken down or set up.
func subscribeBlocks() {
	core.BlockBreak.On(core.Normal, "machine.break", func(e core.BlockBreakEvent) {
		if b, ok := e.Block.(Machine); ok {
			Break(e.S, e.Tx, e.Pos, b)
		}
	})
	core.BlockPlace.On(core.Normal, "machine.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(Machine); ok {
			Place(e.S, e.P, e.Tx, e.Pos, b)
		}
	})
}
