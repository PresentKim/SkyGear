package skygear

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
)

// A player that falls below the island returns from the void.
func init() {
	core.Move.On(40, "island.void", func(e core.MoveEvent) {
		if e.Pos.Y() < island.VoidY {
			e.Cancel()
			island.ReturnFromVoid(e.S, e.P, e.Tx)
		}
	})
}

// Items lost to the void are collected: on join and on a new day the lost
// items are given back; when the session starts the island's chunks are kept
// loaded; Dragonfly closes an item entity below the world's lowest height,
// while a picked-up or expired item despawns above it.
func init() {
	core.Join.On(50, "island.purgeLost", func(e core.Presence) {
		island.PurgeLost(e.S)
	})
	core.NewDay.On(10, "island.purgeLost", func(e core.DayEvent) {
		island.PurgeLost(e.S)
	})
	core.Start.On(10, "island.keepLoaded", func(e core.Tick) {
		for _, c := range island.StateOf(e.S).Chunks {
			island.KeepLoaded(e.S, e.Tx, c)
		}
	})
	core.EntityDespawn.On(10, "island.lost", func(e core.DespawnEvent) {
		ent, ok := e.Entity.(*entity.Ent)
		if !ok || e.Entity.Position().Y() >= float64(e.Tx.Range()[0]) {
			return
		}
		if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
			island.AddLost(e.S, ib.Item())
		}
	})
}
