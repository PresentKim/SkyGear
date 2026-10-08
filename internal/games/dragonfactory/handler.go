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
	if c, ok := ctx.Player().Tx().Block(pos).(Core); ok {
		h.s.breakCore(c, pos)
	}
}
func (h *handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if h.deny(ctx, pos) {
		return
	}
	if _, ok := b.(Core); ok {
		held, _ := ctx.Player().HeldItems()
		if !h.s.placeCore(ctx.Player(), ctx.Player().Tx(), pos, held) {
			ctx.Cancel()
		}
	}
}
func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, _ mgl64.Vec3) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) {
		ctx.Cancel()
		h.s.showMenu(ctx.Player())
		return
	}
	if !h.deny(ctx, pos) {
		h.deny(ctx, pos.Side(face))
	}
}
func (h *handler) HandleItemUse(ctx *player.Context) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) {
		ctx.Cancel()
		h.s.showMenu(ctx.Player())
	}
}

// HandleMove returns players that fall below the island into the void.
func (h *handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, _ cube.Rotation) {
	if newPos.Y() < VoidY {
		ctx.Cancel()
		h.s.returnFromVoid(ctx.Player(), ctx.Player().Tx())
	}
}
