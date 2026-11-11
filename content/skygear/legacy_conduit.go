package skygear

import (
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
