package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
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
	ctx.Player().SendTip(i18n.T(i18n.Default, "dragonfactory.handler.deny.1"))
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
	case block.Composter:
		h.s.breakComposter(pos, b, drops)
	case Barrel:
		h.s.breakBarrel(pos)
	case ConduitBundle:
		h.s.removeVisual(ctx.Player().Tx(), pos)
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
	case Drawer:
		h.s.placeDrawer(ctx.Player(), ctx.Player().Tx(), pos, held)
	}
}
func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	held, _ := ctx.Player().HeldItems()
	if isMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	if h.deny(ctx, pos) {
		return
	}
	p, tx := ctx.Player(), ctx.Player().Tx()
	// A sneaking player who clicks the face of a block where a conduit's
	// connector plate sits, or the side of a conduit that has one, opens the
	// settings of that side, whatever they hold (a conduit item in hand places).
	if p.Sneaking() && !isConduitItem(held) {
		// One click sends several use events; only the first opens a form.
		if at, _, side, ok := plateHit(tx, pos, face, clickPos); ok {
			ctx.Cancel()
			if !h.s.debounced(p) {
				h.s.showPlateForm(p, tx, at, side)
			}
			return
		}
		if b, ok := tx.Block(pos).(ConduitBundle); ok {
			if len(plateKinds(tx, pos, b, faceIndex(face))) > 0 {
				ctx.Cancel()
				if !h.s.debounced(p) {
					h.s.showPlateForm(p, tx, pos, faceIndex(face))
				}
				return
			}
			if bundleHasPlate(tx, pos, b) {
				ctx.Cancel()
				if !h.s.debounced(p) {
					if side, found := h.s.nextPlateSide(p, tx, pos, b); found {
						h.s.showPlateForm(p, tx, pos, side)
					}
				}
				return
			}
		}
	}
	if h.s.useStick(p, tx, pos, face, held) {
		ctx.Cancel()
		return
	}
	if !p.Sneaking() && h.s.useComposter(p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	h.s.stripFiber(tx, pos, face, held)
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
	if m, ok := e.(*Monster); ok {
		// the monsters are ours: the damage of the weapon in hand, no vanilla hit
		ctx.Cancel()
		p := ctx.Player()
		held, _ := p.HeldItems()
		dmg := 2.0
		if w, ok := held.Item().(item.Weapon); ok {
			dmg = w.AttackDamage()
		}
		m.Hurt(p.Tx(), dmg, p)
		return
	}
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
	if isMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
	}
}

// useHeldScreen opens the menu held by p. The client repeats use packets
// while the button is held, so repeats are ignored.
func (h *handler) useHeldScreen(p *player.Player, _ item.Stack) {
	if h.s.debounced(p) {
		return
	}
	h.s.openMenu(p)
}

// HandleMove returns players that fall below the island into the void, and
// keeps players in island expansion mode in place.
func (h *handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	h.s.updateLook(ctx.Player(), ctx.Player().Tx(), newPos, newRot, false)
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
// expansion mode, and otherwise grows saplings nearby by chance
// (sapling.go).
func (h *handler) HandleToggleSneak(ctx *player.Context, after bool) {
	if _, ok := h.s.expansionOf(ctx.Player()); ok && after {
		h.s.showExpandActions(ctx.Player())
		return
	}
	if after {
		h.s.crouchGrow(ctx.Player(), ctx.Player().Tx())
	}
}
