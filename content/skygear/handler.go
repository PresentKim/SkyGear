package skygear

import (
	"time"

	"github.com/presentkim/skygear/content/skygear/storage"

	"github.com/presentkim/skygear/content/skygear/conduit"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/achievement"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// handler handles a player's events inside a skygear instance.
type handler struct {
	player.NopHandler
	s *session
}

func (h *handler) deny(ctx *player.Context, pos cube.Pos) bool {
	if _, expanding := h.s.expansionOf(ctx.Player()); expanding {
		// No building or clicking in island expansion mode. A click on the
		// map makes the server resend the real blocks, so the map chunk is
		// shown again.
		ctx.Cancel()
		h.s.restoreMapNear(ctx.Player(), pos)
		return true
	}
	if island.CanBuild(h.s.Session, pos) {
		return false
	}
	ctx.Cancel()
	ctx.Player().SendTip(i18n.T(i18n.Default, "island.outside"))
	return true
}

func (h *handler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	if h.deny(ctx, pos) {
		return
	}
	tinker.AutoTool(ctx.Player(), ctx.Player().Tx().Block(pos))
}
func (h *handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, _ *int) {
	if h.deny(ctx, pos) {
		return
	}
	held, _ := ctx.Player().HeldItems()
	broken := ctx.Player().Tx().Block(pos)
	hammer := sgitem.IsHammer(held)
	achievement.CountBreak(ctx.Player(), broken, hammer)
	if _, ok := tinker.ToolOf(ctx.Player()); ok {
		tinker.GainXP(h.s.Session, ctx.Player(), 1)
		tinker.Reinforce(h.s.Session, ctx.Player())
	}
	if hammer {
		if into, ok := sgitem.HammerDrop(broken); ok {
			*drops = []item.Stack{into}
		}
		tinker.VeinMine(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos)
	}
	switch b := broken.(type) {
	case storage.Drawer:
		// Hitting the front takes items out; it never mines the drawer.
		if lp, ok := storage.LastPunch.Of(h.s.Session)[ctx.Player().H()]; ok && lp.Pos == pos && lp.Front {
			ctx.Cancel()
			return
		}
		storage.RemoveDisplay(h.s.Session, ctx.Player().Tx(), pos)
	case machine.Machine:
		machine.Break(h.s.Session, ctx.Player().Tx(), pos, b)
	case block.Composter:
		h.s.breakComposter(pos, b, drops)
	case Barrel:
		h.s.breakBarrel(pos, drops)
	case conduit.Bundle:
		conduit.RemoveVisual(h.s.Session, ctx.Player().Tx(), pos)
	}
}
func (h *handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if h.deny(ctx, pos) {
		return
	}
	achievement.Count(ctx.Player(), "block_place", 1)
	if kind, _, ok := cropOf(b); ok {
		achievement.Count(ctx.Player(), "crop_plant", 1)
		h.s.plantCrop(pos, kind)
	}
	held, _ := ctx.Player().HeldItems()
	switch b := b.(type) {
	case machine.Machine:
		machine.Place(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos, b.Kind)
	case storage.Drawer:
		storage.PlaceDrawer(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos, held)
	case conduit.Bundle:
		achievement.Count(ctx.Player(), "conduit_place", 1)
	case block.DragonEgg:
		h.s.completeDragon(ctx.Player(), ctx.Player().Tx(), pos)
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
	// The configuration copier copies and pastes settings and does nothing else.
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		ctx.Cancel()
		_, off := p.HeldItems()
		h.s.useCopier(p, tx, pos, face, clickPos, held, off)
		return
	}
	// Dragonfly places the held item instead of calling Activate when the
	// player sneaks, so the sneaking click that opens a storage controller's
	// list whatever is held is handled here.
	if p.Sneaking() && !held.Empty() && storage.IsStorageController(tx.Block(pos)) {
		ctx.Cancel()
		storage.UseController(h.s.Session, p, tx, pos)
		return
	}
	// A player who clicks a conduit's connector plate on the block it touches
	// opens its settings, whatever they hold, unless the held conduit goes into
	// that conduit (a click on the conduit itself is ConduitBundle.Activate).
	// Sneaking places blocks instead.
	if !p.Sneaking() && conduit.UsePlate(h.s.Session, p, tx, pos, face, clickPos, held) {
		ctx.Cancel()
		return
	}
	if h.s.useStickOnChest(p, tx, pos, held) || h.s.useStick(p, tx, pos, face, held) {
		ctx.Cancel()
		return
	}
	if !p.Sneaking() && h.s.useComposter(p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	h.s.stripFiber(p, tx, pos, face, held)
	h.s.tillSeeds(p, tx, pos, face, held)
	if h.s.useFairyTime(p, tx, pos, held) {
		ctx.Cancel()
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
	if m, ok := e.(*Monster); ok {
		// the monsters are ours: the damage of the weapon in hand, no vanilla hit
		ctx.Cancel()
		p := ctx.Player()
		held, _ := p.HeldItems()
		dmg := 2.0
		if w, ok := held.Item().(item.Weapon); ok {
			dmg = w.AttackDamage()
		}
		if td, ok := tinker.TinkerDamage(held); ok {
			dmg = td
			mods := tinker.ToolMods(held)
			if v := mods["vampiric"]; v > 0 {
				p.Heal(0.5*float64(v), entity.FoodHealingSource{})
			}
			// a use costs the tool durability and gives it experience
			_, off := p.HeldItems()
			p.SetHeldItems(tinker.DescribeTool(held.Damage(2)), off)
			tinker.Reinforce(h.s.Session, p)
		}
		achievement.Count(p, "hit_monster", 1)
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
	held, off := ctx.Player().HeldItems()
	if isMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
	}
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		h.s.clearCopier(ctx.Player(), held, off)
	}
	// The iron hammer used on the air while sneaking switches diagonal vein
	// mining.
	if sgitem.IsIronHammer(held) && ctx.Player().Sneaking() && !h.s.Debounced(ctx.Player()) {
		tinker.ToggleVeinDiagonal(h.s.Session, ctx.Player())
	}
}

// useHeldScreen opens the menu held by p. The client repeats use packets
// while the button is held, so repeats are ignored.
func (h *handler) useHeldScreen(p *player.Player, _ item.Stack) {
	if h.s.Debounced(p) {
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
	achievement.CountWalk(h.s.Session, ctx.Player(), newPos)
	if newPos.Y() < island.VoidY {
		ctx.Cancel()
		island.ReturnFromVoid(h.s.Session, ctx.Player(), ctx.Player().Tx())
	}
}

// HandleToggleSneak shows the island expansion screen again when sneaking in
// expansion mode, and otherwise grows saplings nearby by chance
// (sapling.go).
func (h *handler) HandleToggleSneak(ctx *player.Context, after bool) {
	if _, ok := h.s.expansionOf(ctx.Player()); ok && after {
		h.s.showExpand(ctx.Player())
		return
	}
	if after {
		achievement.Count(ctx.Player(), "sneak", 1)
		h.s.crouchGrow(ctx.Player(), ctx.Player().Tx())
	}
}

func (h *handler) HandleToggleSprint(ctx *player.Context, after bool) {
	if after {
		achievement.Count(ctx.Player(), "sprint", 1)
	}
}
func (h *handler) HandleJump(p *player.Player) { achievement.Count(p, "jump", 1) }

// HandleItemDamage keeps the iron hammer whole: its vein mining costs food
// instead (vein.go).
func (h *handler) HandleItemDamage(ctx *player.Context, it item.Stack, damage *int) {
	if sgitem.IsIronHammer(it) {
		*damage = 0
	}
}
func (h *handler) HandlePunchAir(ctx *player.Context) {
	achievement.Count(ctx.Player(), "punch_air", 1)
}
func (h *handler) HandleChat(ctx *player.Context, _ *string) {
	achievement.Count(ctx.Player(), "chat", 1)
}
func (h *handler) HandleItemConsume(ctx *player.Context, _ item.Stack) {
	achievement.Count(ctx.Player(), "eat", 1)
}
func (h *handler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
	achievement.Count(ctx.Player(), "pickup", int64(i.Count()))
}

// HandleDeath and HandleRespawn keep the food level through death: dying
// does not fill the food bar (it would make vein mining's food cost free).
func (h *handler) HandleDeath(p *player.Player, _ world.DamageSource, _ *bool) {
	achievement.Count(p, "death", 1)
	if h.s.diedFood == nil {
		h.s.diedFood = map[*world.EntityHandle]int{}
	}
	h.s.diedFood[p.H()] = p.Food()
}
func (h *handler) HandleRespawn(p *player.Player, _ *mgl64.Vec3, _ **world.World) {
	if food, ok := h.s.diedFood[p.H()]; ok {
		delete(h.s.diedFood, p.H())
		p.SetFood(food)
	}
}
func (h *handler) HandleBlockPick(ctx *player.Context, _ cube.Pos, _ world.Block) {
	achievement.Count(ctx.Player(), "block_pick", 1)
}
func (h *handler) HandleHeldSlotChange(ctx *player.Context, _, _ int) {
	achievement.Count(ctx.Player(), "hotbar", 1)
}
func (h *handler) HandleItemDrop(ctx *player.Context, _ item.Stack) {
	achievement.Count(ctx.Player(), "item_drop", 1)
}
func (h *handler) HandleHurt(ctx *player.Context, _ *float64, _ bool, _ *time.Duration, _ world.DamageSource) {
	achievement.Count(ctx.Player(), "hurt", 1)
}
func (h *handler) HandleCommandExecution(ctx *player.Context, _ cmd.Command, _ []string) {
	achievement.Count(ctx.Player(), "command", 1)
}
