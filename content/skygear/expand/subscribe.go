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
// screen again when sneaking. The sneak stays a sneak for everyone else: it
// is not cancelled, and the subscribers that have no business with a player
// in a mode say so themselves (core.Takeover).
func subscribeHold() {
	core.Move.On(core.Lowest, "expand.hold", func(e core.MoveEvent) {
		if _, expanding := Of(e.S, e.P); expanding {
			e.Cancel()
		}
	})
	core.Sneak.On(core.Normal, "expand.show", func(e core.PlayerEvent) {
		if _, ok := Of(e.S, e.P); ok {
			Show(e.S, e.P)
		}
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
	core.StartBreak.On(core.Lowest, "expand.guard", func(e core.StartBreakEvent) {
		guard(e.Actor, e.Pos)
	})
	core.BlockBreak.On(core.Lowest, "expand.guard", func(e core.BlockBreakEvent) {
		guard(e.Actor, e.Pos)
	})
	core.BlockPlace.On(core.Lowest, "expand.guard", func(e core.BlockPlaceEvent) {
		guard(e.Actor, e.Pos)
	})
}

// guard cancels the action at pos for a player in island expansion mode and
// shows the map chunk there again.
func guard(a core.Actor, pos cube.Pos) {
	if _, expanding := Of(a.S, a.P); expanding {
		a.Cancel()
		RestoreMapNear(a.S, a.P, pos)
	}
}

// The block clicked and the block next to its face are guarded alike. An
// item that opens its screen wherever it is used (the menu item, the dimension
// stone) keeps its click: its own step takes it. The side is refused last,
// when no subscriber cancelled the click: Highest.
func subscribeSide() {
	core.UseOnBlock.On(core.Lowest, "expand.guard", func(e core.UseOnBlockEvent) {
		if !sgitem.OpensScreen(e.Held) {
			guard(e.Actor, e.Pos)
		}
	})
	core.UseOnBlock.On(core.Highest, "expand.guardSide", func(e core.UseOnBlockEvent) {
		guard(e.Actor, e.Pos.Side(e.Face))
	})
}

// A player in island expansion mode sees no WAILA panel.
func hideLook() {
	look.Hidden = func(s *core.Session, p *player.Player) bool {
		_, expanding := Of(s, p)
		return expanding
	}
}
