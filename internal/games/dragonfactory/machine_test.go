package dragonfactory

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/itemnet"
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
)

var machinePos = cube.Pos{3, IslandY, 3}

// placeTestMachine places a machine and returns its state.
func placeTestMachine(s *session, pos cube.Pos, kind, owner string) *machineState {
	m := newMachine(kind, owner, owner)
	// These tests are about the work itself, not about energy (energy_test.go).
	m.st.unlimited = true
	if isSieve(kind) {
		// The tests sift with the string mesh fitted.
		m.st.mesh = "string"
	}
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
	return m.st
}

func outputCount(st *machineState, it world.Item) int {
	n := 0
	for _, slot := range st.kind.output {
		if got, _ := st.inv().Item(slot); !got.Empty() && itemKey(got.Item()) == itemKey(it) {
			n += got.Count()
		}
	}
	return n
}

func TestCrusher(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	if st.insert(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("crusher took dirt")
	}
	for range 2 {
		if !st.insert(item.NewStack(block.Cobblestone{}, 5)) {
			t.Fatal("crusher refused cobblestone")
		}
	}
	tick(s, machineKinds[KindCrusher].interval+1)
	if n := outputCount(st, block.Gravel{}); n != 1 {
		t.Fatalf("gravel after one interval = %d", n)
	}
	tick(s, machineKinds[KindCrusher].interval+1)
	if n := outputCount(st, block.Gravel{}); n != 2 {
		t.Fatalf("gravel after two intervals = %d", n)
	}
	if _, _, ok := st.firstInput(); ok {
		t.Fatal("input not consumed")
	}
	got, ok := st.takeOutput()
	if !ok || got.Count() != 1 {
		t.Fatalf("takeOutput = %v %v", got, ok)
	}
}

// TestAutoSieveKeepsProductsAndStopsWhenFull: the auto sieve puts its products
// into its four output slots, a hopper or conduit can take them out, and it stops
// (keeps its input) when they have no room.
func TestAutoSieveKeepsProductsAndStopsWhenFull(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindAutoSieve, "")
	if len(st.kind.output) != 4 {
		t.Fatalf("%d output slots, want 4", len(st.kind.output))
	}
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 64))
	tick(s, 64*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); !left.Empty() {
		t.Fatalf("gravel left: %v", left)
	}
	flint := outputCount(st, item.Flint{})
	if flint == 0 || flint == 64 {
		t.Fatalf("flint from 64 gravel = %d, want some but not all", flint)
	}
	// every output slot full of something it cannot stack with: it stops
	for i, slot := range st.kind.output {
		_ = st.inv().SetItem(slot, item.NewStack(block.Dirt{}, 64).WithValue("fill", i))
	}
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 5))
	tick(s, 5*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); left.Count() != 5 {
		t.Fatalf("a full auto sieve sifted: %v left", left)
	}
}

func TestHandSieveDoesNotSiftAlone(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSieve, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 1))
	tick(s, 3*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); left.Count() != 1 {
		t.Fatalf("hand sieve sifted alone: %v left", left)
	}
}

// TestAutoMinerKeepsItsOutput: the auto miner mines the block under it and
// keeps the result in its output slot; it pushes nothing into a chest beside
// or above it (hoppers and conduits carry items away).
func TestAutoMinerKeepsItsOutput(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	below, above := machinePos.Side(cube.FaceDown), machinePos.Side(cube.FaceUp)
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(below, block.Stone{}, nil)
		tx.SetBlock(above, block.NewChest(), nil)
	})
	st := placeTestMachine(s, machinePos, KindMiner, "")
	tick(s, machineKinds[KindMiner].interval+10)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(below).(block.Air); !air {
			t.Fatalf("stone not mined: %v", tx.Block(below))
		}
		if outputCount(st, block.Cobblestone{}) != 1 {
			t.Fatal("cobblestone not in the output slot")
		}
		chest := tx.Block(above).(block.Chest)
		if len(chest.Inventory(tx, above).Items()) != 0 {
			t.Fatal("the miner pushed into the chest")
		}
		// Special blocks are never mined.
		tx.SetBlock(below, DeliveryBox{}, nil)
	})
	tick(s, 2*machineKinds[KindMiner].interval)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(below).(DeliveryBox); !ok {
			t.Fatal("auto miner mined a delivery box")
		}
	})
}

