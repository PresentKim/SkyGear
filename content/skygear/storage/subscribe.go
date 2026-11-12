package storage

import (
	"github.com/presentkim/skygear/content/skygear/core"
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

// Hitting a drawer's front takes items out and never mines it; another drawer
// loses its display. A set machine breaking or placed invalidates the
// networks; a placed drawer is set up.
func subscribeBlocks() {
	core.BlockBreak.On(core.Normal, "storage.break", func(e core.BlockBreakEvent) {
		switch b := e.Block.(type) {
		case Drawer:
			if lp, ok := LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front {
				e.Cancel()
				return
			}
			RemoveDisplay(e.S, e.Tx, e.Pos)
		case machine.Machine:
			if IsSetMachine(b.Kind) {
				InvalidateNetworks(e.S)
			}
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

// A sneaking player uses a drawer with a held item that is no block. It is
// registered before handwork.stick, in the same tier, and takes the click
// first (a stick opens the drawer's menu and counts no stick use): that order
// is one of the exceptions the tier order guard keeps (the root's orderKept).
func subscribeSneakUse() {
	core.UseOnBlock.Handle(core.Normal, "storage.sneakUse", func(e core.UseOnBlockEvent) bool {
		if SneakUse(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
}
