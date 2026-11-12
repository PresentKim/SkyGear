package island

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
)

// subscribe registers the steps of island on the kernel's hooks.
func subscribe() {
	subscribeVoid()
	subscribeLost()
	subscribeGuard()
	subscribeSide()
}

// A player that falls below the island returns from the void: the last step
// that may cancel the move, Highest. The cancelled move is no move for the
// subscribers after it (the walk counter is not called).
func subscribeVoid() {
	core.Move.On(core.Highest, "island.void", func(e core.MoveEvent) {
		if e.Pos.Y() < VoidY {
			e.Cancel()
			ReturnFromVoid(e.S, e.P, e.Tx)
		}
	})
}

// Items lost to the void are collected: on join and on a new day the lost
// items are given back; when the session starts the island's chunks are kept
// loaded; Dragonfly closes an item entity below the world's lowest height,
// while a picked-up or expired item despawns above it.
func subscribeLost() {
	core.Join.On(core.Normal, "island.purgeLost", func(e core.Presence) {
		PurgeLost(e.S)
	})
	core.NewDay.On(core.Normal, "island.purgeLost", func(e core.DayEvent) {
		PurgeLost(e.S)
	})
	core.Start.On(core.Normal, "island.keepLoaded", func(e core.Tick) {
		for _, c := range StateOf(e.S).Chunks {
			KeepLoaded(e.S, e.Tx, c)
		}
	})
	core.EntityDespawn.On(core.Normal, "island.lost", func(e core.DespawnEvent) {
		ent, ok := e.Entity.(*entity.Ent)
		if !ok || e.Entity.Position().Y() >= float64(e.Tx.Range()[0]) {
			return
		}
		if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
			AddLost(e.S, ib.Item())
		}
	})
}

// Building is only allowed on the island's own ground. The guard refuses after
// the modes that hold a player (Lowest) and before any step works on the
// block: Low.
func subscribeGuard() {
	core.StartBreak.On(core.Low, "island.guard", func(e core.StartBreakEvent) {
		guard(e.Actor, e.Pos)
	})
	core.BlockBreak.On(core.Low, "island.guard", func(e core.BlockBreakEvent) {
		guard(e.Actor, e.Pos)
	})
	core.BlockPlace.On(core.Low, "island.guard", func(e core.BlockPlaceEvent) {
		guard(e.Actor, e.Pos)
	})
}

// guard cancels the action at pos when pos is not the island's to build on,
// and tells the player so.
func guard(a core.Actor, pos cube.Pos) {
	if CanBuild(a.S, pos) {
		return
	}
	a.Cancel()
	if core.InArena(pos.Vec3()) {
		return // the second dimension is no one's to build on
	}
	a.P.SendTip(i18n.T(i18n.Default, "island.outside"))
}

// The block clicked and the block next to its face are guarded alike. The side
// is refused last, when no subscriber cancelled the click: Highest.
func subscribeSide() {
	core.UseOnBlock.On(core.Low, "island.guard", func(e core.UseOnBlockEvent) {
		guard(e.Actor, e.Pos)
	})
	core.UseOnBlock.On(core.Highest, "island.guardSide", func(e core.UseOnBlockEvent) {
		guard(e.Actor, e.Pos.Side(e.Face))
	})
}
