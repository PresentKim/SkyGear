package instance

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// Handler is the handler every player has for its whole connection. It
// applies platform rules and forwards all events to the handler of the
// player's current location (the lobby or the game session).
type Handler struct {
	m   *Manager
	loc atomic.Pointer[location]
}

// location is where the player is: inst is nil in the lobby.
type location struct {
	inst  *Instance
	inner player.Handler
}

func (h *Handler) set(inst *Instance, inner player.Handler) {
	if inner == nil {
		inner = player.NopHandler{}
	}
	h.loc.Store(&location{inst: inst, inner: inner})
}

func (h *Handler) instance() *Instance {
	if l := h.loc.Load(); l != nil {
		return l.inst
	}
	return nil
}

func (h *Handler) in() player.Handler {
	if l := h.loc.Load(); l != nil {
		return l.inner
	}
	return player.NopHandler{}
}

// HandleHurt removes fall damage everywhere (docs/design/02-dragonfactory.md §14.1)
// and returns players that reach the void to a spawn if the location's
// handler does not handle it.
func (h *Handler) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
	if _, ok := src.(entity.FallDamageSource); ok {
		ctx.Cancel()
		return
	}
	h.in().HandleHurt(ctx, damage, immune, attackImmunity, src)
	if _, ok := src.(entity.VoidDamageSource); ok && !ctx.Cancelled() {
		ctx.Cancel()
		p := ctx.Player()
		if inst := h.instance(); inst != nil {
			p.Teleport(inst.session.Spawn())
		} else {
			p.Teleport(h.m.lobby.Spawn)
		}
	}
}

// HandleQuit stores the player's state if it was in an instance.
func (h *Handler) HandleQuit(p *player.Player) {
	h.in().HandleQuit(p)
	if inst := h.instance(); inst != nil {
		h.m.quit(p, inst)
	}
	if h.m.lobby.Quit != nil {
		h.m.lobby.Quit(p)
	}
}

func (h *Handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	h.in().HandleMove(ctx, newPos, newRot)
}
func (h *Handler) HandleJump(p *player.Player) { h.in().HandleJump(p) }
func (h *Handler) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	h.in().HandleTeleport(ctx, pos)
}
func (h *Handler) HandleChangeWorld(p *player.Player, before, after *world.World) {
	h.in().HandleChangeWorld(p, before, after)
}
func (h *Handler) HandleToggleSprint(ctx *player.Context, after bool) {
	h.in().HandleToggleSprint(ctx, after)
}
func (h *Handler) HandleToggleSneak(ctx *player.Context, after bool) {
	h.in().HandleToggleSneak(ctx, after)
}