func TestMachineWindowAndNBT(t *testing.T) {
	m := newMachine(KindCrusher, "a", "A")
	st := m.st
	h := st.inv().Handler()
	for _, c := range []struct {
		name   string
		f      func(*inventory.Context, int, item.Stack)
		slot   int
		it     item.Stack
		cancel bool
	}{
		{"place cobble in input", h.HandlePlace, 11, item.NewStack(block.Cobblestone{}, 1), false},
		{"place dirt in input", h.HandlePlace, 11, item.NewStack(block.Dirt{}, 1), true},
		{"place in output", h.HandlePlace, 15, item.NewStack(block.Cobblestone{}, 1), true},
		{"take output", h.HandleTake, 15, item.NewStack(block.Gravel{}, 1), false},
		{"take filler", h.HandleTake, 0, filler(), true},
		{"take indicator", h.HandleTake, 13, item.Stack{}, true},
	} {
		ctx := event.C(inventory.Holder(nil))
		c.f(ctx, c.slot, c.it)
		if ctx.Cancelled() != c.cancel {
			t.Errorf("%s: cancelled = %v", c.name, ctx.Cancelled())
		}
	}

	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 7))
	_ = st.inv().SetItem(15, item.NewStack(block.Gravel{}, 3))
	st.progress = 40
	back := Machine{Kind: KindCrusher}.DecodeNBT(m.EncodeNBT()).(Machine)
	if in, _ := back.st.inv().Item(11); in.Count() != 7 {
		t.Fatalf("input after NBT = %v", in)
	}
	if out, _ := back.st.inv().Item(15); out.Count() != 3 {
		t.Fatalf("output after NBT = %v", out)
	}
	if back.st.progress != 40 || back.st.owner != "a" {
		t.Fatalf("state after NBT = %d %q", back.st.progress, back.st.owner)
	}
	if drops := back.BreakInfo().Drops(nil, nil); len(drops) != 3 {
		t.Fatalf("drops = %v, want machine + input + output", drops)
	}
}

func TestMineable(t *testing.T) {
	for _, c := range []struct {
		b    world.Block
		want bool
	}{
		{block.Stone{}, true}, {block.Log{Wood: block.OakWood()}, true}, {block.CoalOre{}, true},
		{block.Air{}, false}, {block.Bedrock{}, false}, {Core{Type: "t1"}, false},
		{Machine{Kind: KindCrusher}, false}, {Drawer{}, false}, {block.NewChest(), false},
	} {
		if _, got := mineable(c.b); got != c.want {
			t.Errorf("mineable(%T) = %v, want %v", c.b, got, c.want)
		}
	}
}

// TestHandsFreeLine: an auto miner mines the block under it, an item
// conduit carries the cobblestone into a crusher beside it, and the crusher's
// output goes through a hopper into a chest.
func TestHandsFreeLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	minerPos := cube.Pos{0, IslandY + 2, 0}
	conduitPos := minerPos.Side(cube.FaceEast)
	crusherPos := conduitPos.Side(cube.FaceEast)
	hopperPos, chestPos := crusherPos.Side(cube.FaceDown), crusherPos.Add(cube.Pos{0, -2, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(minerPos.Side(cube.FaceDown), block.Cobblestone{}, nil)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(hopperPos, h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
		c := ConduitBundle{Item: 1}
		c.Modes[cube.FaceWest] = itemnet.Input
		c.Modes[cube.FaceEast] = itemnet.Output
		tx.SetBlock(conduitPos, c, nil)
	})
	placeTestMachine(s, minerPos, KindMiner, "")
	placeTestMachine(s, crusherPos, KindCrusher, "")
	runSteps(s, 120)
	s.world.Do(func(tx *world.Tx) {
		inv := tx.Block(chestPos).(block.Chest).Inventory(tx, chestPos)
		if !inv.ContainsItem(item.NewStack(block.Gravel{}, 1)) {
			t.Fatalf("crusher output never reached the chest: %v", inv.Items())
		}
	})
}

