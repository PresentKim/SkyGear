package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
)

// A player in island expansion mode stays in place and sees the expansion
// screen again when sneaking.
func init() {
	core.Move.Handle(20, "expand.hold", func(e core.MoveEvent) bool {
		if _, expanding := expand.Of(e.S, e.P); expanding {
			e.Cancel()
			return true
		}
		return false
	})
	core.Sneak.Handle(10, "expand.show", func(e core.PlayerEvent) bool {
		if _, ok := expand.Of(e.S, e.P); ok {
			expand.Show(e.S, e.P)
			return true
		}
		return false
	})
}

// Island expansion mode ends when the player leaves.
func init() {
	core.Leave.On(30, "expand.stop", func(e core.Presence) {
		expand.Stop(e.S, e.P, e.Tx)
	})
}

// No building or clicking in island expansion mode. A click on the map makes
// the server resend the real blocks, so the map chunk is shown again.
func init() {
	core.StartBreak.Handle(10, "expand.guard", func(e core.StartBreakEvent) bool {
		return expandGuard(e.S, e.Ctx, e.Pos)
	})
	core.BlockBreak.Handle(10, "expand.guard", func(e core.BlockBreakEvent) bool {
		return expandGuard(e.S, e.Ctx, e.Pos)
	})
	core.BlockPlace.Handle(10, "expand.guard", func(e core.BlockPlaceEvent) bool {
		return expandGuard(e.S, e.Ctx, e.Pos)
	})
}

// expandGuard cancels the action at pos and reports true for a player in
// island expansion mode.
func expandGuard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
	if _, expanding := expand.Of(s, ctx.Player()); expanding {
		ctx.Cancel()
		expand.RestoreMapNear(s, ctx.Player(), pos)
		return true
	}
	return false
}

// The block clicked and the block next to its face are guarded alike.
func init() {
	core.UseOnBlock.Handle(40, "expand.guard", func(e core.UseOnBlockEvent) bool {
		return expandGuard(e.S, e.Ctx, e.Pos)
	})
	core.UseOnBlock.Handle(160, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
		return expandGuard(e.S, e.Ctx, e.Pos.Side(e.Face))
	})
}
