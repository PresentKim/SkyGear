package island

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
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
// subscribers after it (the walk counter).
func subscribeVoid() {
	core.Move.Handle(core.Highest, "island.void", func(e core.MoveEvent) bool {
		if e.Pos.Y() < VoidY {
			e.Cancel()
			ReturnFromVoid(e.S, e.P, e.Tx)
			return true
		}
		return false
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
	core.StartBreak.Handle(core.Low, "island.guard", func(e core.StartBreakEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.BlockBreak.Handle(core.Low, "island.guard", func(e core.BlockBreakEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.BlockPlace.Handle(core.Low, "island.guard", func(e core.BlockPlaceEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
}

// guard reports whether the action at pos is refused because pos is not the
// island's to build on; it cancels the action then.
func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
	if CanBuild(s, pos) {
		return false
	}
	ctx.Cancel()
	if core.InArena(pos.Vec3()) {
		return true // the hunting ground is no one's to build on
	}
	ctx.Player().SendTip(i18n.T(i18n.Default, "island.outside"))
	return true
}

// The block clicked and the block next to its face are guarded alike. The side
// is refused last, when no step took the click: Highest.
func subscribeSide() {
	core.UseOnBlock.Handle(core.Low, "island.guard", func(e core.UseOnBlockEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.UseOnBlock.Handle(core.Highest, "island.guardSide", func(e core.UseOnBlockEvent) bool {
		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
	})
}