// TestCustomBlockBreakTimeIgnoresTools: the client breaks custom blocks in
// the bare-hand time of their definition, so the server must agree.
func TestCustomBlockBreakTimeIgnoresTools(t *testing.T) {
	pick := item.NewStack(item.Pickaxe{Tier: item.ToolTierDiamond}, 1)
	axe := item.NewStack(item.Axe{Tier: item.ToolTierDiamond}, 1)
	for _, b := range []world.Block{Machine{Kind: KindCrusher}, DeliveryBox{}, Drawer{}, Core{Type: "t1"}} {
		hand := block.BreakDuration(b, item.Stack{}, block.BreakContext{})
		for _, tool := range []item.Stack{pick, axe} {
			if d := block.BreakDuration(b, tool, block.BreakContext{}); d != hand {
				t.Errorf("%T: %v with %v, %v by hand", b, d, tool.Item(), hand)
			}
		}
		if hand > 3*time.Second {
			t.Errorf("%T takes %v to break", b, hand)
		}
	}
}

// TestNoMachineTextAboveBlock: machines show no floating text; their
// progress is in the machine window only.
func TestNoMachineTextAboveBlock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	st.insert(item.NewStack(block.Cobblestone{}, 5))
	tick(s, machineKinds[KindCrusher].interval/2+1)
	s.world.Do(func(tx *world.Tx) {
		if len(s.shapes) != 0 {
			t.Errorf("shapes shown above machines: %v", s.shapes)
		}
	})
}

// TestServerFinishesCustomBlockBreak: clients finish the crack of custom
// blocks without sending the destroy action, so the server breaks the block
// once its break time has passed while the player still mines it.
func TestServerFinishesCustomBlockBreak(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 1}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, DeliveryBox{}, nil)
		tx.SetBlock(cube.Pos{0, IslandY, 2}, block.Stone{}, nil)
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { p.Teleport(mgl64.Vec3{0.5, IslandY + 1, 2.5}) })
	tick(s, 2) // land
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { p.StartBreaking(pos, cube.FaceUp) })
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(pos).(DeliveryBox); !ok {
			t.Fatal("broke before its break time")
		}
	})
	time.Sleep(block.BreakDuration(DeliveryBox{}, item.Stack{}, block.BreakContext{}) + 200*time.Millisecond)
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(pos).(DeliveryBox); ok {
			t.Fatal("not broken after its break time")
		}
	})
}

// TestMinerLegacySlots: items saved in the auto miner's old output slots
// are kept and move into its one output slot.
func TestMinerLegacySlots(t *testing.T) {
	old := item.WriteNBT(item.NewStack(block.Cobblestone{}, 5), true)
	old["Slot"] = byte(14)
	m := Machine{Kind: KindMiner}.DecodeNBT(map[string]any{"Items": []any{old}}).(Machine)
	if len(m.st.legacy) != 1 {
		t.Fatalf("legacy %v, want the slot 14 stack", m.st.legacy)
	}
	saved := m.EncodeNBT()["Items"].([]any)
	again := Machine{Kind: KindMiner}.DecodeNBT(map[string]any{"Items": saved}).(Machine)
	if len(again.st.legacy) != 1 {
		t.Fatalf("legacy after a save %v", again.st.legacy)
	}
	again.st.drainLegacy()
	if it, _ := again.st.inv().Item(13); it.Count() != 5 || len(again.st.legacy) != 0 {
		t.Errorf("output %v legacy %v, want the stack moved into slot 13", it, again.st.legacy)
	}
}

// TestHopperFillsHandSieve: a hopper puts a block into a hand sieve with a
// mesh, one at a time and only while it is empty; without a mesh it takes
// nothing.
func TestHopperFillsHandSieve(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSieve, "")
	st.mesh = ""
	m := Machine{Kind: KindSieve, st: st}
	gravel := item.NewStack(block.Gravel{}, 5)
	s.world.Do(func(tx *world.Tx) {
		if n := m.InsertItems(tx, machinePos, cube.FaceUp, gravel); n != 0 {
			t.Errorf("a sieve without a mesh took %d", n)
		}
		st.mesh = "string"
		if n := m.InsertItems(tx, machinePos, cube.FaceUp, gravel); n != 1 {
			t.Errorf("hand sieve took %d, want 1", n)
		}
		if n := m.InsertItems(tx, machinePos, cube.FaceUp, gravel); n != 0 {
			t.Errorf("a full hand sieve took %d", n)
		}
		if n := m.InsertItems(tx, machinePos, cube.FaceDown, gravel); n != 0 {
			t.Errorf("the bottom face took %d", n)
		}
	})
}

