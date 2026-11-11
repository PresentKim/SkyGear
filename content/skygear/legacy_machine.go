package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The monitor reads its machines every second.
func init() {
	core.Second.On(80, "machine.monitor", func(e core.Tick) {
		machine.TickMonitor(e.S, e.Tx)
	})
}

// A machine breaking or placed is taken down or set up.
func init() {
	core.BlockBreak.On(80, "machine.break", func(e core.BlockBreakEvent) {
		if b, ok := e.Block.(machine.Machine); ok {
			machine.Break(e.S, e.Tx, e.Pos, b)
		}
	})
	core.BlockPlace.On(70, "machine.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(machine.Machine); ok {
			machine.Place(e.S, e.P, e.Tx, e.Pos, b.Kind)
		}
	})
}

// A net-shaped machine or a miner in hand remembers the face it is placed on.
func init() {
	core.UseOnBlock.On(60, "machine.noteFace", func(e core.UseOnBlockEvent) {
		if m, ok := e.Held.Item().(machine.Machine); ok && (machine.NetShaped(m.Kind) || m.Kind == machine.KindMiner) {
			machine.NotePlaceFace(e.S, e.P, e.Face)
		}
	})
}
