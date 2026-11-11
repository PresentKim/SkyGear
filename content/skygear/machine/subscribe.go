package machine

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the steps of machine on the kernel's hooks.
func subscribe() {
	subscribeMonitor()
	subscribeBlocks()
	subscribeFace()
}

// The monitor reads its machines every second.
func subscribeMonitor() {
	core.Second.On(80, "machine.monitor", func(e core.Tick) {
		TickMonitor(e.S, e.Tx)
	})
}

// A machine breaking or placed is taken down or set up.
func subscribeBlocks() {
	core.BlockBreak.On(80, "machine.break", func(e core.BlockBreakEvent) {
		if b, ok := e.Block.(Machine); ok {
			Break(e.S, e.Tx, e.Pos, b)
		}
	})
	core.BlockPlace.On(70, "machine.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(Machine); ok {
			Place(e.S, e.P, e.Tx, e.Pos, b.Kind)
		}
	})
}

// A net-shaped machine or a miner in hand remembers the face it is placed on.
func subscribeFace() {
	core.UseOnBlock.On(60, "machine.noteFace", func(e core.UseOnBlockEvent) {
		if m, ok := e.Held.Item().(Machine); ok && (NetShaped(m.Kind) || m.Kind == KindMiner) {
			NotePlaceFace(e.S, e.P, e.Face)
		}
	})
}
