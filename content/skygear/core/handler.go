package core

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ player.Handler      = (*PlayerHandler)(nil)
	_ world.Handler       = (*WorldHandler)(nil)
	_ world.ColumnHandler = (*WorldHandler)(nil)
)

// PlayerHandler is a player's Dragonfly handler: it turns each event the game
// handles into the hook of the same name. S is read as a field, so an event
// costs no lookup. The other Handle methods are Dragonfly's no-ops.
//
// Each method builds its event once, before the first subscriber runs. Emit
// returns nothing: a subscriber cancels through the event, and Dragonfly
// reads the context it passed.
type PlayerHandler struct {
	player.NopHandler
	S *Session
}

func (h *PlayerHandler) who(p *player.Player) Who {
	return Who{S: h.S, P: p, Tx: p.Tx()}
}

func (h *PlayerHandler) actor(ctx *player.Context) Actor {
	return Actor{Who: h.who(ctx.Player()), Ctx: ctx}
}

func (h *PlayerHandler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	StartBreak.Emit(StartBreakEvent{Actor: h.actor(ctx), Pos: pos})
}

func (h *PlayerHandler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	a := h.actor(ctx)
	held, _ := a.P.HeldItems()
	BlockBreak.Emit(BlockBreakEvent{Actor: a, Pos: pos, Block: a.Tx.Block(pos), Held: held, Drops: drops, XP: xp})
}

func (h *PlayerHandler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	a := h.actor(ctx)
	held, _ := a.P.HeldItems()
	BlockPlace.Emit(BlockPlaceEvent{Actor: a, Pos: pos, Block: b, Held: held})
}

func (h *PlayerHandler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	a := h.actor(ctx)
	held, _ := a.P.HeldItems()
	UseOnBlock.Emit(UseOnBlockEvent{Actor: a, Pos: pos, Face: face, Click: clickPos, Held: held})
}

func (h *PlayerHandler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	UseOnEntity.Emit(EntityEvent{Actor: h.actor(ctx), Target: e})
}

func (h *PlayerHandler) HandleAttackEntity(ctx *player.Context, e world.Entity, _, _ *float64, _ *bool) {
	AttackEntity.Emit(EntityEvent{Actor: h.actor(ctx), Target: e})
}

func (h *PlayerHandler) HandleItemUse(ctx *player.Context) {
	a := h.actor(ctx)
	held, off := a.P.HeldItems()
	Use.Emit(UseEvent{Actor: a, Held: held, Off: off})
}

func (h *PlayerHandler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	a := h.actor(ctx)
	Move.Emit(MoveEvent{Actor: a, From: a.P.Position(), Pos: newPos, Rot: newRot})
}

// HandleToggleSneak emits Sneak when sneaking starts; nothing happens when it
// ends.
func (h *PlayerHandler) HandleToggleSneak(ctx *player.Context, after bool) {
	if after {
		Sneak.Emit(PlayerEvent{h.actor(ctx)})
	}
}

// HandleToggleSprint emits Sprint when sprinting starts; nothing happens when
// it ends.
func (h *PlayerHandler) HandleToggleSprint(ctx *player.Context, after bool) {
	if after {
		Sprint.Emit(PlayerEvent{h.actor(ctx)})
	}
}

func (h *PlayerHandler) HandleJump(p *player.Player) {
	Jump.Emit(NoticeEvent{h.who(p)})
}

func (h *PlayerHandler) HandleItemDamage(ctx *player.Context, it item.Stack, damage *int) {
	ItemDamage.Emit(ItemDamageEvent{Actor: h.actor(ctx), Item: it, Damage: damage})
}

func (h *PlayerHandler) HandlePunchAir(ctx *player.Context) {
	PunchAir.Emit(PlayerEvent{h.actor(ctx)})
}

func (h *PlayerHandler) HandleChat(ctx *player.Context, _ *string) {
	Chat.Emit(PlayerEvent{h.actor(ctx)})
}

func (h *PlayerHandler) HandleItemConsume(ctx *player.Context, it item.Stack) {
	ItemConsume.Emit(ItemEvent{Actor: h.actor(ctx), Item: it})
}

func (h *PlayerHandler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
	ItemPickup.Emit(ItemEvent{Actor: h.actor(ctx), Item: *i, Pick: i})
}

func (h *PlayerHandler) HandleDeath(p *player.Player, _ world.DamageSource, _ *bool) {
	Death.Emit(NoticeEvent{h.who(p)})
}

func (h *PlayerHandler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, _ **world.World) {
	Respawn.Emit(RespawnEvent{Who: h.who(p), Pos: pos})
}

func (h *PlayerHandler) HandleBlockPick(ctx *player.Context, _ cube.Pos, _ world.Block) {
	BlockPick.Emit(PlayerEvent{h.actor(ctx)})
}

func (h *PlayerHandler) HandleHeldSlotChange(ctx *player.Context, _, _ int) {
	HeldSlot.Emit(PlayerEvent{h.actor(ctx)})
}

func (h *PlayerHandler) HandleItemDrop(ctx *player.Context, s item.Stack) {
	ItemDrop.Emit(ItemEvent{Actor: h.actor(ctx), Item: s})
}

func (h *PlayerHandler) HandleHurt(ctx *player.Context, dmg *float64, _ bool, _ *time.Duration, _ world.DamageSource) {
	Hurt.Emit(HurtEvent{Actor: h.actor(ctx), Damage: dmg})
}

func (h *PlayerHandler) HandleCommandExecution(ctx *player.Context, _ cmd.Command, _ []string) {
	Command.Emit(PlayerEvent{h.actor(ctx)})
}

// WorldHandler is the instance world's Dragonfly handler: it emits the
// entity and column hooks. The other Handle methods are Dragonfly's no-ops.
type WorldHandler struct {
	world.NopHandler
	S *Session
}

func (h *WorldHandler) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	EntityDespawn.Emit(DespawnEvent{S: h.S, Tx: tx, Entity: e})
}

func (h *WorldHandler) HandleColumnLoad(tx *world.Tx, _ world.ChunkPos, blockEntities map[cube.Pos]world.Block) {
	ColumnLoad.Emit(ColumnEvent{S: h.S, Tx: tx, Blocks: blockEntities})
}

func (h *WorldHandler) HandleColumnUnload(tx *world.Tx, _ world.ChunkPos, blockEntities map[cube.Pos]world.Block) {
	ColumnUnload.Emit(ColumnEvent{S: h.S, Tx: tx, Blocks: blockEntities})
}
