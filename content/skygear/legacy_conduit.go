package skygear

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The conduit shares are written on autosave and at close, the visuals removed
// at close; a loaded column puts its bundles into the grids and an unloaded one
// takes them out, before the column is saved.
func init() {
	core.Autosave.On(20, "conduit.shares", func(e core.Tick) {
		conduit.WriteShares(e.S, e.Tx)
	})
	core.Close.On(20, "conduit.close", func(e core.Tick) {
		conduit.WriteShares(e.S, e.Tx)
		conduit.RemoveVisuals(e.S, e.Tx)
	})
	core.ColumnLoad.On(10, "conduit.load", func(e core.ColumnEvent) {
		conduit.LoadColumn(e.S, e.Tx, e.Blocks)
	})
	core.ColumnUnload.On(10, "conduit.unload", func(e core.ColumnEvent) {
		conduit.UnloadColumn(e.S, e.Tx, e.Blocks)
	})
}

// A conduit bundle breaking loses its visual; a placed one counts.
func init() {
	core.BlockBreak.On(90, "conduit.break", func(e core.BlockBreakEvent) {
		if _, ok := e.Block.(conduit.Bundle); ok {
			conduit.RemoveVisual(e.S, e.Tx, e.Pos)
		}
	})
	core.BlockPlace.On(90, "conduit.place", func(e core.BlockPlaceEvent) {
		if _, ok := e.Block.(conduit.Bundle); ok {
			achievement.Count(e.P, "conduit_place", 1)
		}
	})
}

// A player who clicks a conduit's connector plate on the block it touches
// opens its settings, whatever they hold, unless the held conduit goes into
// that conduit (a click on the conduit itself is ConduitBundle.Activate).
// Sneaking places blocks instead.
func init() {
	core.UseOnBlock.Handle(100, "conduit.plate", func(e core.UseOnBlockEvent) bool {
		if !e.P.Sneaking() && conduit.UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
}
