package skygear

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/bridge"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// handler handles a player's events inside a skygear instance.
type handler struct {
	player.NopHandler
	s *session
}

func (h *handler) deny(ctx *player.Context, pos cube.Pos) bool {
	if _, expanding := expand.Of(h.s.Session, ctx.Player()); expanding {
		// No building or clicking in island expansion mode. A click on the
		// map makes the server resend the real blocks, so the map chunk is
		// shown again.
		ctx.Cancel()
		expand.RestoreMapNear(h.s.Session, ctx.Player(), pos)
		return true
	}
	if island.CanBuild(h.s.Session, pos) {
		return false
	}
	ctx.Cancel()
	if core.InArena(pos.Vec3()) {
		return true // the hunting ground is no one's to build on
	}
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
		if storage.IsSetMachine(b.Kind) {
			storage.InvalidateNetworks(h.s.Session)
		}
		if netstore.IsNode(b.Kind) {
			// The block goes after this handler: drop the network once it is gone.
			ctx.Player().Tx().Defer(func(tx *world.Tx) {
				netstore.Invalidate(tx, pos)
				netstore.Forget(h.s.Session, pos)
			})
		}
		machine.Break(h.s.Session, ctx.Player().Tx(), pos, b)
	case block.Composter:
		farm.BreakComposter(h.s.Session, pos, b, drops)
	case farm.Barrel:
		farm.BreakBarrel(h.s.Session, pos, drops)
	case conduit.Bundle:
		conduit.RemoveVisual(h.s.Session, ctx.Player().Tx(), pos)
	}
}
func (h *handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	if h.deny(ctx, pos) {
		return
	}
	achievement.Count(ctx.Player(), "block_place", 1)
	if held, _ := ctx.Player().HeldItems(); held.Count() == 1 {
		core.RefillHandLater(ctx.Player(), ctx.Player().Tx(), held)
	}
	if kind, _, ok := farm.CropOf(b); ok {
		achievement.Count(ctx.Player(), "crop_plant", 1)
		farm.PlantCrop(h.s.Session, pos, kind)
	}
	held, _ := ctx.Player().HeldItems()
	switch b := b.(type) {
	case machine.Machine:
		if storage.IsSetMachine(b.Kind) {
			storage.InvalidateNetworks(h.s.Session)
		}
		machine.Place(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos, b.Kind)
		if netstore.IsNode(b.Kind) {
			netstore.Forget(h.s.Session, pos) // a node broken by an explosion left its setting
			// machine.Place sets the block in a deferred call; this one runs after it.
			ctx.Player().Tx().Defer(func(tx *world.Tx) { netstore.Invalidate(tx, pos) })
		}
	case storage.Drawer:
		storage.PlaceDrawer(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos, held)
	case conduit.Bundle:
		achievement.Count(ctx.Player(), "conduit_place", 1)
	case block.DragonEgg:
		h.s.completeDragon(ctx.Player(), ctx.Player().Tx(), pos)
	}
}
func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		ctx.Cancel()
		return
	}
	held, _ := ctx.Player().HeldItems()
	if screens.IsMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	if h.useStone(ctx, held) {
		return
	}
	if h.deny(ctx, pos) {
		return
	}
	p, tx := ctx.Player(), ctx.Player().Tx()
	if m, ok := held.Item().(machine.Machine); ok && (machine.NetShaped(m.Kind) || m.Kind == machine.KindMiner) {
		machine.NotePlaceFace(h.s.Session, p, face)
	}
	// The configuration copier copies and pastes settings and does nothing else.
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		ctx.Cancel()
		_, off := p.HeldItems()
		copier.UseCopier(h.s.Session, p, tx, pos, face, clickPos, held, off)
		return
	}
	// The construction wand builds out the face clicked and does nothing else.
	if _, ok := sgitem.WandOf(held); ok {
		ctx.Cancel()
		wand.Use(h.s.Session, p, tx, pos, face, held)
		return
	}
	if storage.SneakUse(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
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
	if handwork.UseStickOnChest(h.s.Session, p, tx, pos, held) || handwork.UseStick(h.s.Session, p, tx, pos, face, held) {
		ctx.Cancel()
		return
	}
	if !p.Sneaking() && farm.UseComposter(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	handwork.StripFiber(h.s.Session, p, tx, pos, face, held)
	handwork.TillSeeds(h.s.Session, p, tx, pos, face, held)
	if fairy.UseFairyTime(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	// Using a crop shows its time or harvests it, and places nothing;
	// sneaking uses it as usual (placing blocks, bone meal).
	if !ctx.Player().Sneaking() && farm.UseCrop(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos) {
		ctx.Cancel()
		return
	}
	h.deny(ctx, pos.Side(face))
}

// HandleItemUseOnEntity and HandleAttackEntity take the gift of a visiting
// fairy, whichever way the player touches it.
func (h *handler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		ctx.Cancel()
		return
	}
	h.touchFairy(ctx, e)
}

func (h *handler) HandleAttackEntity(ctx *player.Context, e world.Entity, _, _ *float64, _ *bool) {
	if _, ok := e.(*dimension.Monster); ok || dimension.Hunting(h.s.Session, ctx.Player()) {
		// a hunter only moves: its weapons strike on their own (dimension/gear.go)
		ctx.Cancel()
		return
	}
	h.touchFairy(ctx, e)
}

func (h *handler) touchFairy(ctx *player.Context, e world.Entity) {
	if f, ok := fairy.Of(h.s.Session, e); ok {
		ctx.Cancel()
		f.Gift(ctx.Player().Tx(), h.s.Session, ctx.Player())
	}
}

func (h *handler) HandleItemUse(ctx *player.Context) {
	held, off := ctx.Player().HeldItems()
	// a thrown snowball, egg or pearl may be the last of the stack
	if held.Count() == 1 {
		core.RefillHandLater(ctx.Player(), ctx.Player().Tx(), held)
	}
	if screens.IsMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		// a hunter only moves; the menu item is the way out
		ctx.Cancel()
		return
	}
	if h.useStone(ctx, held) {
		return
	}
	// A block used on the air goes where the bridging assist finds a place
	// for it (bridge).
	if p := ctx.Player(); bridge.Place(h.s.Session, p) {
		ctx.Cancel()
		h.updateBridge(p, p.Position(), p.Rotation(), true)
		return
	}
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		copier.ClearCopier(h.s.Session, ctx.Player(), held, off)
	}
	// The construction wand used on the air while sneaking shows its settings.
	if _, ok := sgitem.WandOf(held); ok && ctx.Player().Sneaking() && !h.s.Debounced(ctx.Player()) {
		wand.ShowOptions(h.s.Session, ctx.Player())
	}
	// The iron hammer used on the air while sneaking switches diagonal vein
	// mining.
	if sgitem.IsIronHammer(held) && ctx.Player().Sneaking() && !h.s.Debounced(ctx.Player()) {
		tinker.ToggleVeinDiagonal(h.s.Session, ctx.Player())
	}
}

