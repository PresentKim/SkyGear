package expand

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the takeover and the steps of expand on the kernel.
func subscribe() {
	core.RegisterTakeover("expand", func(s *core.Session, p *player.Player) bool {
		_, ok := Of(s, p)
		return ok
	})
	subscribeHold()
	subscribeLeave()
	subscribeGuard()
	subscribeSide()
	hideLook()
}

// A player in island expansion mode stays in place and sees the expansion
// screen again when sneaking.
func subscribeHold() {
	core.Move.Handle(core.Lowest, "expand.hold", func(e core.MoveEvent) bool {
		if _, expanding := Of(e.S, e.P); expanding {
			e.Cancel()
			return true
		}
		return false
	})
	core.Sneak.Handle(core.Lowest, "expand.show", func(e core.PlayerEvent) bool {
		if _, ok := Of(e.S, e.P); ok {
			Show(e.S, e.P)
			return true
		}
		return false
	})
}

// Island expansion mode ends when the player leaves. The player's entries are
// dropped when every Leave subscriber has run (core.Lifecycle.Leave), so Stop
// finds the expansion there and restores the saved game mode, the movement
// and the camera.
func subscribeLeave() {
	core.Leave.On(core.Normal, "expand.stop", func(e core.Presence) {
		Stop(e.S, e.P, e.Tx)
	})
}

// No building or clicking in island expansion mode. A click on the map makes
// the server resend the real blocks, so the map chunk is shown again. The mode
// refuses before anything else looks at the event: Lowest.
func subscribeGuard() {
	core.StartBreak.Handle(core.Lowest, "expand.guard", func(e core.StartBreakEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.BlockBreak.Handle(core.Lowest, "expand.guard", func(e core.BlockBreakEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.BlockPlace.Handle(core.Lowest, "expand.guard", func(e core.BlockPlaceEvent) bool {
		return guard(e.S, e.Ctx, e.Pos)
	})
}

// guard cancels the action at pos and reports true for a player in island
// expansion mode.
func guard(s *core.Session, ctx *player.Context, pos cube.Pos) bool {
	if _, expanding := Of(s, ctx.Player()); expanding {
		ctx.Cancel()
		RestoreMapNear(s, ctx.Player(), pos)
		return true
	}
	return false
}

// The block clicked and the block next to its face are guarded alike. An
// item that opens its screen wherever it is used (the menu item, the dimension
// stone) keeps its click: its own step takes it. The side is refused last,
// when no step took the click: Highest.
func subscribeSide() {
	core.UseOnBlock.Handle(core.Lowest, "expand.guard", func(e core.UseOnBlockEvent) bool {
		if sgitem.OpensScreen(e.Held) {
			return false
		}
		return guard(e.S, e.Ctx, e.Pos)
	})
	core.UseOnBlock.Handle(core.Highest, "expand.guardSide", func(e core.UseOnBlockEvent) bool {
		return guard(e.S, e.Ctx, e.Pos.Side(e.Face))
	})
}

// A player in island expansion mode sees no WAILA panel.
func hideLook() {
	look.Hidden = func(s *core.Session, p *player.Player) bool {
		_, expanding := Of(s, p)
		return expanding
	}
}
