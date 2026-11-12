package netstore

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// subscribe registers the steps of netstore on the kernel's hooks.
func subscribe() {
	subscribeFlush()
	subscribeBlocks()
	registerKinds()
}

// Disk contents changed since the controller last flushed them are part of the
// save.
func subscribeFlush() {
	core.Autosave.On(core.Normal, "netstore.flush", func(e core.Tick) {
		Disks(e.S).Flush()
	})
	core.Close.On(core.Normal, "netstore.flush", func(e core.Tick) {
		Disks(e.S).Flush()
	})
}

// A node that was broken drops its network through the block itself (the
// node kinds' Broken, kinds.go). A placed node forgets the setting a node
// left that went without being broken (another block was set in its place)
// and drops the network after machine.place's deferred set: netstore.place
// is in the High tier, after the machine is set up (Normal).
func subscribeBlocks() {
	core.BlockPlace.On(core.High, "netstore.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(machine.Machine); ok && IsNode(b.Kind) {
			Forget(e.S, e.Pos) // a node that went without being broken left its setting
			// machine.Place sets the block in a deferred call; this one runs after it.
			pos := e.Pos
			e.Tx.Defer(func(tx *world.Tx) {
				core.Mark("netstore.placeDeferred")
				Invalidate(tx, pos)
			})
		}
	})
}
