package skygear

import (
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
)

// TestPinUse records, per scenario, the marks that HandleItemUseOnBlock,
// HandleItemUse, HandleItemUseOnEntity and HandleAttackEntity reach and the
// outcome, into testdata/pin/handler-use.golden.
//
// Every outcome has cancelled= and screens=: the screens the call asked the
// UI to show, named by the innermost function outside common/ui that asked
// (a test player has no connection, so nothing is drawn; the sink is
// ui.InLobby, which showScreen consults first). Most scenarios add the
// effect that is cheap to read: the held stack, the items dropped, the card
// a copier holds, the block that ended up there.
//
// A repeat scenario ("...Repeat") first makes one untraced use, then traces
// the second one at once: the client sends a click twice and repeats it while
// the button is held, and Session.Debounced (500ms) or the stick's own
// cooldown (100ms) answers the repeat. "...AfterWindow" forgets the first
// use first, as if the window had passed.
//
// Each scenario runs on a fresh session and player. The golden is recorded
// from the legacy code; it may be re-recorded (SKYGEAR_GOLDEN=update) only
// when the legacy behaviour itself is being changed on purpose.
func TestPinUse(t *testing.T) {
	lines := map[string]string{}
	pos := cube.Pos{4, island.IslandY + 2, 4}
	outsidePos := cube.Pos{84, island.IslandY + 2, 4}
	centre := mgl64.Vec3{0.5, 0.5, 0.5}
	type setupFn = func(s *session, p *player.Player, tx *world.Tx)

	type scene struct {
		s *session
		h *world.EntityHandle
	}
	// retire takes the players out of the world when the test ends, before
	// the world closes: a hunt or a watch schedules the player's landing a
	// moment later (instance.PlaceAt), and that work must not meet the
	// closing world.
	retire := func(s *session, hs ...*world.EntityHandle) {
		t.Cleanup(func() {
			s.World.Do(func(tx *world.Tx) {
				for _, h := range hs {
					if e, ok := h.Entity(tx); ok {
						tx.RemoveEntity(e)
					}
				}
			})
		})
	}
	start := func(setup setupFn) scene {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		retire(s, h)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			if setup != nil {
				setup(s, p, tx)
			}
		})
		return scene{s, h}
	}
	join := func(parts ...string) string {
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return strings.Join(out, " ")
	}
	chain := func(fs ...setupFn) setupFn {
		return func(s *session, p *player.Player, tx *world.Tx) {
			for _, f := range fs {
				if f != nil {
					f(s, p, tx)
				}
			}
		}
	}
	holding := func(it item.Stack) setupFn {
		return func(s *session, p *player.Player, tx *world.Tx) { p.SetHeldItems(it, item.Stack{}) }
	}
	blockAt := func(at cube.Pos, b world.Block) setupFn {
		return func(s *session, p *player.Player, tx *world.Tx) { tx.SetBlock(at, b, nil) }
	}
	sneaking := func(s *session, p *player.Player, tx *world.Tx) { p.StartSneaking() }
	expanding := func(s *session, p *player.Player, tx *world.Tx) { expand.Show(s.Session, p) }
	hunting := func(s *session, p *player.Player, tx *world.Tx) {
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
	}
	slot := func(i int, it item.Stack) setupFn {
		return func(s *session, p *player.Player, tx *world.Tx) { _ = p.Inventory().SetItem(i, it) }
	}

	// screensOf collects the screens asked of the UI while call runs.
	var shown []string
	asked := func(call func()) string {
		saved := ui.InLobby
		shown = nil
		ui.InLobby = func(*player.Player) bool {
			pcs := make([]uintptr, 32)
			n := runtime.Callers(2, pcs)
			frames := runtime.CallersFrames(pcs[:n])
			viaScreen, name := false, ""
			for {
				f, more := frames.Next()
				if strings.HasSuffix(f.Function, "ui.showScreen") {
					viaScreen = true
				} else if viaScreen && name == "" && !strings.Contains(f.Function, "/common/ui.") {
					name = f.Function[strings.LastIndex(f.Function, "/skygear/")+len("/skygear/"):]
				}
				if !more {
					break
				}
			}
			if viaScreen {
				shown = append(shown, name)
			}
			return false
		}
		defer func() { ui.InLobby = saved }()
		call()
		if len(shown) == 0 {
			return "screens=-"
		}
		return "screens=" + strings.Join(shown, ",")
	}
	heldOf := func(p *player.Player) string {
		held, _ := p.HeldItems()
		if held.Empty() {
			return "held=-"
		}
		return fmt.Sprintf("held=%sx%d", naming.ItemKey(held.Item()), held.Count())
	}
	// itemsOnGround counts the item entities in the world.
	itemsOnGround := func(s *session, p *player.Player, tx *world.Tx) string {
		n := 0
		for e := range tx.Entities() {
			if e.H().Type() == entity.ItemType {
				n++
			}
		}
		return fmt.Sprintf("dropped=%d", n)
	}

	// event runs the click inside one traced transaction. prior is how many
	// untraced uses come first; between (when set by repeatWith) puts the
	// scene back after them, so that a use which was not debounced would
	// show; forget clears the debounce (the screens' and the stick's) before
	// the traced use; extra reads effects afterwards (it may open its own
	// transactions).
	var between setupFn
	repeatWith := func(f setupFn, record func()) {
		between = f
		record()
		between = nil
	}
	event := func(name string, sc scene, prior int, forget bool, call func(hd *handler, ctx *player.Context, p *player.Player), extra func() string) {
		fire := func() *player.Context {
			var ctx *player.Context
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				ctx = player.NewEventContext(tx, p)
				call(sc.s.Handler(p).(*handler), ctx, p)
			})
			return ctx
		}
		for range prior {
			fire()
		}
		if between != nil {
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) { between(sc.s, p, tx) })
		}
		if forget {
			sc.s.ForgetOpens()
			delete(handwork.LastStick.Of(sc.s.Session), sc.h)
		}
		var ctx *player.Context
		scr := ""
		pinRun(t, lines, name, func() string {
			out := ""
			if extra != nil {
				out = extra()
			}
			return join(fmt.Sprintf("cancelled=%t", ctx.Cancelled()), scr, out)
		}, func() {
			scr = asked(func() { ctx = fire() })
		})
	}
	// effects reads f on the player in a transaction after the call.
	effects := func(sc scene, f func(s *session, p *player.Player, tx *world.Tx) string) func() string {
		return func() string {
			out := ""
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) { out = f(sc.s, p, tx) })
			return out
		}
	}
	held := func(s *session, p *player.Player, tx *world.Tx) string { return heldOf(p) }
	blockIs := func(at cube.Pos) func(s *session, p *player.Player, tx *world.Tx) string {
		return func(s *session, p *player.Player, tx *world.Tx) string { return fmt.Sprintf("block=%T", tx.Block(at)) }
	}
	both := func(fs ...func(s *session, p *player.Player, tx *world.Tx) string) func(s *session, p *player.Player, tx *world.Tx) string {
		return func(s *session, p *player.Player, tx *world.Tx) string {
			var out []string
			for _, f := range fs {
				out = append(out, f(s, p, tx))
			}
			return join(out...)
		}
	}

	// UseOnBlock.
	clickAt := func(at cube.Pos, face cube.Face, click mgl64.Vec3) func(hd *handler, ctx *player.Context, p *player.Player) {
		return func(hd *handler, ctx *player.Context, p *player.Player) {
			hd.HandleItemUseOnBlock(ctx, at, face, click)
		}
	}
	onBlock := func(name string, setup setupFn, at cube.Pos, face cube.Face, click mgl64.Vec3, extra func(s *session, p *player.Player, tx *world.Tx) string) scene {
		sc := start(setup)
		var ex func() string
		if extra != nil {
			ex = effects(sc, extra)
		}
		event("UseOnBlock/"+name, sc, 0, false, clickAt(at, face, click), ex)
		return sc
	}
	// onBlockTwice records the repeat (and, with window, the use after the
	// debounce window) of the same click.
	onBlockTwice := func(name string, setup setupFn, at cube.Pos, face cube.Face, click mgl64.Vec3, window bool, extra func(s *session, p *player.Player, tx *world.Tx) string) {
		sc := start(setup)
		var ex func() string
		if extra != nil {
			ex = effects(sc, extra)
		}
		event("UseOnBlock/"+name, sc, 1, window, clickAt(at, face, click), ex)
	}

	stone := item.NewStack(block.Stone{}, 3)
	menuItem := screens.MenuItem("ko")
	dimStone := item.NewStack(sgitem.DimensionStone{}, 1)
	copierItem := item.NewStack(sgitem.ConfigCopier{}, 1)
	wandItem := item.NewStack(sgitem.NewWand(sgitem.WandStone), 1)
	stick := item.NewStack(item.Stick{}, 1)
	ironHammer := item.NewStack(sgitem.Hammer{}, 1)
	netMiner := item.NewStack(machine.New(machine.KindMiner), 1)

	onBlock("plain", holding(stone), pos, cube.FaceUp, centre, held)
	// 1 dimension.hunting
	onBlock("hunting", hunting, pos, cube.FaceUp, centre, nil)
	onBlock("huntingMenuItem", chain(hunting, holding(menuItem)), pos, cube.FaceUp, centre, nil)
	// 2 screens.menuItem
	onBlock("menuItem", holding(menuItem), pos, cube.FaceUp, centre, nil)
	onBlockTwice("menuItemRepeat", holding(menuItem), pos, cube.FaceUp, centre, false, nil)
	onBlockTwice("menuItemAfterWindow", holding(menuItem), pos, cube.FaceUp, centre, true, nil)
	onBlock("menuItemOutside", holding(menuItem), outsidePos, cube.FaceUp, centre, nil)
	onBlock("menuItemExpanding", chain(holding(menuItem), expanding), pos, cube.FaceUp, centre, nil)
	// 3 dimension.stone
	onBlock("stone", holding(dimStone), pos, cube.FaceUp, centre, nil)
	onBlockTwice("stoneRepeat", holding(dimStone), pos, cube.FaceUp, centre, false, nil)
	onBlockTwice("stoneAfterWindow", holding(dimStone), pos, cube.FaceUp, centre, true, nil)
	onBlock("stoneExpanding", chain(holding(dimStone), expanding), pos, cube.FaceUp, centre, nil)
	onBlock("stoneOutside", holding(dimStone), outsidePos, cube.FaceUp, centre, nil)
	// 4 expand.guard, 5 island.guard
	onBlock("expanding", chain(holding(stone), expanding), pos, cube.FaceUp, centre, held)
	onBlock("expandingCopier", chain(holding(copierItem), expanding), pos, cube.FaceUp, centre, held)
	onBlock("outside", holding(stone), outsidePos, cube.FaceUp, centre, held)
	onBlock("arena", holding(stone), core.ArenaCentre, cube.FaceUp, centre, held)
	onBlock("outsideCopier", holding(copierItem), outsidePos, cube.FaceUp, centre, held)
	// 6 machine.noteFace: the face a miner or network block is placed by.
	placedFacing := func(sc scene) func() string {
		return func() string {
			at := cube.Pos{6, island.IslandY + 2, 6}
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				sc.s.Handler(p).HandleBlockPlace(player.NewEventContext(tx, p), at, machine.New(machine.KindMiner))
			})
			out := ""
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				m, ok := tx.Block(at).(machine.Machine)
				if !ok || m.State == nil {
					out = "placed=-"
					return
				}
				out = fmt.Sprintf("placedFacing=%v", m.State.Facing)
			})
			return out
		}
	}
	for _, c := range []struct {
		name string
		held item.Stack
		face cube.Face
	}{
		{"heldMiner", netMiner, cube.FaceEast},
		{"heldNetCable", item.NewStack(machine.New(machine.KindNetCable), 1), cube.FaceEast},
		{"heldCrusher", item.NewStack(machine.New(machine.KindCrusher), 1), cube.FaceEast},
		{"heldStone", stone, cube.FaceEast},
	} {
		sc := start(holding(c.held))
		event("UseOnBlock/"+c.name, sc, 0, false, clickAt(pos, c.face, centre), placedFacing(sc))
	}
	// The denied clicks note no face.
	for _, c := range []struct {
		name  string
		setup setupFn
		at    cube.Pos
	}{
		{"heldMinerOutside", holding(netMiner), outsidePos},
	} {
		sc := start(c.setup)
		event("UseOnBlock/"+c.name, sc, 0, false, clickAt(c.at, cube.FaceEast, centre), placedFacing(sc))
	}
	// 7 copier.use
	bundlePos := pos.Add(cube.Pos{1, 0, 0})
	withBundle := chain(func(s *session, p *player.Player, tx *world.Tx) {
		if got := chestWithBundle(tx, pos, itemnet.Input, 5, 9); got != bundlePos {
			t.Fatalf("bundle at %v", got)
		}
	}, holding(copierItem))
	cardOf := func(s *session, p *player.Player, tx *world.Tx) string {
		held, _ := p.HeldItems()
		if c, ok := copier.CardOf(held); ok {
			return "card=" + c.Kind
		}
		return "card=-"
	}
	onBlock("copier", withBundle, bundlePos, cube.FaceWest, centre, cardOf)
	repeatWith(holding(copierItem), func() {
		onBlockTwice("copierRepeat", withBundle, bundlePos, cube.FaceWest, centre, false, cardOf)
		onBlockTwice("copierAfterWindow", withBundle, bundlePos, cube.FaceWest, centre, true, cardOf)
	})
	onBlock("copierNothing", chain(blockAt(pos, block.Stone{}), holding(copierItem)), pos, cube.FaceUp, centre, cardOf)
	onBlock("copierSneaking", chain(withBundle, sneaking), bundlePos, cube.FaceWest, centre, cardOf)
	// 8 wand.use
	wandWallSetup := func(s *session, p *player.Player, tx *world.Tx) {
		wandWall(tx)
		p.SetHeldItems(wandItem, item.Stack{})
		_, _ = p.Inventory().AddItem(item.NewStack(block.Stone{}, 20))
	}
	wandEffect := func(s *session, p *player.Player, tx *world.Tx) string {
		h, _ := p.HeldItems()
		front := 0
		for x := 3; x < 6; x++ {
			for y := island.IslandY + 1; y < island.IslandY+4; y++ {
				if _, ok := tx.Block(cube.Pos{x, y, 7}).(block.Stone); ok {
					front++
				}
			}
		}
		return fmt.Sprintf("stone=%d wear=%d front=%d", countStone(p), h.MaxDurability()-h.Durability(), front)
	}
	clearFront := func(s *session, p *player.Player, tx *world.Tx) {
		for x := 3; x < 6; x++ {
			for y := island.IslandY + 1; y < island.IslandY+4; y++ {
				tx.SetBlock(cube.Pos{x, y, 7}, block.Air{}, nil)
			}
		}
	}
	wallMid := cube.Pos{4, island.IslandY + 2, 8}
	onBlock("wand", wandWallSetup, wallMid, cube.FaceNorth, centre, wandEffect)
	repeatWith(clearFront, func() {
		onBlockTwice("wandRepeat", wandWallSetup, wallMid, cube.FaceNorth, centre, false, wandEffect)
		onBlockTwice("wandAfterWindow", wandWallSetup, wallMid, cube.FaceNorth, centre, true, wandEffect)
	})
	onBlock("expandingWand", chain(wandWallSetup, expanding), wallMid, cube.FaceNorth, centre, wandEffect)
	onBlock("wandSneaking", chain(wandWallSetup, sneaking), wallMid, cube.FaceNorth, centre, wandEffect)
	// 9 storage.sneakUse: sneaking, a drawer, a held item that is not a block.
	drawerAt := blockAt(pos, storage.Drawer{Facing: cube.South})
	onBlock("drawerSneak", chain(drawerAt, holding(stick), sneaking), pos, cube.FaceUp, centre, held)
	onBlockTwice("drawerSneakRepeat", chain(drawerAt, holding(stick), sneaking), pos, cube.FaceUp, centre, false, held)
	onBlockTwice("drawerSneakAfterWindow", chain(drawerAt, holding(stick), sneaking), pos, cube.FaceUp, centre, true, held)
	onBlock("drawerStanding", chain(drawerAt, holding(stick)), pos, cube.FaceUp, centre, held)
	onBlock("drawerSneakBlock", chain(drawerAt, holding(stone), sneaking), pos, cube.FaceUp, centre, held)
	onBlock("drawerSneakEmpty", chain(drawerAt, sneaking), pos, cube.FaceUp, centre, held)
	onBlock("drawerSneakNotDrawer", chain(blockAt(pos, block.Stone{}), holding(stick), sneaking), pos, cube.FaceUp, centre, held)
	// 10 conduit.plate
	plateClick := mgl64.Vec3{1, 0.5, 0.5}
	plate := func(s *session, p *player.Player, tx *world.Tx) {
		chestWithBundle(tx, pos, itemnet.Input, 5, 9)
	}
	onBlock("plate", plate, pos, cube.FaceEast, plateClick, held)
	onBlockTwice("plateRepeat", plate, pos, cube.FaceEast, plateClick, false, held)
	onBlockTwice("plateAfterWindow", plate, pos, cube.FaceEast, plateClick, true, held)
	onBlock("plateWithPickaxe", chain(plate, holding(item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1))), pos, cube.FaceEast, plateClick, held)
	onBlock("plateHeldFits", chain(plate, holding(item.NewStack(conduit.EnergyConduit{}, 1))), pos, cube.FaceEast, plateClick, held)
	onBlock("plateSneaking", chain(plate, sneaking), pos, cube.FaceEast, plateClick, held)
	onBlock("plateOffCentre", plate, pos, cube.FaceEast, mgl64.Vec3{1, 0.95, 0.5}, held)
	onBlock("plateNoBundle", blockAt(pos, block.NewChest()), pos, cube.FaceEast, plateClick, held)
	// 11 handwork.stick
	chestStocked := func(s *session, p *player.Player, tx *world.Tx) {
		tx.SetBlock(pos, block.NewChest(), nil)
		chest := tx.Block(pos).(block.Chest).Inventory(tx, pos)
		_ = chest.SetItem(0, item.NewStack(block.Dirt{}, 1))
		_ = p.Inventory().SetItem(14, item.NewStack(block.Dirt{}, 10))
	}
	slot14 := func(s *session, p *player.Player, tx *world.Tx) string {
		it, _ := p.Inventory().Item(14)
		return fmt.Sprintf("slot14=%d", it.Count())
	}
	onBlock("stickChest", chain(chestStocked, holding(stick), sneaking), pos, cube.FaceUp, centre, slot14)
	restock := slot(14, item.NewStack(block.Dirt{}, 10))
	repeatWith(restock, func() {
		onBlockTwice("stickChestRepeat", chain(chestStocked, holding(stick), sneaking), pos, cube.FaceUp, centre, false, slot14)
		onBlockTwice("stickChestAfterWindow", chain(chestStocked, holding(stick), sneaking), pos, cube.FaceUp, centre, true, slot14)
	})
	onBlock("stickChestStanding", chain(chestStocked, holding(stick)), pos, cube.FaceUp, centre, both(slot14, itemsOnGround))
	onBlock("stickGrass", chain(blockAt(pos, block.Grass{}), holding(stick)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlock("stickDirt", chain(blockAt(pos, block.Dirt{}), holding(stick)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlockTwice("stickGrassRepeat", chain(blockAt(pos, block.Grass{}), holding(stick)), pos, cube.FaceUp, centre, false, itemsOnGround)
	onBlockTwice("stickGrassAfterWindow", chain(blockAt(pos, block.Grass{}), holding(stick)), pos, cube.FaceUp, centre, true, itemsOnGround)
	onBlock("stickStone", chain(blockAt(pos, block.Stone{}), holding(stick)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlock("stickSneakingGrass", chain(blockAt(pos, block.Grass{}), holding(stick), sneaking), pos, cube.FaceUp, centre, itemsOnGround)
	// 12 farm.composter
	fullComposter := chain(blockAt(pos, block.Composter{Level: farm.ComposterLevels}), func(s *session, p *player.Player, tx *world.Tx) {
		farm.SetComposterCount(s.Session, pos, farm.MatterCapacity)
	})
	dirtHeld := func(s *session, p *player.Player, tx *world.Tx) string {
		n := 0
		for _, st := range p.Inventory().Slots() {
			if _, ok := st.Item().(block.Dirt); ok {
				n += st.Count()
			}
		}
		return fmt.Sprintf("dirt=%d", n)
	}
	composterFacts := both(held, itemsOnGround, dirtHeld, func(s *session, p *player.Player, tx *world.Tx) string {
		return fmt.Sprintf("compost=%d", farm.StateOf(s.Session).Composters[farm.CropKey(pos)])
	})
	seeds := item.NewStack(block.WheatSeeds{}, 4)
	onBlock("composterFull", chain(fullComposter, holding(stone)), pos, cube.FaceUp, centre, composterFacts)
	onBlock("composterFullSneaking", chain(fullComposter, holding(stone), sneaking), pos, cube.FaceUp, centre, composterFacts)
	onBlock("composterFill", chain(blockAt(pos, block.Composter{}), holding(seeds)), pos, cube.FaceUp, centre, composterFacts)
	onBlock("composterFillSneaking", chain(blockAt(pos, block.Composter{}), holding(seeds), sneaking), pos, cube.FaceUp, centre, composterFacts)
	onBlock("composterNoMatter", chain(blockAt(pos, block.Composter{}), holding(stone)), pos, cube.FaceUp, centre, composterFacts)
	// 13 handwork.gather: not a stop; a bark stripper on a log, a hoe on dirt.
	var fiberSeed uint64
	for fiberSeed = 0; ; fiberSeed++ {
		if rand.New(rand.NewPCG(fiberSeed, 2)).IntN(100) < handwork.FiberChance {
			break
		}
	}
	reseed := func(s *session, p *player.Player, tx *world.Tx) { s.Rand = rand.New(rand.NewPCG(fiberSeed, 2)) }
	axe := item.NewStack(item.Axe{Tier: item.ToolTierIron}, 1)
	onBlock("fiber", chain(reseed, blockAt(pos, block.Log{Wood: block.OakWood()}), holding(axe)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlock("fiberStripped", chain(reseed, blockAt(pos, block.Log{Wood: block.OakWood(), Stripped: true}), holding(axe)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlock("fiberNotAxe", chain(reseed, blockAt(pos, block.Log{Wood: block.OakWood()}), holding(stone)), pos, cube.FaceUp, centre, itemsOnGround)
	hoe := item.NewStack(item.Hoe{Tier: item.ToolTierIron}, 1)
	var grassSeed uint64
	for grassSeed = 0; ; grassSeed++ {
		if rand.New(rand.NewPCG(grassSeed, 2)).IntN(100) < handwork.GrassSeedChance {
			break
		}
	}
	reseedGrass := func(s *session, p *player.Player, tx *world.Tx) { s.Rand = rand.New(rand.NewPCG(grassSeed, 2)) }
	onBlock("tillSeeds", chain(reseedGrass, blockAt(pos, block.Dirt{}), holding(hoe)), pos, cube.FaceUp, centre, itemsOnGround)
	onBlock("tillSeedsFaceDown", chain(reseedGrass, blockAt(pos, block.Dirt{}), holding(hoe)), pos, cube.FaceDown, centre, itemsOnGround)
	// 14 fairy.time
	cropAt := func(at cube.Pos, b world.Block) setupFn {
		return func(s *session, p *player.Player, tx *world.Tx) {
			tx.SetBlock(at.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
			tx.SetBlock(at, b, nil)
		}
	}
	young := cropAt(pos, block.WheatSeeds{})
	grown := cropAt(pos, farm.WithGrowth(block.WheatSeeds{}, 7))
	// A harvested crop drops a random number of items (the block's own roll,
	// not the session's), so only whether anything dropped is recorded.
	anyDropped := func(s *session, p *player.Player, tx *world.Tx) string {
		if itemsOnGround(s, p, tx) == "dropped=0" {
			return "dropped=none"
		}
		return "dropped=some"
	}
	cropFacts := both(held, anyDropped, blockIs(pos))
	boneMeal := item.NewStack(item.BoneMeal{}, 3)
	fairyMeal := item.NewStack(sgitem.FairyBoneMeal{}, 3)
	onBlock("fairyTime", chain(young, holding(fairyMeal)), pos, cube.FaceUp, centre, cropFacts)
	onBlock("fairyTimeOnStone", chain(blockAt(pos, block.Stone{}), holding(fairyMeal)), pos, cube.FaceUp, centre, cropFacts)
	onBlock("boneMealCropSneaking", chain(young, holding(boneMeal), sneaking), pos, cube.FaceUp, centre, cropFacts)
	onBlock("boneMealCrop", chain(young, holding(boneMeal)), pos, cube.FaceUp, centre, cropFacts)
	onBlock("boneMealStoneSneaking", chain(blockAt(pos, block.Stone{}), holding(boneMeal), sneaking), pos, cube.FaceUp, centre, cropFacts)
	onBlock("hourglassNoMachine", chain(blockAt(pos, block.Stone{}), holding(item.NewStack(sgitem.FairyHourglass{}, 1))), pos, cube.FaceUp, centre, cropFacts)
	// 15 farm.crop
	onBlock("crop", chain(young, holding(stone)), pos, cube.FaceUp, centre, cropFacts)
	onBlock("cropGrown", chain(grown, holding(stone)), pos, cube.FaceUp, centre, cropFacts)
	onBlock("cropSneaking", chain(young, holding(stone), sneaking), pos, cube.FaceUp, centre, cropFacts)
	// 16 expand.guardSide (never stops: expansion mode stopped at expand.guard),
	// 17 island.guardSide: the side of the face is another chunk.
	border := cube.Pos{15, island.IslandY + 2, 4}
	// The crop at the border is used before its side is checked.
	borderCrop := func(b world.Block) setupFn { return cropAt(border, b) }
	onBlock("cropGrownSideOutside", chain(borderCrop(farm.WithGrowth(block.WheatSeeds{}, 7)), holding(stone)), border, cube.FaceEast, centre, both(held, anyDropped, blockIs(border)))
	onBlock("cropSneakingSideOutside", chain(borderCrop(block.WheatSeeds{}), holding(stone), sneaking), border, cube.FaceEast, centre, both(held, anyDropped, blockIs(border)))
	onBlock("sideOutside", holding(stone), border, cube.FaceEast, centre, held)
	onBlock("sideOwned", holding(stone), border, cube.FaceWest, centre, held)

	// Use.
	onUse := func(name string, setup setupFn, prior int, forget bool, extra func(s *session, p *player.Player, tx *world.Tx) string) {
		sc := start(setup)
		var ex func() string
		if extra != nil {
			ex = effects(sc, extra)
		}
		event("Use/"+name, sc, prior, forget, func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleItemUse(ctx) }, ex)
	}
	veinDiagonal := func(s *session, p *player.Player, tx *world.Tx) string {
		return fmt.Sprintf("diagonal=%t", account.On(p, tinker.VeinDiagonalSetting))
	}
	spare := slot(20, item.NewStack(block.Stone{}, 5))
	onUse("plain", holding(stone), 0, false, held)
	// 1 autofill.use: the last of a stack is refilled once the use is over.
	// The game takes the thrown item after the handler, unless it was
	// cancelled; the deferred refill then finds the hand empty.
	thrown := func(name string, setup setupFn) {
		sc := start(setup)
		event("Use/"+name, sc, 0, false, func(hd *handler, ctx *player.Context, p *player.Player) {
			hd.HandleItemUse(ctx)
			if !ctx.Cancelled() {
				h, off := p.HeldItems()
				p.SetHeldItems(h.Grow(-1), off)
			}
		}, effects(sc, held))
	}
	thrown("lastSnowball", chain(holding(item.NewStack(item.Snowball{}, 1)), slot(20, item.NewStack(item.Snowball{}, 5))))
	thrown("twoSnowballs", chain(holding(item.NewStack(item.Snowball{}, 2)), slot(20, item.NewStack(item.Snowball{}, 5))))
	thrown("lastStone", chain(holding(item.NewStack(block.Stone{}, 1)), spare))
	thrown("lastMenuItem", chain(holding(menuItem), slot(20, menuItem)))
	// 2 screens.menuItem
	onUse("menuItem", holding(menuItem), 0, false, held)
	onUse("menuItemRepeat", holding(menuItem), 1, false, held)
	onUse("menuItemAfterWindow", holding(menuItem), 1, true, held)
	onUse("menuItemHunting", chain(hunting, holding(menuItem)), 0, false, held)
	onUse("menuItemExpanding", chain(holding(menuItem), expanding), 0, false, held)
	watching := func() scene {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h, other := spawnPlayer(s), spawnPlayer(s)
		retire(s, h, other)
		withPlayer(t, s, other, func(p *player.Player, tx *world.Tx) { hunting(s, p, tx) })
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			dimension.Watch(s.Session, tx, p)
			if !dimension.Watching(s.Session, p) {
				t.Fatal("the player is not watching")
			}
			p.SetHeldItems(menuItem, item.Stack{})
		})
		return scene{s, h}
	}
	sw := watching()
	event("Use/menuItemWatching", sw, 0, false, func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleItemUse(ctx) }, effects(sw, held))
	// 3 dimension.hunting
	onUse("hunting", chain(hunting, holding(stone)), 0, false, held)
	onUse("huntingStone", chain(hunting, holding(dimStone)), 0, false, held)
	// 4 dimension.stone
	onUse("stone", holding(dimStone), 0, false, held)
	onUse("stoneRepeat", holding(dimStone), 1, false, held)
	onUse("stoneAfterWindow", holding(dimStone), 1, true, held)
	// 5 copier.clear, 6 wand.options, 7 tinker.veinToggle
	card := &copier.ConfigCard{Kind: copier.CardSide, Side: 4, Sides: []*copier.ConduitCard{{Mode: int(itemnet.Input), Extract: 5, Insert: 9}}}
	filled := copier.WithCard(copierItem, card, "ko")
	cardIn := func(s *session, p *player.Player, tx *world.Tx) string {
		held, _ := p.HeldItems()
		if _, ok := copier.CardOf(held); ok {
			return "card=held"
		}
		return "card=-"
	}
	onUse("copierFilled", holding(filled), 0, false, cardIn)
	repeatWith(holding(filled), func() {
		onUse("copierFilledRepeat", holding(filled), 1, false, cardIn)
		onUse("copierFilledAfterWindow", holding(filled), 1, true, cardIn)
	})
	onUse("copierEmpty", holding(copierItem), 0, false, cardIn)
	onUse("wandSneak", chain(holding(wandItem), sneaking), 0, false, held)
	onUse("wandStanding", holding(wandItem), 0, false, held)
	onUse("wandSneakRepeat", chain(holding(wandItem), sneaking), 1, false, held)
	onUse("wandSneakAfterWindow", chain(holding(wandItem), sneaking), 1, true, held)
	onUse("ironHammerSneak", chain(holding(ironHammer), sneaking), 0, false, both(held, veinDiagonal))
	onUse("ironHammerStanding", holding(ironHammer), 0, false, both(held, veinDiagonal))
	onUse("ironHammerSneakRepeat", chain(holding(ironHammer), sneaking), 1, false, both(held, veinDiagonal))
	onUse("ironHammerSneakAfterWindow", chain(holding(ironHammer), sneaking), 1, true, both(held, veinDiagonal))
	onUse("stoneHammerSneak", chain(holding(item.NewStack(sgitem.Hammer{Stone: true}, 1)), sneaking), 0, false, both(held, veinDiagonal))

	// UseOnEntity and AttackEntity.
	itemEntity := func(tx *world.Tx, p *player.Player) world.Entity {
		return tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{1, 0, 0})}, item.NewStack(block.Stone{}, 1)))
	}
	monster := func(tx *world.Tx, p *player.Player) world.Entity {
		h := world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{2, 0, 0})}.New(dimension.VariantType(sgitem.MobSlime, dimension.VariantBase),
			dimension.MonsterConfig{Mob: sgitem.MobSlime})
		return tx.AddEntity(h)
	}
	fairyOf := func(s *session) func(tx *world.Tx, p *player.Player) world.Entity {
		return func(tx *world.Tx, p *player.Player) world.Entity {
			if fairy.LocalOf(s.Session).Fairy == nil {
				t.Fatal("no fairy")
			}
			e, ok := fairy.LocalOf(s.Session).Fairy.Entity(tx)
			if !ok {
				t.Fatal("fairy entity is gone")
			}
			return e
		}
	}
	gave := func(s *session, p *player.Player, tx *world.Tx) string {
		if fairy.LocalOf(s.Session).Fairy == nil {
			return "gift=none"
		}
		e, ok := fairy.LocalOf(s.Session).Fairy.Entity(tx)
		if !ok {
			return "gift=gone"
		}
		return fmt.Sprintf("gift=%t", e.(*fairy.Fairy).Data.Data.(*fairy.FairyData).Gave)
	}
	// withFairy spawns a fairy for the player before the event.
	withFairy := func(setup setupFn) scene {
		sc := start(chain(setup, func(s *session, p *player.Player, tx *world.Tx) { fairy.Spawn(s.Session, tx, p) }))
		tick(sc.s, 2)
		return sc
	}
	entityEvent := func(prefix, name string, sc scene, prior int, target func(s *session) func(tx *world.Tx, p *player.Player) world.Entity, attack bool, extra func(s *session, p *player.Player, tx *world.Tx) string) {
		call := func(hd *handler, ctx *player.Context, p *player.Player) {
			e := target(sc.s)(p.Tx(), p)
			if attack {
				hd.HandleAttackEntity(ctx, e, nil, nil, nil)
			} else {
				hd.HandleItemUseOnEntity(ctx, e)
			}
		}
		var ex func() string
		if extra != nil {
			ex = effects(sc, extra)
		}
		event(prefix+"/"+name, sc, prior, false, call, ex)
	}
	fixed := func(f func(tx *world.Tx, p *player.Player) world.Entity) func(*session) func(tx *world.Tx, p *player.Player) world.Entity {
		return func(*session) func(tx *world.Tx, p *player.Player) world.Entity { return f }
	}
	for _, c := range []struct {
		prefix string
		attack bool
	}{{"UseOnEntity", false}, {"AttackEntity", true}} {
		entityEvent(c.prefix, "other", start(nil), 0, fixed(itemEntity), c.attack, held)
		entityEvent(c.prefix, "hunting", start(hunting), 0, fixed(itemEntity), c.attack, held)
		entityEvent(c.prefix, "monster", start(nil), 0, fixed(monster), c.attack, held)
		entityEvent(c.prefix, "fairy", withFairy(nil), 0, fairyOf, c.attack, gave)
		entityEvent(c.prefix, "fairyTwice", withFairy(nil), 1, fairyOf, c.attack, gave)
		entityEvent(c.prefix, "huntingFairy", withFairy(hunting), 0, fairyOf, c.attack, gave)
	}
	entityEvent("AttackEntity", "huntingMonster", start(hunting), 0, fixed(monster), true, held)

	// Every mark written in the four methods and useStone's callers is
	// reached, and the shapes of the rows are right.
	src, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	from, to := strings.Index(text, "func (h *handler) HandleItemUseOnBlock"), strings.Index(text, "func (h *handler) HandleMove")
	if from < 0 || to < from {
		t.Fatal("handler.go: the methods were not found")
	}
	var names []string
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`core\.Mark\("([^"]+)"\)`).FindAllStringSubmatch(text[from:to], -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	names = append(names, "expand.guard", "island.guard", "expand.guardSide", "island.guardSide")
	pinCheckReached(t, names, lines)

	rows := map[string]string{
		"UseOnBlock/plain": "dimension.hunting screens.menuItem dimension.stone expand.guard island.guard machine.noteFace copier.use wand.use storage.sneakUse conduit.plate handwork.stick farm.composter handwork.gather fairy.time farm.crop expand.guardSide island.guardSide",
		"Use/plain":        "autofill.use screens.menuItem dimension.hunting dimension.stone copier.clear wand.options tinker.veinToggle",
	}
	for name, row := range rows {
		if got := lines[name]; !strings.HasPrefix(got, row+" | ") {
			t.Errorf("%s: %q does not start with the row %q", name, got, row)
		}
	}
	for name, want := range map[string]string{
		"UseOnBlock/hunting":      "dimension.hunting | cancelled=true",
		"UseOnBlock/menuItem":     "dimension.hunting screens.menuItem | cancelled=true",
		"UseOnBlock/stone":        "dimension.hunting screens.menuItem dimension.stone | cancelled=true",
		"UseOnBlock/expanding":    "dimension.hunting screens.menuItem dimension.stone expand.guard | cancelled=true",
		"UseOnBlock/outside":      "dimension.hunting screens.menuItem dimension.stone expand.guard island.guard | cancelled=true",
		"UseOnBlock/cropSneaking": "farm.crop expand.guardSide island.guardSide",
		"Use/menuItemHunting":     "autofill.use screens.menuItem | cancelled=true",
		"AttackEntity/monster":    "dimension.hunter | cancelled=true",
	} {
		if got := lines[name]; !strings.Contains(got, want) {
			t.Errorf("%s: %q lacks %q", name, got, want)
		}
	}

	pinGolden(t, "testdata/pin/handler-use.golden", lines)
}