// useStone opens the second dimension's window for a held teleport stone,
// on a block or the air; it reports whether held is one.
func (h *handler) useStone(ctx *player.Context, held item.Stack) bool {
	if _, ok := held.Item().(sgitem.DimensionStone); !ok {
		return false
	}
	ctx.Cancel()
	if !h.s.Debounced(ctx.Player()) {
		dimension.ShowWindow(h.s.Session, ctx.Player(), "")
	}
	return true
}

// useHeldScreen opens the menu held by p. The client repeats use packets
// while the button is held, so repeats are ignored.
func (h *handler) useHeldScreen(p *player.Player, _ item.Stack) {
	if h.s.Debounced(p) {
		return
	}
	if dimension.Hunting(h.s.Session, p) {
		dimension.ShowRunMenu(h.s.Session, p)
		return
	}
	if dimension.Watching(h.s.Session, p) {
		dimension.ShowWatchMenu(h.s.Session, p)
		return
	}
	screens.OpenMenu(h.s.Session, p)
}

// HandleMove returns players that fall below the island into the void, and
// keeps players in island expansion mode in place.
func (h *handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	look.Update(h.s.Session, ctx.Player(), ctx.Player().Tx(), newPos, newRot, false)
	if _, expanding := expand.Of(h.s.Session, ctx.Player()); expanding {
		ctx.Cancel()
		return
	}
	h.updateBridge(ctx.Player(), newPos, newRot, false)
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
	if _, ok := expand.Of(h.s.Session, ctx.Player()); ok && after {
		expand.Show(h.s.Session, ctx.Player())
		return
	}
	if after {
		achievement.Count(ctx.Player(), "sneak", 1)
		farm.CrouchGrow(h.s.Session, ctx.Player(), ctx.Player().Tx())
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
	// a held tool that breaks is replaced by another of its kind
	if held, _ := ctx.Player().HeldItems(); *damage > 0 && it.Durability() >= 0 && *damage >= it.Durability() && sameItemName(held, it) {
		core.RefillHandLater(ctx.Player(), ctx.Player().Tx(), it)
	}
}

// sameItemName reports whether a and b are the same item, however worn.
func sameItemName(a, b item.Stack) bool {
	n1, m1 := a.Item().EncodeItem()
	n2, m2 := b.Item().EncodeItem()
	return !a.Empty() && !b.Empty() && n1 == n2 && m1 == m2
}
func (h *handler) HandlePunchAir(ctx *player.Context) {
	achievement.Count(ctx.Player(), "punch_air", 1)
}
func (h *handler) HandleChat(ctx *player.Context, _ *string) {
	achievement.Count(ctx.Player(), "chat", 1)
}
func (h *handler) HandleItemConsume(ctx *player.Context, it item.Stack) {
	achievement.Count(ctx.Player(), "eat", 1)
	core.RefillHandLater(ctx.Player(), ctx.Player().Tx(), it)
}
func (h *handler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
	achievement.Count(ctx.Player(), "pickup", int64(i.Count()))
}

// HandleDeath and HandleRespawn keep the food level through death: dying
// does not fill the food bar (it would make vein mining's food cost free).
func (h *handler) HandleDeath(p *player.Player, _ world.DamageSource, _ *bool) {
	achievement.Count(p, "death", 1)
	dimension.Died(h.s.Session, p.Tx(), p)
	if h.s.diedFood == nil {
		h.s.diedFood = map[*world.EntityHandle]int{}
	}
	h.s.diedFood[p.H()] = p.Food()
}
func (h *handler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, _ **world.World) {
	dimension.Respawned(h.s.Session, p, pos)
	if food, ok := h.s.diedFood[p.H()]; ok {
		delete(h.s.diedFood, p.H())
		p.SetFood(food)
	}
}
func (h *handler) HandleBlockPick(ctx *player.Context, _ cube.Pos, _ world.Block) {
	achievement.Count(ctx.Player(), "block_pick", 1)
}
func (h *handler) HandleHeldSlotChange(ctx *player.Context, _, to int) {
	achievement.Count(ctx.Player(), "hotbar", 1)
	p := ctx.Player()
	held, _ := p.Inventory().Item(to)
	if bridge.Update(h.s.Session, p, p.Tx(), p.Position(), p.Rotation(), held, true) {
		screens.UpdateBoard(h.s.Session, p)
	}
}

// updateBridge updates p's bridging assist crosshair mark for p at pos
// with rotation rot, and the HUD that shows it when it changed.
func (h *handler) updateBridge(p *player.Player, pos mgl64.Vec3, rot cube.Rotation, force bool) {
	held, _ := p.HeldItems()
	if bridge.Update(h.s.Session, p, p.Tx(), pos, rot, held, force) {
		screens.UpdateBoard(h.s.Session, p)
	}
}
func (h *handler) HandleItemDrop(ctx *player.Context, _ item.Stack) {
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		ctx.Cancel()
		return
	}
	achievement.Count(ctx.Player(), "item_drop", 1)
}
func (h *handler) HandleHurt(ctx *player.Context, dmg *float64, _ bool, _ *time.Duration, _ world.DamageSource) {
	if dimension.Frozen(h.s.Session, ctx.Player()) {
		ctx.Cancel() // the hunt waits while the hunter chooses an upgrade
		return
	}
	achievement.Count(ctx.Player(), "hurt", 1)
	if dimension.Fallen(h.s.Session, ctx.Player().Tx(), ctx.Player(), *dmg) {
		ctx.Cancel()
	}
}
func (h *handler) HandleCommandExecution(ctx *player.Context, _ cmd.Command, _ []string) {
	achievement.Count(ctx.Player(), "command", 1)
}
