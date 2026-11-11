package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// handler handles a player's events inside a skygear instance. The methods it
// does not declare are the kernel's: they emit the hook of the same name.
// A method is declared here (a legacy body) or served by the hook, never both
// (TestHandlerMethodsOneWay).
type handler struct {
	core.PlayerHandler
	s *session
}

// deny reports whether the action at pos is refused, with the marks of the two
// guards under suffix. The block events run the same two halves as the
// subscribers expand.guard and island.guard.
func (h *handler) deny(ctx *player.Context, pos cube.Pos, suffix string) bool {
	if core.Trace != nil {
		core.Mark("expand." + suffix)
	}
	if expandGuard(h.s.Session, ctx, pos) {
		return true
	}
	if core.Trace != nil {
		core.Mark("island." + suffix)
	}
	return islandGuard(h.s.Session, ctx, pos)
}

func (h *handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	core.Mark("dimension.hunting")
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		ctx.Cancel()
		return
	}
	held, _ := ctx.Player().HeldItems()
	core.Mark("screens.menuItem")
	if screens.IsMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	core.Mark("dimension.stone")
	if h.useStone(ctx, held) {
		return
	}
	if h.deny(ctx, pos, "guard") {
		return
	}
	p, tx := ctx.Player(), ctx.Player().Tx()
	core.Mark("machine.noteFace")
	if m, ok := held.Item().(machine.Machine); ok && (machine.NetShaped(m.Kind) || m.Kind == machine.KindMiner) {
		machine.NotePlaceFace(h.s.Session, p, face)
	}
	// The configuration copier copies and pastes settings and does nothing else.
	core.Mark("copier.use")
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		ctx.Cancel()
		_, off := p.HeldItems()
		copier.UseCopier(h.s.Session, p, tx, pos, face, clickPos, held, off)
		return
	}
	// The construction wand builds out the face clicked and does nothing else.
	core.Mark("wand.use")
	if _, ok := sgitem.WandOf(held); ok {
		ctx.Cancel()
		wand.Use(h.s.Session, p, tx, pos, face, held)
		return
	}
	core.Mark("storage.sneakUse")
	if storage.SneakUse(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	// A player who clicks a conduit's connector plate on the block it touches
	// opens its settings, whatever they hold, unless the held conduit goes into
	// that conduit (a click on the conduit itself is ConduitBundle.Activate).
	// Sneaking places blocks instead.
	core.Mark("conduit.plate")
	if !p.Sneaking() && conduit.UsePlate(h.s.Session, p, tx, pos, face, clickPos, held) {
		ctx.Cancel()
		return
	}
	core.Mark("handwork.stick")
	if handwork.UseStickOnChest(h.s.Session, p, tx, pos, held) || handwork.UseStick(h.s.Session, p, tx, pos, face, held) {
		ctx.Cancel()
		return
	}
	core.Mark("farm.composter")
	if !p.Sneaking() && farm.UseComposter(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	core.Mark("handwork.gather")
	handwork.StripFiber(h.s.Session, p, tx, pos, face, held)
	handwork.TillSeeds(h.s.Session, p, tx, pos, face, held)
	core.Mark("fairy.time")
	if fairy.UseFairyTime(h.s.Session, p, tx, pos, held) {
		ctx.Cancel()
		return
	}
	// Using a crop shows its time or harvests it, and places nothing;
	// sneaking uses it as usual (placing blocks, bone meal).
	core.Mark("farm.crop")
	if !ctx.Player().Sneaking() && farm.UseCrop(h.s.Session, ctx.Player(), ctx.Player().Tx(), pos) {
		ctx.Cancel()
		return
	}
	h.deny(ctx, pos.Side(face), "guardSide")
}

// HandleItemUseOnEntity and HandleAttackEntity take the gift of a visiting
// fairy, whichever way the player touches it.
func (h *handler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	core.Mark("dimension.hunting")
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		ctx.Cancel()
		return
	}
	core.Mark("fairy.touch")
	h.touchFairy(ctx, e)
}

func (h *handler) HandleAttackEntity(ctx *player.Context, e world.Entity, _, _ *float64, _ *bool) {
	core.Mark("dimension.hunter")
	if _, ok := e.(*dimension.Monster); ok || dimension.Hunting(h.s.Session, ctx.Player()) {
		// a hunter only moves: its weapons strike on their own (dimension/gear.go)
		ctx.Cancel()
		return
	}
	core.Mark("fairy.touch")
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
	core.Mark("autofill.use")
	if held.Count() == 1 {
		core.RefillHandLater(ctx.Player(), ctx.Player().Tx(), held)
	}
	core.Mark("screens.menuItem")
	if screens.IsMenuItem(held) {
		ctx.Cancel()
		h.useHeldScreen(ctx.Player(), held)
		return
	}
	core.Mark("dimension.hunting")
	if dimension.Hunting(h.s.Session, ctx.Player()) {
		// a hunter only moves; the menu item is the way out
		ctx.Cancel()
		return
	}
	core.Mark("dimension.stone")
	if h.useStone(ctx, held) {
		return
	}
	core.Mark("copier.clear")
	if _, ok := held.Item().(sgitem.ConfigCopier); ok {
		copier.ClearCopier(h.s.Session, ctx.Player(), held, off)
	}
	// The construction wand used on the air while sneaking shows its settings.
	core.Mark("wand.options")
	if _, ok := sgitem.WandOf(held); ok && ctx.Player().Sneaking() && !h.s.Debounced(ctx.Player()) {
		wand.ShowOptions(h.s.Session, ctx.Player())
	}
	// The iron hammer used on the air while sneaking switches diagonal vein
	// mining.
	core.Mark("tinker.veinToggle")
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
