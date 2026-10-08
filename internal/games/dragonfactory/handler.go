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
	if h.s.canBuild(pos) {
		return false
	}
	ctx.Cancel()
	ctx.Player().SendTip("§c섬 밖입니다. /menu 의 섬 확장으로 청크를 구매하세요.")
	return true
}

func (h *handler) HandleStartBreak(ctx *player.Context, pos cube.Pos) { h.deny(ctx, pos) }
func (h *handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, _ *[]item.Stack, _ *int) {
	if h.deny(ctx, pos) {
		return
	}
	switch b := ctx.Player().Tx().Block(pos).(type) {
	case Core:
		h.s.breakCore(b, pos)
	case Drawer:
		// Hitting the front takes items out; it never mines the drawer.
		if lp := h.s.lastPunch; lp.pos == pos && lp.player == ctx.Player().H() && lp.front {
			ctx.Cancel()
			return
		}
		h.s.removeDisplay(ctx.Player().Tx(), pos)
	}
}
func (h *handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if h.deny(ctx, pos) {
		return
	}
	held, _ := ctx.Player().HeldItems()
	switch b.(type) {
	case Core:
		if !h.s.placeCore(ctx.Player(), ctx.Player().Tx(), pos, held) {
			ctx.Cancel()
		}
	case Drawer:
		h.s.placeDrawer(ctx.Player(), ctx.Player().Tx(), pos, held)
	}
}
func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, _ mgl64.Vec3) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) {
		ctx.Cancel()
		h.s.showMenu(ctx.Player())
		return
	}
	if isCheque(held) {
		ctx.Cancel()
		h.s.redeemCheque(ctx.Player())
		return
	}
	if !h.deny(ctx, pos) {
		h.deny(ctx, pos.Side(face))
	}
}
func (h *handler) HandleItemUse(ctx *player.Context) {
	held, _ := ctx.Player().HeldItems()
	switch {
	case isMenuItem(held):
		ctx.Cancel()
		h.s.showMenu(ctx.Player())
	case isCheque(held):
		ctx.Cancel()
		h.s.redeemCheque(ctx.Player())
	}
}

// HandleMove returns players that fall below the island into the void.
func (h *handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, _ cube.Rotation) {
	if newPos.Y() < VoidY {
		ctx.Cancel()
		h.s.returnFromVoid(ctx.Player(), ctx.Player().Tx())
	}
}
