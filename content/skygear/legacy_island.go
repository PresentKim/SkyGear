package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
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

// Building is only allowed on the island's own ground.
func init() {
	core.StartBreak.Handle(20, "island.guard", func(e core.StartBreakEvent) bool {
		return islandGuard(e.S, e.Ctx, e.Pos)
	})
	core.BlockBreak.Handle(20, "island.guard", func(e core.BlockBreakEvent) bool {
		return islandGuard(e.S, e.Ctx, e.Pos)
	})
	core.BlockPlace.Handle(20, "island.guard", func(e core.BlockPlaceEvent) bool {
		return islandGuard(e.S, e.Ctx, e.Pos)
	})
}

// islandGuard reports whether the action at pos is refused because pos is not
// the island's to build on; it cancels the action then.
func islandGuard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
	if island.CanBuild(s, pos) {
		return false
	}
	ctx.Cancel()
	if core.InArena(pos.Vec3()) {
		return true // the hunting ground is no one's to build on
	}
	ctx.Player().SendTip(i18n.T(i18n.Default, "island.outside"))
	return true
}
