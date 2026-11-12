package skygear

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// TestPinBlocks records, per scenario, the marks that HandleStartBreak,
// HandleBlockBreak and HandleBlockPlace reach and the outcome (cancelled, for
// a break the final drops and xp, and the effects a scenario can read), into
// testdata/pin/handler-blocks.golden.
//
// Work the handler hands to Tx.Defer runs after the transaction callback, not
// where the Defer line is: the netstore step marks inside its deferred
// function (netstore.placeDeferred), so the golden shows it after every step
// that ran in the callback. A scenario runs the
// whole transaction inside the traced call, so deferred work is part of the
// line. Deferred work of the feature functions (machine.Place, PlaceDrawer,
// RefillHandLater) has no mark; the outcome reads its effect after the
// transaction (the block set, the hand refilled).
//
// Each scenario runs on a fresh session and player. The golden is recorded
// from the legacy code; it may be re-recorded (SKYGEAR_GOLDEN=update) only
// when the legacy behaviour itself is being changed on purpose.
func TestPinBlocks(t *testing.T) {
	lines := map[string]string{}
	pos := cube.Pos{4, island.IslandY + 2, 4}
	outsidePos := cube.Pos{84, island.IslandY + 2, 4}

	type scene struct {
		s *session
		h *world.EntityHandle
	}
	// start builds the session, runs setup and returns the scene.
	start := func(setup func(s *session, p *player.Player, tx *world.Tx)) scene {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			if setup != nil {
				setup(s, p, tx)
			}
		})
		return scene{s, h}
	}
	// read runs f on the player in a transaction after the traced call, so
	// everything the call deferred has run.
	read := func(sc scene, f func(s *session, p *player.Player, tx *world.Tx) string) string {
		out := ""
		if f != nil {
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) { out = f(sc.s, p, tx) })
		}
		return out
	}
	// event runs call inside one traced transaction.
	event := func(name string, sc scene, outcome func(ctx *player.Context) string, call func(hd *core.PlayerHandler, ctx *player.Context, p *player.Player)) {
		var ctx *player.Context
		pinRun(t, lines, name, func() string { return outcome(ctx) }, func() {
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				ctx = player.NewEventContext(tx, p)
				call(sc.s.Handler(p).(*core.PlayerHandler), ctx, p)
			})
		})
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
	blockName := func(b world.Block) string {
		if m, ok := b.(machine.Machine); ok {
			return "machine:" + m.Kind
		}
		return fmt.Sprintf("%T", b)
	}
	dropsOf := func(drops []item.Stack) string {
		if len(drops) == 0 {
			return "drops=-"
		}
		parts := make([]string, len(drops))
		for i, d := range drops {
			parts[i] = fmt.Sprintf("%sx%d", naming.ItemKey(d.Item()), d.Count())
		}
		return "drops=" + strings.Join(parts, ",")
	}
	heldOf := func(p *player.Player) string {
		held, _ := p.HeldItems()
		if held.Empty() {
			return "held=-"
		}
		return fmt.Sprintf("held=%sx%d", naming.ItemKey(held.Item()), held.Count())
	}
	holding := func(it item.Stack) func(s *session, p *player.Player, tx *world.Tx) {
		return func(s *session, p *player.Player, tx *world.Tx) { p.SetHeldItems(it, item.Stack{}) }
	}
	chain := func(fs ...func(s *session, p *player.Player, tx *world.Tx)) func(s *session, p *player.Player, tx *world.Tx) {
		return func(s *session, p *player.Player, tx *world.Tx) {
			for _, f := range fs {
				f(s, p, tx)
			}
		}
	}
	expanding := func(s *session, p *player.Player, tx *world.Tx) { expand.Show(s.Session, p) }
	blockAt := func(at cube.Pos, b world.Block) func(s *session, p *player.Player, tx *world.Tx) {
		return func(s *session, p *player.Player, tx *world.Tx) { tx.SetBlock(at, b, nil) }
	}
	cancelled := func(ctx *player.Context) string { return fmt.Sprintf("cancelled=%t", ctx.Cancelled()) }

	// StartBreak.
	startBreak := func(name string, at cube.Pos, setup func(s *session, p *player.Player, tx *world.Tx)) {
		event("StartBreak/"+name, start(setup), cancelled, func(hd *core.PlayerHandler, ctx *player.Context, p *player.Player) {
			hd.HandleStartBreak(ctx, at)
		})
	}
	startBreak("own", pos, nil)
	startBreak("outside", outsidePos, nil)
	startBreak("arena", core.ArenaCentre, nil)
	startBreak("expanding", pos, expanding)

	// BlockBreak: drops0 are the drops Dragonfly passes in, and the outcome
	// has the drops and xp the steps left, whatever a step changed.
	breakAt := func(name string, at cube.Pos, drops0 []item.Stack, setup func(s *session, p *player.Player, tx *world.Tx), after func(s *session, p *player.Player, tx *world.Tx) string) {
		sc := start(setup)
		drops := append([]item.Stack(nil), drops0...)
		xp := 0
		var ctx *player.Context
		pinRun(t, lines, "BlockBreak/"+name, func() string {
			return join(cancelled(ctx), dropsOf(drops), fmt.Sprintf("xp=%d", xp), read(sc, after))
		}, func() {
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				ctx = player.NewEventContext(tx, p)
				sc.s.Handler(p).HandleBlockBreak(ctx, at, &drops, &xp)
			})
		})
	}
	stoneDrop := []item.Stack{item.NewStack(block.Stone{}, 1)}
	hammer := item.NewStack(sgitem.Hammer{}, 1)
	fullComposter := chain(blockAt(pos, block.Composter{Level: farm.ComposterLevels}), func(s *session, p *player.Player, tx *world.Tx) {
		farm.SetComposterCount(s.Session, pos, farm.MatterCapacity)
	})
	clayBarrel := chain(blockAt(pos, farm.Barrel{Wood: "oak"}), func(s *session, p *player.Player, tx *world.Tx) {
		farm.StateOf(s.Session).Barrels = map[string]*farm.BarrelRecord{farm.CropKey(pos): {Clay: 3}}
	})
	punchDrawer := func(face cube.Face) func(s *session, p *player.Player, tx *world.Tx) {
		return chain(blockAt(pos, storage.Drawer{Facing: cube.South}), func(s *session, p *player.Player, tx *world.Tx) {
			tx.Block(pos).(storage.Drawer).Punch(pos, face, tx, p)
		})
	}
	// nodeConfig gives the node at pos a setting; a placed node forgets it.
	// A broken node forgets it through the block (blockbroken_test.go), which
	// the handler alone does not reach: the break scenario keeps it.
	nodeConfig := func(s *session, p *player.Player, tx *world.Tx) {
		netstore.Config(s.Session, pos).Filter = []string{"pin"}
	}
	configLeft := func(s *session, p *player.Player, tx *world.Tx) string {
		return fmt.Sprintf("filter=%d", len(netstore.Config(s.Session, pos).Filter))
	}
	toolHeld := func(it item.Stack) func(s *session, p *player.Player, tx *world.Tx) {
		return holding(it)
	}

	breakAt("stone", pos, stoneDrop, blockAt(pos, block.Stone{}), nil)
	breakAt("outside", outsidePos, stoneDrop, nil, nil)
	breakAt("expanding", pos, stoneDrop, expanding, nil)
	breakAt("hammerCobble", pos, []item.Stack{item.NewStack(block.Cobblestone{}, 1)}, chain(blockAt(pos, block.Cobblestone{}), holding(hammer)), nil)
	breakAt("hammerComposter", pos, stoneDrop, chain(fullComposter, holding(hammer)), nil)
	// The seed makes the crook's roll give a sapling. Birch leaves, because
	// oak and dark oak leaves also roll an apple from Dragonfly's global
	// random source, which a test cannot seed.
	crookSeed := func(s *session, p *player.Player, tx *world.Tx) { s.Rand = rand.New(rand.NewPCG(5, 2)) }
	breakAt("crookLeaves", pos, []item.Stack{item.NewStack(block.Leaves{Type: block.BirchLeaves()}, 1)},
		chain(crookSeed, blockAt(pos, block.Leaves{Type: block.BirchLeaves()}), holding(item.NewStack(sgitem.Crook{Tier: sgitem.CrookDiamond}, 1))), nil)
	breakAt("crookStone", pos, stoneDrop, chain(blockAt(pos, block.Stone{}), holding(item.NewStack(sgitem.Crook{Tier: sgitem.CrookDiamond}, 1))), nil)
	breakAt("tinkerTool", pos, stoneDrop, chain(blockAt(pos, block.Stone{}),
		toolHeld(item.NewStack(tinker.Tool{Kind: tinker.ToolPickaxe, Head: "iron", Handle: "wood"}, 1))),
		func(s *session, p *player.Player, tx *world.Tx) string {
			held, _ := p.HeldItems()
			return fmt.Sprintf("toolxp=%d", tinker.ToolXP(held))
		})
	breakAt("drawerFront", pos, stoneDrop, punchDrawer(cube.South.Face()), nil)
	breakAt("drawerSidePunch", pos, stoneDrop, punchDrawer(cube.North.Face()), nil)
	breakAt("drawer", pos, stoneDrop, blockAt(pos, storage.Drawer{Facing: cube.South}), nil)
	breakAt("machine", pos, stoneDrop, blockAt(pos, machine.New(machine.KindCrusher)), nil)
	breakAt("setMachine", pos, stoneDrop, blockAt(pos, machine.New(machine.KindDrawerController)), nil)
	breakAt("netNode", pos, stoneDrop, chain(blockAt(pos, machine.New(machine.KindNetCable)), nodeConfig), configLeft)
	breakAt("composter", pos, stoneDrop, fullComposter, nil)
	breakAt("barrel", pos, stoneDrop, clayBarrel, nil)
	breakAt("bundle", pos, stoneDrop, blockAt(pos, conduit.Bundle{Energy: 1}), nil)

	// BlockPlace: after the handler the game takes one block from the hand
	// (unless the event was cancelled), then the transaction ends and the
	// deferred work runs.
	placeAt := func(name string, at cube.Pos, b world.Block, setup func(s *session, p *player.Player, tx *world.Tx), after func(s *session, p *player.Player, tx *world.Tx) string) {
		sc := start(setup)
		var ctx *player.Context
		pinRun(t, lines, "BlockPlace/"+name, func() string {
			return join(cancelled(ctx), read(sc, func(s *session, p *player.Player, tx *world.Tx) string {
				out := ""
				if after != nil {
					out = after(s, p, tx)
				}
				return join(out, heldOf(p))
			}))
		}, func() {
			withPlayer(t, sc.s, sc.h, func(p *player.Player, tx *world.Tx) {
				ctx = player.NewEventContext(tx, p)
				sc.s.Handler(p).HandleBlockPlace(ctx, at, b)
				if !ctx.Cancelled() {
					held, off := p.HeldItems()
					p.SetHeldItems(held.Grow(-1), off)
				}
			})
		})
	}
	blockIs := func(s *session, p *player.Player, tx *world.Tx) string { return "block=" + blockName(tx.Block(pos)) }
	spare := func(it item.Stack, n int) func(s *session, p *player.Player, tx *world.Tx) {
		return func(s *session, p *player.Player, tx *world.Tx) {
			_ = p.Inventory().SetItem(20, item.NewStack(it.Item(), n))
		}
	}
	dirt := item.NewStack(block.Dirt{}, 2)
	machineItem := func(kind string, n int) item.Stack { return item.NewStack(machine.New(kind), n) }

	placeAt("dirt", pos, block.Dirt{}, holding(dirt), blockIs)
	placeAt("outside", outsidePos, block.Dirt{}, holding(dirt), nil)
	placeAt("expanding", pos, block.Dirt{}, chain(holding(dirt), expanding), nil)
	placeAt("lastOfStack", pos, block.Dirt{}, chain(holding(item.NewStack(block.Dirt{}, 1)), spare(dirt, 5)), nil)
	placeAt("crop", pos, block.BeetrootSeeds{}, holding(item.NewStack(block.BeetrootSeeds{}, 2)), func(s *session, p *player.Player, tx *world.Tx) string {
		if r := farm.StateOf(s.Session).Crops[farm.CropKey(pos)]; r != nil {
			return "crop=" + r.Kind
		}
		return "crop=-"
	})
	placeAt("machine", pos, machine.New(machine.KindCrusher), holding(machineItem(machine.KindCrusher, 1)), blockIs)
	placeAt("setMachine", pos, machine.New(machine.KindDrawerController), holding(machineItem(machine.KindDrawerController, 1)), blockIs)
	placeAt("netNode", pos, machine.New(machine.KindNetCable), chain(holding(machineItem(machine.KindNetCable, 2)), nodeConfig),
		func(s *session, p *player.Player, tx *world.Tx) string {
			return join(blockIs(s, p, tx), configLeft(s, p, tx))
		})
	placeAt("lastNetNode", pos, machine.New(machine.KindNetCable), chain(holding(machineItem(machine.KindNetCable, 1)), spare(machineItem(machine.KindNetCable, 1), 4), nodeConfig),
		func(s *session, p *player.Player, tx *world.Tx) string {
			return join(blockIs(s, p, tx), configLeft(s, p, tx))
		})
	placeAt("drawer", pos, storage.Drawer{}, holding(item.NewStack(storage.Drawer{}, 1)), blockIs)
	placeAt("bundle", pos, conduit.Bundle{Energy: 1}, nil, nil)
	placeAt("dragonEgg", pos, block.DragonEgg{}, holding(item.NewStack(block.DragonEgg{}, 1)), func(s *session, p *player.Player, tx *world.Tx) string {
		return fmt.Sprintf("dragon=%t", progress.Advanced(s.Session, advancement.Dragon))
	})

	// The shapes of the rows: a scenario that stops nowhere equals its hook's
	// lines of order.golden; the guards stop where they should. A run that
	// reorders the subscribers (TestTierOrder) has other traces.
	rows := map[string]string{
		"StartBreak/own":   "expand.guard island.guard tinker.autoTool",
		"BlockBreak/stone": "expand.guard island.guard storage.front tinker.break farm.break achievement.break",
		"BlockPlace/dirt":  "expand.guard island.guard machine.place storage.place farm.plant autofill.place netstore.place achievement.place progress.dragonEgg conduit.place",
	}
	for name, row := range rows {
		if got := lines[name]; hookOrder == "" && !strings.HasPrefix(got, row+" | ") {
			t.Errorf("%s: %q does not start with the row %q", name, got, row)
		}
	}
	for _, name := range []string{"StartBreak", "BlockBreak", "BlockPlace"} {
		if got, want := lines[name+"/outside"], "expand.guard island.guard | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
			t.Errorf("%s/outside: %q, want prefix %q", name, got, want)
		}
		if got, want := lines[name+"/expanding"], "expand.guard | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
			t.Errorf("%s/expanding: %q, want prefix %q", name, got, want)
		}
	}
	if got, want := lines["BlockBreak/drawerFront"], "expand.guard island.guard storage.front | cancelled=true"; hookOrder == "" && !strings.HasPrefix(got, want) {
		t.Errorf("BlockBreak/drawerFront: %q, want prefix %q", got, want)
	}
	if got := lines["BlockBreak/hammerCobble"]; !strings.Contains(got, "drops="+naming.ItemKey(block.Gravel{})+"x1") {
		t.Errorf("BlockBreak/hammerCobble: %q lacks the hammer's drop", got)
	}
	if got := lines["BlockPlace/lastOfStack"]; !strings.HasSuffix(got, "held="+naming.ItemKey(block.Dirt{})+"x5") {
		t.Errorf("BlockPlace/lastOfStack: %q, the hand was not refilled", got)
	}

	// Every subscriber of the three hooks and every mark of their deferred
	// work is reached.
	names := pinMarksIn(t, "netstore/subscribe.go", "package netstore")
	for _, sub := range core.Subscriptions() {
		switch sub.Hook {
		case "StartBreak", "BlockBreak", "BlockPlace":
			names = append(names, sub.Name)
		}
	}
	pinCheckReached(t, names, lines)

	pinGolden(t, "testdata/pin/handler-blocks.golden", lines)
}
