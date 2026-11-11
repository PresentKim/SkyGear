package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

// Disk contents changed since the controller last flushed them are part of the
// save.
func init() {
	core.Autosave.On(10, "netstore.flush", func(e core.Tick) {
		netstore.Disks(e.S).Flush()
	})
	core.Close.On(30, "netstore.flush", func(e core.Tick) {
		netstore.Disks(e.S).Flush()
	})
}

// A network node breaking drops its network once the block is gone; a placed
// node forgets the setting a node broken by an explosion left and drops the
// network after machine.place's deferred set.
func init() {
	core.BlockBreak.On(70, "netstore.break", func(e core.BlockBreakEvent) {
		if b, ok := e.Block.(machine.Machine); ok && netstore.IsNode(b.Kind) {
			// The block goes after this handler: drop the network once it is gone.
			s, pos := e.S, e.Pos
			e.Tx.Defer(func(tx *world.Tx) {
				core.Mark("netstore.breakDeferred")
				netstore.Invalidate(tx, pos)
				netstore.Forget(s, pos)
			})
		}
	})
	core.BlockPlace.On(80, "netstore.place", func(e core.BlockPlaceEvent) {
		if b, ok := e.Block.(machine.Machine); ok && netstore.IsNode(b.Kind) {
			netstore.Forget(e.S, e.Pos) // a node broken by an explosion left its setting
			// machine.Place sets the block in a deferred call; this one runs after it.
			pos := e.Pos
			e.Tx.Defer(func(tx *world.Tx) {
				core.Mark("netstore.placeDeferred")
				netstore.Invalidate(tx, pos)
			})
		}
	})
}
