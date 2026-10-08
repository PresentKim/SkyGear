package platform

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// lobbyHandler protects the lobby: no building, damage, dropping or PvP.
type lobbyHandler struct {
	player.NopHandler
	pf *Platform
}

func (lobbyHandler) HandleStartBreak(ctx *player.Context, _ cube.Pos) { ctx.Cancel() }
func (lobbyHandler) HandleBlockBreak(ctx *player.Context, _ cube.Pos, _ *[]item.Stack, _ *int) {
	ctx.Cancel()
}
func (lobbyHandler) HandleBlockPlace(ctx *player.Context, _ cube.Pos, _ world.Block) { ctx.Cancel() }
func (lobbyHandler) HandleItemDrop(ctx *player.Context, _ item.Stack)                { ctx.Cancel() }
func (lobbyHandler) HandleHurt(ctx *player.Context, _ *float64, _ bool, _ *time.Duration, src world.DamageSource) {
	// Void damage is left to the platform handler, which returns the player.
	ctx.Cancel()
}
func (lobbyHandler) HandleFoodLoss(ctx *player.Context, _ int, _ *int) { ctx.Cancel() }
func (lobbyHandler) HandleAttackEntity(ctx *player.Context, _ world.Entity, _, _ *float64, _ *bool) {
	ctx.Cancel()
}
func (h lobbyHandler) HandleItemUse(ctx *player.Context) {
	if h.pf.useLobbyItem(ctx.Player()) {
		ctx.Cancel()
	}
}
func (h lobbyHandler) HandleItemUseOnBlock(ctx *player.Context, _ cube.Pos, _ cube.Face, _ mgl64.Vec3) {
	ctx.Cancel()
	h.pf.useLobbyItem(ctx.Player())
}
