package conduit

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
)

// subscribe registers the steps of conduit on the kernel's hooks.
func subscribe() {
	subscribeShares()
	subscribeBlocks()
	subscribePlate()
}

// The conduit shares are written on autosave and at close, the visuals removed
// at close; a loaded column puts its bundles into the grids and an unloaded one
// takes them out, before the column is saved.
func subscribeShares() {
	core.Autosave.On(core.Normal, "conduit.shares", func(e core.Tick) {
		WriteShares(e.S, e.Tx)
	})
	core.Close.On(core.Normal, "conduit.close", func(e core.Tick) {
		WriteShares(e.S, e.Tx)
		RemoveVisuals(e.S, e.Tx)
	})
	core.ColumnLoad.On(core.Normal, "conduit.load", func(e core.ColumnEvent) {
		LoadColumn(e.S, e.Tx, e.Blocks)
	})
	core.ColumnUnload.On(core.Normal, "conduit.unload", func(e core.ColumnEvent) {
		UnloadColumn(e.S, e.Tx, e.Blocks)
	})
}

// A conduit bundle breaking loses its visual; a placed one counts (a record:
// Monitor).
func subscribeBlocks() {
	core.BlockBreak.On(core.Normal, "conduit.break", func(e core.BlockBreakEvent) {
		if _, ok := e.Block.(Bundle); ok {
			RemoveVisual(e.S, e.Tx, e.Pos)
		}
	})
	core.BlockPlace.On(core.Monitor, "conduit.place", func(e core.BlockPlaceEvent) {
		if _, ok := e.Block.(Bundle); ok {
			achievement.Count(e.P, "conduit_place", 1)
		}
	})
}

// A player who clicks a conduit's connector plate on the block it touches
// opens its settings, whatever they hold, unless the held conduit goes into
// that conduit (a click on the conduit itself is ConduitBundle.Activate).
// Sneaking places blocks instead. The plate takes the click before the block
// behind it is asked (Activate) and before the held item is used: a tier kit
// or a fairy hourglass on a plate opens the settings.
func subscribePlate() {
	core.UseOnBlock.On(core.Normal, "conduit.plate", func(e core.UseOnBlockEvent) {
		if !e.P.Sneaking() && UsePlate(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held) {
			e.Cancel()
		}
	})
}
