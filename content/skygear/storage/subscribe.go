package storage

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// subscribe registers the steps of storage on the kernel's hooks.
func subscribe() {
	subscribeDisplays()
	subscribeBlocks()
	subscribeSneakUse()
	registerKinds()
}

// The drawer displays are removed when the save closes.
func subscribeDisplays() {
	core.Close.On(core.Normal, "storage.displays", func(e core.Tick) {
		RemoveDisplays(e.S, e.Tx)
	})
}

// Hitting a drawer's front takes items out and never mines it: the break
// that ends such a hit is refused before any step works on the block (Low),
// where the island allows building (elsewhere the island's guard refuses it).
// A drawer that does break loses its display and a broken set machine
// invalidates the networks through the block itself (Drawer.BreakInfo, the
// set kinds' Broken). A placed set machine invalidates the networks; a
// placed drawer is set up.
func subscribeBlocks() {
	core.BlockBreak.On(core.Low, "storage.front", func(e core.BlockBreakEvent) {
		if _, ok := e.Block.(Drawer); !ok {
			return
		}
		if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front && island.CanBuild(e.S, e.Pos) {
			e.Cancel()
		}
	})
	core.BlockPlace.On(core.Normal, "storage.place", func(e core.BlockPlaceEvent) {
		switch b := e.Block.(type) {
		case machine.Machine:
			if IsSetMachine(b.Kind) {
				InvalidateNetworks(e.S)
			}
		case Drawer:
			PlaceDrawer(e.S, e.P, e.Tx, e.Pos, e.Held)
		}
	})
}

// A sneaking player uses a drawer with a held item that is no block.
func subscribeSneakUse() {
	core.UseOnBlock.On(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) {
		if SneakUse(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
		}
	})
}