// TestRingGaugesInTheWindows: the generator and the auto miner show their
// progress as a ring (CycleGauge), like the harvester.
func TestRingGaugesInTheWindows(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	for _, kind := range []string{KindGenerator, KindMiner, KindHarvester} {
		st := placeTestMachine(s, machinePos.Add(cube.Pos{3 * len(kind) % 7, 0, 0}), kind, "")
		st.progress = st.kind.interval / 2
		st.shownPct = -1
		st.showProgress()
		got, _ := st.inv().Item(st.kind.indicator)
		if _, ok := got.Item().(CycleGauge); !ok {
			t.Errorf("%s shows %T, want a ring gauge", kind, got.Item())
		}
	}
}

// TestEnergyGauge: machines with a window show a battery in the gauge slot,
// filled in tenths, whose hover name has the energy held and the most it holds.
func TestEnergyGauge(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	for _, kind := range []string{KindGenerator, KindCrusher, KindMiner, KindHarvester} {
		st := placeTestMachine(s, machinePos.Add(cube.Pos{3 * len(kind) % 7, 0, 0}), kind, "")
		st.unlimited = false
		st.energy = st.kind.energyCap / 2
		st.showEnergy()
		got, _ := st.inv().Item(energyGaugeSlot)
		g, ok := got.Item().(EnergyGauge)
		if !ok || g.Step != 5 {
			t.Fatalf("%s: slot %d holds %v", kind, energyGaugeSlot, got)
		}
		want := fmt.Sprintf("%d", st.energy)
		if name := got.CustomName(); !strings.Contains(name, want) || !strings.Contains(name, fmt.Sprint(st.kind.energyCap)) {
			t.Errorf("%s: hover name %q lacks %s and %d", kind, name, want, st.kind.energyCap)
		}
		if role := st.role(energyGaugeSlot); role != roleNone {
			t.Errorf("%s: the gauge slot is a %v slot", kind, role)
		}
	}
}

// TestAutoSieveWithManyPossibleDrops: a mesh whose table has more kinds of
// products than the four output slots still sifts (it only waits when the slots
// have no room for what it actually gave).
func TestAutoSieveWithManyPossibleDrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindAutoSieve, "")
	st.mesh = "iron"
	kinds := map[string]bool{}
	for _, d := range sieveTables[st.mesh][itemKey(block.Gravel{})] {
		kinds[itemKey(d.Item.Item())] = true
	}
	if len(kinds) <= len(st.kind.output) {
		t.Skipf("only %d kinds of products", len(kinds))
	}
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 16))
	tick(s, 16*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); left.Count() == 16 {
		t.Fatal("the auto sieve never started")
	}
}

// TestCrusherDoublesOre: an ore block crushes into two dusts of its metal, a
// chunk (a quarter of an ore) into one dust half of the time, and a dust smelts
// into an ingot (docs/design/07 §3).
func TestCrusherDoublesOre(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.IronOre{}, 2))
	tick(s, 2*(st.kind.interval+1))
	if n := outputCount(st, MetalDust{Metal: OreIron}); n != 4 {
		t.Fatalf("iron dust from two ore blocks = %d, want 4", n)
	}
	// chunks: over many, half of them give a dust (an ore's four chunks give two)
	crusherChunks := 0
	for range 400 {
		out := []item.Stack{}
		if _, chance, _ := oreDust(item.NewStack(OreChunk{Ore: MetalTin}, 1), crusherDust); s.rand.IntN(100) < chance {
			out = append(out, item.NewStack(MetalDust{Metal: MetalTin}, 1))
		}
		crusherChunks += len(out)
	}
	if crusherChunks < 160 || crusherChunks > 240 {
		t.Fatalf("400 chunks gave %d dusts, want about 200", crusherChunks)
	}
	for _, m := range dustMetals {
		d := MetalDust{Metal: m}
		info := d.SmeltInfo()
		if info.Product.Empty() || info.Product.Item() != ingotOf(m) {
			t.Errorf("%s dust smelts into %v", m, info.Product.Item())
		}
	}
}