// HandleChat keeps chat within a save: a message reaches the players of the
// same instance only, with the display name (ChatName) instead of
// Dragonfly's plain "<name> message". The lobby has no chat
// (docs/design/01-platform.md §1). The server console gets every message.
func (h *Handler) HandleChat(ctx *player.Context, message *string) {
	h.in().HandleChat(ctx, message)
	if ctx.Cancelled() {
		return
	}
	ctx.Cancel()
	if h.instance() == nil {
		return
	}
	p := ctx.Player()
	name := p.Name()
	if h.m.lobby.ChatName != nil {
		name = h.m.lobby.ChatName(p)
	}
	line := fmt.Sprintf("<%s> %s", name, *message)
	for e := range p.Tx().Players() {
		e.(*player.Player).Message(line)
	}
	chat.StdoutSubscriber{}.Message(line)
}
func (h *Handler) HandleFoodLoss(ctx *player.Context, from int, to *int) {
	h.in().HandleFoodLoss(ctx, from, to)
}
func (h *Handler) HandleHeal(ctx *player.Context, health *float64, src world.HealingSource) {
	h.in().HandleHeal(ctx, health, src)
}
func (h *Handler) HandleSetOnFire(ctx *player.Context, duration *time.Duration) {
	h.in().HandleSetOnFire(ctx, duration)
}
func (h *Handler) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	// Inventories are kept on death everywhere (docs/design/02-dragonfactory.md §14.1).
	*keepInv = true
	h.in().HandleDeath(p, src, keepInv)
}
func (h *Handler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	if inst := h.instance(); inst != nil {
		*pos, *w = inst.session.Spawn(), inst.World
	} else {
		*pos, *w = h.m.lobby.Spawn, h.m.lobby.World
	}
	h.in().HandleRespawn(p, pos, w)
}
func (h *Handler) HandleSkinChange(ctx *player.Context, skin *skin.Skin) {
	h.in().HandleSkinChange(ctx, skin)
}
func (h *Handler) HandleFireExtinguish(ctx *player.Context, pos cube.Pos) {
	h.in().HandleFireExtinguish(ctx, pos)
}
func (h *Handler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleStartBreak(ctx, pos)
}
func (h *Handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleBlockBreak(ctx, pos, drops, xp)
}
func (h *Handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleBlockPlace(ctx, pos, b)
}
func (h *Handler) HandleBlockPick(ctx *player.Context, pos cube.Pos, b world.Block) {
	h.in().HandleBlockPick(ctx, pos, b)
}
func (h *Handler) HandleItemUse(ctx *player.Context) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleItemUse(ctx)
}
func (h *Handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	if expiredHeld(ctx) {
		return
	}
	sweepContainer(ctx, pos)
	h.in().HandleItemUseOnBlock(ctx, pos, face, clickPos)
}
func (h *Handler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleItemUseOnEntity(ctx, e)
}
func (h *Handler) HandleItemRelease(ctx *player.Context, it item.Stack, dur time.Duration) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleItemRelease(ctx, it, dur)
}
func (h *Handler) HandleItemConsume(ctx *player.Context, it item.Stack) {
	if timed.Expired(it) {
		ctx.Cancel()
		timed.SweepPlayer(ctx.Player())
		return
	}
	h.in().HandleItemConsume(ctx, it)
}
func (h *Handler) HandleAttackEntity(ctx *player.Context, e world.Entity, force, height *float64, critical *bool) {
	if expiredHeld(ctx) {
		return
	}
	h.in().HandleAttackEntity(ctx, e, force, height, critical)
}
func (h *Handler) HandleExperienceGain(ctx *player.Context, amount *int) {
	h.in().HandleExperienceGain(ctx, amount)
}
func (h *Handler) HandlePunchAir(ctx *player.Context) { h.in().HandlePunchAir(ctx) }
func (h *Handler) HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText, newText string) {
	h.in().HandleSignEdit(ctx, pos, frontSide, oldText, newText)
}
func (h *Handler) HandleSleep(ctx *player.Context, sendReminder *bool) {
	h.in().HandleSleep(ctx, sendReminder)
}
func (h *Handler) HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int) {
	h.in().HandleLecternPageTurn(ctx, pos, oldPage, newPage)
}
func (h *Handler) HandleItemDamage(ctx *player.Context, i item.Stack, damage *int) {
	h.in().HandleItemDamage(ctx, i, damage)
}
func (h *Handler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
	if timed.Expired(*i) {
		// The item entity is left to despawn or fall into the void.
		ctx.Cancel()
		return
	}
	h.in().HandleItemPickup(ctx, i)
}
func (h *Handler) HandleHeldSlotChange(ctx *player.Context, from, to int) {
	h.in().HandleHeldSlotChange(ctx, from, to)
}
func (h *Handler) HandleItemDrop(ctx *player.Context, s item.Stack) { h.in().HandleItemDrop(ctx, s) }
func (h *Handler) HandleTransfer(ctx *player.Context, addr *net.UDPAddr) {
	h.in().HandleTransfer(ctx, addr)
}
func (h *Handler) HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string) {
	h.in().HandleCommandExecution(ctx, command, args)
}
func (h *Handler) HandleDiagnostics(p *player.Player, d session.Diagnostics) {
	h.in().HandleDiagnostics(p, d)
}

var _ player.Handler = (*Handler)(nil)

// expiredHeld cancels an action done with an expired time-limited item in
// hand and removes the player's expired items. The once-a-second sweep
// (Instance.sweepTimed) would remove it too; this closes the gap until then.
func expiredHeld(ctx *player.Context) bool {
	main, off := ctx.Player().HeldItems()
	if !timed.Expired(main) && !timed.Expired(off) {
		return false
	}
	ctx.Cancel()
	timed.SweepPlayer(ctx.Player())
	return true
}

// sweepContainer removes expired items from a container a player uses, so
// items kept in chests expire too.
func sweepContainer(ctx *player.Context, pos cube.Pos) {
	tx := ctx.Player().Tx()
	if c, ok := tx.Block(pos).(block.Container); ok {
		timed.Sweep(c.Inventory(tx, pos))
	}
}
