package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// handler handles a player's events inside a dragonfactory instance.
type handler struct {
	player.NopHandler
	s *session
}

func (h *handler) deny(ctx *player.Context, pos cube.Pos) bool {
	if _, expanding := h.s.expansionOf(ctx.Player()); expanding {
		// No building or clicking in island expansion mode. A click on the
		// fake glass floor makes the server resend the real blocks, so the
		// floor is shown again.
		ctx.Cancel()
		h.s.restoreFloorNear(ctx.Player(), pos)
		return true
	}
	if h.s.canBuild(pos) {
		return false
	}
	ctx.Cancel()
	ctx.Player().SendTip("§c섬 밖입니다. /menu 의 섬 확장으로 청크를 구매하세요.")
	return true
}

func (h *handler) HandleStartBreak(ctx *player.Context, pos cube.Pos) { h.deny(ctx, pos) }
func (h *handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, _ *int) {
	if h.deny(ctx, pos) {
		return
	}
	if held, _ := ctx.Player().HeldItems(); isHammer(held) {
		if into, ok := crushed(ctx.Player().Tx().Block(pos)); ok {
			*drops = []item.Stack{item.NewStack(into, 1)}
		}
	}
	switch b := ctx.Player().Tx().Block(pos).(type) {
	case Drawer:
		// Hitting the front takes items out; it never mines the drawer.
		if lp, ok := h.s.lastPunch[ctx.Player().H()]; ok && lp.pos == pos && lp.front {
			ctx.Cancel()
			return
		}
		h.s.removeDisplay(ctx.Player().Tx(), pos)
	case Machine:
		h.s.breakMachine(ctx.Player().Tx(), pos, b)
	}
}
func (h *handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if h.deny(ctx, pos) {
		return
	}
	if kind, _, ok := cropOf(b); ok {
		h.s.plantCrop(pos, kind)
	}
	held, _ := ctx.Player().HeldItems()
	switch b := b.(type) {
	case Machine:
		h.s.placeMachine(ctx.Player(), ctx.Player().Tx(), pos, b.Kind)
	case Core:
		h.s.placeCoreAchievement(ctx.Player(), b)
	case Drawer:
		h.s.placeDrawer(ctx.Player(), ctx.Player().Tx(), pos, held)
	}
}
func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, _ mgl64.Vec3) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) || isCheque(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	if h.deny(ctx, pos) {
		return
	}
	// Using a crop shows its time or harvests it, and places nothing;
	// sneaking uses it as usual (placing blocks, bone meal).
	if !ctx.Player().Sneaking() && h.s.useCrop(ctx.Player(), ctx.Player().Tx(), pos) {
		ctx.Cancel()
		return
	}
	h.deny(ctx, pos.Side(face))
}

// HandleItemUseOnEntity and HandleAttackEntity take the gift of a visiting
// fairy, whichever way the player touches it.
func (h *handler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	h.touchFairy(ctx, e)
}

func (h *handler) HandleAttackEntity(ctx *player.Context, e world.Entity, _, _ *float64, _ *bool) {
	h.touchFairy(ctx, e)
}

func (h *handler) touchFairy(ctx *player.Context, e world.Entity) {
	if f, ok := h.s.fairyOf(e); ok {
		ctx.Cancel()
		f.gift(ctx.Player().Tx(), h.s, ctx.Player())
	}
}

func (h *handler) HandleItemUse(ctx *player.Context) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) || isCheque(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
	}
}

// useHeldScreen opens the menu or redeems a cheque held by p. The client
// repeats use packets while the button is held, so repeats are ignored.
func (h *handler) useHeldScreen(p *player.Player, held item.Stack) {
	if h.s.debounced(p) {
		return
	}
	if isMenuItem(held) {
		h.s.openMenu(p)
	} else {
		h.s.redeemCheque(p)
	}
}

// HandleMove returns players that fall below the island into the void, and
// keeps players in island expansion mode in place.
func (h *handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, _ cube.Rotation) {
	if _, expanding := h.s.expansionOf(ctx.Player()); expanding {
		ctx.Cancel()
		return
	}
	if newPos.Y() < VoidY {
		ctx.Cancel()
		h.s.returnFromVoid(ctx.Player(), ctx.Player().Tx())
	}
}

// HandleToggleSneak opens the island expansion actions when sneaking in
// expansion mode.
func (h *handler) HandleToggleSneak(ctx *player.Context, after bool) {
	if _, ok := h.s.expansionOf(ctx.Player()); ok && after {
		h.s.showExpandActions(ctx.Player())
	}
}