// TestAlloySmelter: two ingredients in either slot make the alloy; a wrong pair
// does nothing; the output waits when it is full.
func TestAlloySmelter(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindAlloy, "")
	a, b := st.kind.input[0], st.kind.input[1]
	// copper in the second slot, tin in the first: bronze, whichever way round
	_ = st.inv().SetItem(a, item.NewStack(Ingot{Metal: MetalTin}, 2))
	_ = st.inv().SetItem(b, item.NewStack(item.CopperIngot{}, 6))
	tick(s, 2*(st.kind.interval+1))
	if n := outputCount(st, Ingot{Metal: MetalBronze}); n != 8 {
		t.Fatalf("bronze from 6 copper and 2 tin = %d, want 8", n)
	}
	// steel: one iron and four coal
	st2 := placeTestMachine(s, machinePos.Add(cube.Pos{2, 0, 0}), KindAlloy, "")
	_ = st2.inv().SetItem(a, item.NewStack(item.IronIngot{}, 1))
	_ = st2.inv().SetItem(b, item.NewStack(item.Coal{}, 3))
	tick(s, st2.kind.interval+2)
	if n := outputCount(st2, Ingot{Metal: MetalSteel}); n != 0 {
		t.Fatalf("steel from three coal = %d", n)
	}
	if !alloyAccepts(item.NewStack(item.Coal{}, 1)) || alloyAccepts(item.NewStack(block.Dirt{}, 1)) {
		t.Error("the smelter does not take exactly its ingredients")
	}
}

// TestEnergyCell: a cell is a source and a sink of energy that keeps what it
// gets.
func TestEnergyCell(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindEnergyCell, "")
	st.unlimited = false
	m := Machine{Kind: KindEnergyCell, st: st}
	src, snk := energy.RolesOf(m)
	if src == nil || snk == nil {
		t.Fatal("a cell is not both a source and a sink")
	}
	if got := m.ReceiveEnergy(nil, cube.Pos{}, cube.FaceUp, 500, false); got != 500 {
		t.Fatalf("received %d of 500", got)
	}
	if got := m.ReceiveEnergy(nil, cube.Pos{}, cube.FaceUp, 1<<30, false); got != cellCapacity-500 {
		t.Errorf("a cell took %d past its capacity", got)
	}
	if got := m.ExtractEnergy(nil, cube.Pos{}, cube.FaceUp, 200, false); got != 200 || st.energy != cellCapacity-200 {
		t.Errorf("gave %d, holds %d", got, st.energy)
	}
}

// TestMobSimulator: an essence charges the simulator for simCharges runs; each run
// rolls the drops of the mob into the output; the machine remembers its charge
// in its NBT.
func TestMobSimulator(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSimulator, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(Essence{Mob: MobZombie}, 2))
	tick(s, (st.kind.interval+1)*3)
	if st.mob != MobZombie || st.charges != simCharges-3 {
		t.Fatalf("mob %q charges %d after three runs", st.mob, st.charges)
	}
	if n := outputCount(st, item.RottenFlesh{}); n < 3 {
		t.Errorf("rotten flesh after three runs = %d", n)
	}
	if left, _ := st.inv().Item(st.kind.input[0]); left.Count() != 1 {
		t.Errorf("essence left: %v", left)
	}
	m := Machine{Kind: KindSimulator, st: st}
	back := m.DecodeNBT(m.EncodeNBT()).(Machine)
	if back.st.mob != MobZombie || back.st.charges != st.charges {
		t.Errorf("NBT round trip: %q %d", back.st.mob, back.st.charges)
	}
	for _, mob := range mobKinds {
		if len(simDrops[mob]) == 0 {
			t.Errorf("%s drops nothing", mob)
		}
	}
	if acceptsEssence(item.NewStack(block.Dirt{}, 1)) {
		t.Error("the simulator takes dirt")
	}
}

// TestAutoComposter: 32 plants make a dirt, as the composter does; it keeps
// them waiting when the output has no room.
func TestAutoComposter(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindAutoComposter, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(item.Wheat{}, 40))
	tick(s, 33*(st.kind.interval+1))
	if n := outputCount(st, block.Dirt{}); n != 1 {
		t.Fatalf("dirt from 33 plants = %d, want 1", n)
	}
	if !st.accepts(item.NewStack(item.Wheat{}, 1)) || st.accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Error("the auto composter takes the wrong items")
	}
}

