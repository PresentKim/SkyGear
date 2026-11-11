package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The drawer displays are removed when the save closes.
func init() {
	core.Close.On(10, "storage.displays", func(e core.Tick) {
		storage.RemoveDisplays(e.S, e.Tx)
	})
}

// Hitting a drawer's front takes items out and never mines it; another drawer
// loses its display. A set machine breaking or placed invalidates the
// networks; a placed drawer is set up.
func init() {
	core.BlockBreak.On(60, "storage.break", func(e core.BlockBreakEvent) {
		switch b := e.Block.(type) {
		case storage.Drawer:
			if lp, ok := storage.LastPunch.Of(e.S)[e.P.H()]; ok && lp.Pos == e.Pos && lp.Front {
				e.Cancel()
				return
			}
			storage.RemoveDisplay(e.S, e.Tx, e.Pos)
		case machine.Machine:
			if storage.IsSetMachine(b.Kind) {
				storage.InvalidateNetworks(e.S)
			}
		}
	})
	core.BlockPlace.On(60, "storage.place", func(e core.BlockPlaceEvent) {
		switch b := e.Block.(type) {
		case machine.Machine:
			if storage.IsSetMachine(b.Kind) {
				storage.InvalidateNetworks(e.S)
			}
		case storage.Drawer:
			storage.PlaceDrawer(e.S, e.P, e.Tx, e.Pos, e.Held)
		}
	})
}