// TestFluidLine: an electric crucible melts cobblestone into lava, a fluid
// conduit carries it to a lava generator, which burns it into energy.
func TestFluidLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	crucible := placeTestMachine(s, machinePos, KindCrucible, "")
	gen := placeTestMachine(s, machinePos.Add(cube.Pos{2, 0, 0}), KindLavaGenerator, "")
	gen.unlimited = false
	_ = crucible.inv().SetItem(crucible.kind.input[0], item.NewStack(block.Cobblestone{}, 4))
	tick(s, 4*(crucible.kind.interval+1))
	if crucible.fluid != fluidnet.Lava || crucible.fluidAmt != 4*crucibleLava {
		t.Fatalf("crucible holds %v %d, want %d lava", crucible.fluid, crucible.fluidAmt, 4*crucibleLava)
	}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(machinePos.Add(cube.Pos{1, 0, 0}), ConduitBundle{Fluid: 1}, nil)
		s.fluidGrid().AddConduit(machinePos.Add(cube.Pos{1, 0, 0}))
		if moved := s.fluidGrid().Step(tx); moved != 4*crucibleLava {
			t.Errorf("moved %d, want %d", moved, 4*crucibleLava)
		}
	})
	if gen.fluid != fluidnet.Lava || gen.fluidAmt != 4*crucibleLava || crucible.fluidAmt != 0 {
		t.Fatalf("generator %v %d, crucible %d", gen.fluid, gen.fluidAmt, crucible.fluidAmt)
	}
	tick(s, 20)
	if gen.energy != 20*lavaGenOutput {
		t.Errorf("energy %d after 20 ticks, want %d", gen.energy, 20*lavaGenOutput)
	}
	if gen.fluidAmt != 4*crucibleLava-10 {
		t.Errorf("lava %d, want %d (one mB every %d ticks)", gen.fluidAmt, 4*crucibleLava-10, lavaGenBurn)
	}
	// the generator takes no water, the tank takes both
	m := Machine{Kind: KindLavaGenerator, st: gen}
	if m.FillFluid(nil, cube.Pos{}, fluidnet.Water, 100, true) != 0 {
		t.Error("a lava generator takes water")
	}
	tank := Machine{Kind: KindTank, st: placeTestMachine(s, machinePos.Add(cube.Pos{0, 0, 2}), KindTank, "")}
	if tank.FillFluid(nil, cube.Pos{}, fluidnet.Water, 100, false) != 100 || tank.FillFluid(nil, cube.Pos{}, fluidnet.Lava, 100, true) != 0 {
		t.Error("a tank mixes fluids")
	}
}

// TestOreWasher: with water in its tank it washes an ore block into three dusts
// (a chunk into one dust three quarters of the time) and uses 250 mB; without
// water it does nothing.
func TestOreWasher(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindWasher, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.IronOre{}, 2))
	tick(s, 2*(st.kind.interval+1))
	if n := outputCount(st, MetalDust{Metal: OreIron}); n != 0 {
		t.Fatalf("dust without water = %d", n)
	}
	st.fluid, st.fluidAmt = fluidnet.Water, 500
	tick(s, 2*(st.kind.interval+1))
	if n := outputCount(st, MetalDust{Metal: OreIron}); n != 6 {
		t.Errorf("dust from two ore blocks = %d, want 6", n)
	}
	if st.fluidAmt != 0 || st.fluid != fluidnet.None {
		t.Errorf("tank %v %d", st.fluid, st.fluidAmt)
	}
}

// TestTankBuckets: a full bucket pours into a tank (and comes back empty), an empty
// bucket takes a bucket's worth out, a tank refuses the other fluid.
func TestTankBuckets(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	st := placeTestMachine(s, machinePos, KindTank, "")
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := tx.Block(machinePos).(Machine)
		use := func(held item.Stack) {
			p.SetHeldItems(held, item.Stack{})
			s.useFluidMachine(p, tx, machinePos, m, held, &item.UseContext{})
		}
		use(bucketOf(fluidnet.Lava))
		if st.fluid != fluidnet.Lava || st.fluidAmt != bucketMB {
			t.Fatalf("tank %v %d after a bucket of lava", st.fluid, st.fluidAmt)
		}
		if held, _ := p.HeldItems(); !held.Item().(item.Bucket).Empty() {
			t.Errorf("the bucket stayed full: %v", held)
		}
		use(bucketOf(fluidnet.Water))
		if st.fluid != fluidnet.Lava || st.fluidAmt != bucketMB {
			t.Errorf("the tank took water into lava: %v %d", st.fluid, st.fluidAmt)
		}
		use(item.NewStack(item.Bucket{}, 1))
		if st.fluidAmt != 0 {
			t.Errorf("tank still has %d", st.fluidAmt)
		}
		if held, _ := p.HeldItems(); fluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava {
			t.Errorf("did not get a bucket of lava: %v", held)
		}
	})
}
