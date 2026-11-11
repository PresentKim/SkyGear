package machine_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/storage"
)

func TestMachineIDsDistinct(t *testing.T) {
	seen := map[machine.ID]string{}
	for kind, k := range machine.Kinds {
		if k.ID == 0 {
			t.Errorf("%s has no id", kind)
		}
		if other, ok := seen[k.ID]; ok {
			t.Errorf("%s and %s share id %d", kind, other, k.ID)
		}
		seen[k.ID] = kind
	}
}

func TestCrusher(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindCrusher)
	if st.Insert(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("crusher took dirt")
	}
	for range 2 {
		if !st.Insert(item.NewStack(block.Cobblestone{}, 5)) {
			t.Fatal("crusher refused cobblestone")
		}
	}
	coretest.Tick(s, machine.Kinds[machine.KindCrusher].Interval+1)
	if n := machine.OutputCount(st, block.Gravel{}); n != 1 {
		t.Fatalf("gravel after one interval = %d", n)
	}
	coretest.Tick(s, machine.Kinds[machine.KindCrusher].Interval+1)
	if n := machine.OutputCount(st, block.Gravel{}); n != 2 {
		t.Fatalf("gravel after two intervals = %d", n)
	}
	if _, _, ok := st.FirstInput(); ok {
		t.Fatal("input not consumed")
	}
}

// TestAutoSieveKeepsProductsAndStopsWhenFull: the auto sieve puts its products
// into its four output slots, a hopper or conduit can take them out, and it stops
// (keeps its input) when they have no room.
func TestAutoSieveKeepsProductsAndStopsWhenFull(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindAutoSieve)
	if len(st.Def.Output) != 4 {
		t.Fatalf("%d output slots, want 4", len(st.Def.Output))
	}
	// gravel gives many kinds (ATM10 table): it sifts until its four slots
	// hold four kinds, so a hopper or conduit must carry them away
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.Gravel{}, 64))
	coretest.Tick(s, 64*(st.Def.Interval+1))
	left, _ := st.Inv().Item(st.Def.Input[0])
	if left.Count() == 64 {
		t.Fatal("the auto sieve sifted no gravel")
	}
	products := 0
	for _, slot := range st.Def.Output {
		if it, _ := st.Inv().Item(slot); !it.Empty() {
			products += it.Count()
		}
	}
	if products == 0 {
		t.Fatal("no products in the output slots")
	}
	// every output slot full of something it cannot stack with: it stops
	for i, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, 64).WithValue("fill", i))
	}
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.Gravel{}, 5))
	coretest.Tick(s, 5*(st.Def.Interval+1))
	if left, _ := st.Inv().Item(st.Def.Input[0]); left.Count() != 5 {
		t.Fatalf("a full auto sieve sifted: %v left", left)
	}
}

func TestHandSieveDoesNotSiftAlone(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindSieve)
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.Gravel{}, 1))
	coretest.Tick(s, 3*(st.Def.Interval+1))
	if left, _ := st.Inv().Item(st.Def.Input[0]); left.Count() != 1 {
		t.Fatalf("hand sieve sifted alone: %v left", left)
	}
}

// TestAutoMinerKeepsItsOutput: the auto miner mines the block under it and
// keeps the result in its output slot; it pushes nothing into a chest beside
// or above it (hoppers and conduits carry items away).
func TestAutoMinerKeepsItsOutput(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	below, above := machinetest.Pos.Side(cube.FaceDown), machinetest.Pos.Side(cube.FaceUp)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(below, block.Stone{}, nil)
		tx.SetBlock(above, block.NewChest(), nil)
	})
	st := machinetest.Place(s, machinetest.Pos, machine.KindMiner)
	coretest.Tick(s, machine.Kinds[machine.KindMiner].Interval+10)
	s.World.Do(func(tx *world.Tx) {
		if _, air := tx.Block(below).(block.Air); !air {
			t.Fatalf("stone not mined: %v", tx.Block(below))
		}
		if machine.OutputCount(st, block.Cobblestone{}) != 1 {
			t.Fatal("cobblestone not in the output slot")
		}
		chest := tx.Block(above).(block.Chest)
		if len(chest.Inventory(tx, above).Items()) != 0 {
			t.Fatal("the miner pushed into the chest")
		}
		// Special blocks are never mined.
		tx.SetBlock(below, storage.Drawer{Facing: cube.South}, nil)
	})
	coretest.Tick(s, 2*machine.Kinds[machine.KindMiner].Interval)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(below).(storage.Drawer); !ok {
			t.Fatal("auto miner mined a drawer")
		}
	})
}

func TestAutoMinerMinesTheBlockItFaces(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	west, below := machinetest.Pos.Side(cube.FaceWest), machinetest.Pos.Side(cube.FaceDown)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(west, block.Stone{}, nil)
		tx.SetBlock(below, block.Stone{}, nil)
	})
	st := machinetest.Place(s, machinetest.Pos, machine.KindMiner)
	st.Facing = cube.FaceWest
	coretest.Tick(s, machine.Kinds[machine.KindMiner].Interval+10)
	s.World.Do(func(tx *world.Tx) {
		if _, air := tx.Block(west).(block.Air); !air {
			t.Errorf("the faced stone was not mined: %v", tx.Block(west))
		}
		if _, stone := tx.Block(below).(block.Stone); !stone {
			t.Errorf("the stone below was mined: %v", tx.Block(below))
		}
	})
}

func TestMachineWindowAndNBT(t *testing.T) {
	m := machine.New(machine.KindCrusher)
	st := m.State
	h := st.Inv().Handler()
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
		{"take filler", h.HandleTake, 0, machine.Filler(), true},
		{"take indicator", h.HandleTake, 13, item.Stack{}, true},
	} {
		ctx := event.C(inventory.Holder(nil))
		c.f(ctx, c.slot, c.it)
		if ctx.Cancelled() != c.cancel {
			t.Errorf("%s: cancelled = %v", c.name, ctx.Cancelled())
		}
	}

	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 7))
	_ = st.Inv().SetItem(15, item.NewStack(block.Gravel{}, 3))
	st.Progress = 40
	back := machine.Machine{Kind: machine.KindCrusher}.DecodeNBT(m.EncodeNBT()).(machine.Machine)
	if in, _ := back.State.Inv().Item(11); in.Count() != 7 {
		t.Fatalf("input after NBT = %v", in)
	}
	if out, _ := back.State.Inv().Item(15); out.Count() != 3 {
		t.Fatalf("output after NBT = %v", out)
	}
	if back.State.Progress != 40 {
		t.Fatalf("state after NBT = %d", back.State.Progress)
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
		{block.Air{}, false}, {block.Bedrock{}, false},
		{machine.Machine{Kind: machine.KindCrusher}, false}, {storage.Drawer{}, false}, {block.NewChest(), false},
	} {
		if _, got := machine.Mineable(c.b); got != c.want {
			t.Errorf("mineable(%T) = %v, want %v", c.b, got, c.want)
		}
	}
}

// TestCustomBlockBareHandTime: the bare-hand time, the one the client's
// definition carries, stays short (minetool_test.go has the tools).
func TestCustomBlockBareHandTime(t *testing.T) {
	for _, b := range []world.Block{machine.Machine{Kind: machine.KindCrusher}, storage.Drawer{}, farm.Barrel{}} {
		if hand := block.BreakDuration(b, item.Stack{}, block.BreakContext{}); hand > 3*time.Second {
			t.Errorf("%T takes %v to break", b, hand)
		}
	}
}

// TestServerFinishesCustomBlockBreak: clients finish the crack of custom
// blocks without sending the destroy action, so the server breaks the block
// once its break time has passed while the player still mines it.
func TestServerFinishesCustomBlockBreak(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	pos := cube.Pos{0, island.IslandY, 1}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		tx.SetBlock(cube.Pos{0, island.IslandY, 2}, block.Stone{}, nil)
	})
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { p.Teleport(mgl64.Vec3{0.5, island.IslandY + 1, 2.5}) })
	coretest.Tick(s, 2) // land
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { p.StartBreaking(pos, cube.FaceUp) })
	coretest.Tick(s, 2)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(pos).(storage.Drawer); !ok {
			t.Fatal("broke before its break time")
		}
	})
	time.Sleep(block.BreakDuration(storage.Drawer{}, item.Stack{}, block.BreakContext{}) + 200*time.Millisecond)
	coretest.Tick(s, 2)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(pos).(storage.Drawer); ok {
			t.Fatal("not broken after its break time")
		}
	})
}

// TestServerFinishesCustomBlockBreakWithTool: the client only knows the
// bare-hand time of a custom block, so with its tool held the server breaks
// it once the tool's shorter time has passed.
func TestServerFinishesCustomBlockBreakWithTool(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	pos := cube.Pos{0, island.IslandY, 1}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
		tx.SetBlock(cube.Pos{0, island.IslandY, 2}, block.Stone{}, nil)
	})
	axe := item.NewStack(item.Axe{Tier: item.ToolTierIron}, 1)
	withTool := block.BreakDuration(storage.Drawer{}, axe, block.BreakContext{})
	hand := block.BreakDuration(storage.Drawer{}, item.Stack{}, block.BreakContext{})
	if withTool+400*time.Millisecond >= hand {
		t.Fatalf("an iron axe takes %v, the hand %v", withTool, hand)
	}
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Teleport(mgl64.Vec3{0.5, island.IslandY + 1, 2.5})
		p.SetHeldItems(axe, item.Stack{})
	})
	coretest.Tick(s, 2) // land
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { p.StartBreaking(pos, cube.FaceUp) })
	time.Sleep(withTool + 200*time.Millisecond)
	coretest.Tick(s, 2)
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(pos).(storage.Drawer); ok {
			t.Fatalf("not broken %v after starting with an iron axe (hand %v)", withTool+200*time.Millisecond, hand)
		}
	})
}

// TestMinerLegacySlots: items saved in the auto miner's old output slots
// are kept and move into its one output slot.
func TestMinerLegacySlots(t *testing.T) {
	old := item.WriteNBT(item.NewStack(block.Cobblestone{}, 5), true)
	old["Slot"] = byte(14)
	m := machine.Machine{Kind: machine.KindMiner}.DecodeNBT(map[string]any{"Items": []any{old}}).(machine.Machine)
	if len(m.State.Legacy) != 1 {
		t.Fatalf("legacy %v, want the slot 14 stack", m.State.Legacy)
	}
	saved := m.EncodeNBT()["Items"].([]any)
	again := machine.Machine{Kind: machine.KindMiner}.DecodeNBT(map[string]any{"Items": saved}).(machine.Machine)
	if len(again.State.Legacy) != 1 {
		t.Fatalf("legacy after a save %v", again.State.Legacy)
	}
	again.State.DrainLegacy()
	if it, _ := again.State.Inv().Item(13); it.Count() != 5 || len(again.State.Legacy) != 0 {
		t.Errorf("output %v legacy %v, want the stack moved into slot 13", it, again.State.Legacy)
	}
}

// TestHopperFillsHandSieve: a hopper puts a block into a hand sieve with a
// mesh, one at a time and only while it is empty; without a mesh it takes
// nothing.
func TestHopperFillsHandSieve(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindSieve)
	st.Mesh = ""
	m := machine.Machine{Kind: machine.KindSieve, State: st}
	gravel := item.NewStack(block.Gravel{}, 5)
	s.World.Do(func(tx *world.Tx) {
		if n := m.InsertItems(tx, machinetest.Pos, cube.FaceUp, gravel); n != 0 {
			t.Errorf("a sieve without a mesh took %d", n)
		}
		st.Mesh = "string"
		if n := m.InsertItems(tx, machinetest.Pos, cube.FaceUp, gravel); n != 1 {
			t.Errorf("hand sieve took %d, want 1", n)
		}
		if n := m.InsertItems(tx, machinetest.Pos, cube.FaceUp, gravel); n != 0 {
			t.Errorf("a full hand sieve took %d", n)
		}
		if n := m.InsertItems(tx, machinetest.Pos, cube.FaceDown, gravel); n != 0 {
			t.Errorf("the bottom face took %d", n)
		}
	})
}

// TestRingGaugesInTheWindows: the generator and the auto miner show their
// progress as a ring (CycleGauge), like the harvester.
func TestRingGaugesInTheWindows(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	for _, kind := range []string{machine.KindGenerator, machine.KindMiner, machine.KindHarvester} {
		st := machinetest.Place(s, machinetest.Pos.Add(cube.Pos{3 * len(kind) % 7, 0, 0}), kind)
		st.Progress = st.Def.Interval / 2
		st.ShownPct = -1
		st.ShowProgress()
		got, _ := st.Inv().Item(st.Def.Indicator)
		if _, ok := got.Item().(machine.CycleGauge); !ok {
			t.Errorf("%s shows %T, want a ring gauge", kind, got.Item())
		}
	}
}

// TestEnergyGauge: machines with a window show a battery in the gauge slot,
// filled in tenths, whose hover name has the energy held and the most it holds.
func TestEnergyGauge(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	for _, kind := range []string{machine.KindGenerator, machine.KindCrusher, machine.KindMiner, machine.KindHarvester} {
		st := machinetest.Place(s, machinetest.Pos.Add(cube.Pos{3 * len(kind) % 7, 0, 0}), kind)
		st.Unlimited = false
		st.Energy = st.Def.EnergyCap / 2
		st.ShowEnergy()
		got, _ := st.Inv().Item(machine.EnergyGaugeSlot)
		g, ok := got.Item().(machine.EnergyGauge)
		if !ok || g.Step != 5 {
			t.Fatalf("%s: slot %d holds %v", kind, machine.EnergyGaugeSlot, got)
		}
		want := fmt.Sprintf("%d", st.Energy)
		if name := got.CustomName(); !strings.Contains(name, want) || !strings.Contains(name, fmt.Sprint(st.Def.EnergyCap)) {
			t.Errorf("%s: hover name %q lacks %s and %d", kind, name, want, st.Def.EnergyCap)
		}
		if role := st.Role(machine.EnergyGaugeSlot); role != machine.RoleNone {
			t.Errorf("%s: the gauge slot is a %v slot", kind, role)
		}
	}
}

// TestAutoSieveWithManyPossibleDrops: a mesh whose table has more kinds of
// products than the four output slots still sifts (it only waits when the slots
// have no room for what it actually gave).
func TestAutoSieveWithManyPossibleDrops(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindAutoSieve)
	st.Mesh = "iron"
	kinds := map[string]bool{}
	for _, d := range sievetable.Tables[st.Mesh][naming.ItemKey(block.Gravel{})] {
		kinds[naming.ItemKey(d.Item.Item())] = true
	}
	if len(kinds) <= len(st.Def.Output) {
		t.Skipf("only %d kinds of products", len(kinds))
	}
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.Gravel{}, 16))
	coretest.Tick(s, 16*(st.Def.Interval+1))
	if left, _ := st.Inv().Item(st.Def.Input[0]); left.Count() == 16 {
		t.Fatal("the auto sieve never started")
	}
}

// TestCrusherOre: the crusher breaks an ore block into three raw ores and
// two more at half a chance each, takes no chunks, and raw ores and dusts
// smelt into ingots (ATM10).
func TestCrusherOre(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindCrusher)
	if st.Accepts(item.NewStack(sgitem.OreChunk{Ore: sgitem.OreIron}, 1)) {
		t.Fatal("the crusher takes ore chunks")
	}
	if out, _ := machine.CrusherOutput(item.NewStack(sgitem.OreBlock{Metal: sgitem.MetalTin}, 1)); out.Count() != sgitem.CrusherRaw || naming.ItemKey(out.Item()) != naming.ItemKey(sgitem.RawOre{Metal: sgitem.MetalTin}) {
		t.Fatalf("a tin ore crushes into %v", out)
	}
	if out, _ := machine.CrusherOutput(item.NewStack(block.IronOre{}, 1)); out.Count() != sgitem.CrusherRaw || naming.ItemKey(out.Item()) != naming.ItemKey(item.RawIron{}) {
		t.Fatalf("an iron ore crushes into %v", out)
	}
	// the bonus: 400 ores × 2 rolls at 50 % ≈ 400 more
	bonus := 0
	for range 400 * sgitem.CrusherBonus {
		if s.Rand.IntN(100) < sgitem.CrusherBonusChance {
			bonus++
		}
	}
	if bonus < 340 || bonus > 460 {
		t.Fatalf("bonus raw ores %d, want about 400", bonus)
	}
	for _, m := range sgitem.DustMetals {
		if info := (sgitem.MetalDust{Metal: m}).SmeltInfo(); info.Product.Empty() || naming.ItemKey(info.Product.Item()) != naming.ItemKey(sgitem.IngotOf(m)) {
			t.Errorf("%s dust smelts into %v", m, info.Product.Item())
		}
		if got, ok := machine.SmeltOf(item.NewStack(sgitem.RawOf(m), 1)); !ok || naming.ItemKey(got.Item()) != naming.ItemKey(sgitem.IngotOf(m)) {
			t.Errorf("raw %s smelts into %v", m, got)
		}
	}
}

// TestCrusherBonusRolls: a crushed ore block gives three to five raw ores.
func TestCrusherBonusRolls(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindCrusher)
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.IronOre{}, 1))
	coretest.Tick(s, st.Def.Interval+2)
	if n := machine.OutputCount(st, item.RawIron{}); n < sgitem.CrusherRaw || n > sgitem.CrusherRaw+sgitem.CrusherBonus {
		t.Fatalf("raw iron from one ore = %d, want 3 to 5", n)
	}
}

// TestAlloySmelter: two ingredients in either slot make the alloy; a wrong pair
// does nothing; the output waits when it is full.
func TestAlloySmelter(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindAlloy)
	a, b := st.Def.Input[0], st.Def.Input[1]
	// copper in the second slot, tin in the first: bronze, whichever way round
	_ = st.Inv().SetItem(a, item.NewStack(sgitem.Ingot{Metal: sgitem.MetalTin}, 2))
	_ = st.Inv().SetItem(b, item.NewStack(item.CopperIngot{}, 6))
	coretest.Tick(s, 2*(st.Def.Interval+1))
	if n := machine.OutputCount(st, sgitem.Ingot{Metal: sgitem.MetalBronze}); n != 8 {
		t.Fatalf("bronze from 6 copper and 2 tin = %d, want 8", n)
	}
	// steel: one iron and four coal
	st2 := machinetest.Place(s, machinetest.Pos.Add(cube.Pos{2, 0, 0}), machine.KindAlloy)
	_ = st2.Inv().SetItem(a, item.NewStack(item.IronIngot{}, 1))
	_ = st2.Inv().SetItem(b, item.NewStack(item.Coal{}, 3))
	coretest.Tick(s, st2.Def.Interval+2)
	if n := machine.OutputCount(st2, sgitem.Ingot{Metal: sgitem.MetalSteel}); n != 0 {
		t.Fatalf("steel from three coal = %d", n)
	}
	if !machinerecipe.AlloyAccepts(item.NewStack(item.Coal{}, 1)) || machinerecipe.AlloyAccepts(item.NewStack(block.Dirt{}, 1)) {
		t.Error("the smelter does not take exactly its ingredients")
	}
}

// TestEnergyCell: a cell is a source and a sink of energy that keeps what it
// gets.
func TestEnergyCell(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindEnergyCell)
	st.Unlimited = false
	m := machine.Machine{Kind: machine.KindEnergyCell, State: st}
	src, snk := energy.RolesOf(m)
	if src == nil || snk == nil {
		t.Fatal("a cell is not both a source and a sink")
	}
	if got := m.ReceiveEnergy(nil, cube.Pos{}, cube.FaceUp, 500, false); got != 500 {
		t.Fatalf("received %d of 500", got)
	}
	if got := m.ReceiveEnergy(nil, cube.Pos{}, cube.FaceUp, 1<<30, false); got != machine.CellCapacity-500 {
		t.Errorf("a cell took %d past its capacity", got)
	}
	if got := m.ExtractEnergy(nil, cube.Pos{}, cube.FaceUp, 200, false); got != 200 || st.Energy != machine.CellCapacity-200 {
		t.Errorf("gave %d, holds %d", got, st.Energy)
	}
}

// TestMobSimulator: an essence charges the simulator for sgitem.SimCharges runs; each run
// rolls the drops of the mob into the output; the machine remembers its charge
// in its NBT.
func TestMobSimulator(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindSimulator)
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 2))
	coretest.Tick(s, (st.Def.Interval+1)*3)
	if st.Mob != sgitem.MobZombie || st.Charges != sgitem.SimCharges-3 {
		t.Fatalf("mob %q charges %d after three runs", st.Mob, st.Charges)
	}
	if n := machine.OutputCount(st, item.RottenFlesh{}); n < 3 {
		t.Errorf("rotten flesh after three runs = %d", n)
	}
	if left, _ := st.Inv().Item(st.Def.Input[0]); left.Count() != 1 {
		t.Errorf("essence left: %v", left)
	}
	m := machine.Machine{Kind: machine.KindSimulator, State: st}
	back := m.DecodeNBT(m.EncodeNBT()).(machine.Machine)
	if back.State.Mob != sgitem.MobZombie || back.State.Charges != st.Charges {
		t.Errorf("NBT round trip: %q %d", back.State.Mob, back.State.Charges)
	}
	for _, mob := range sgitem.MobKinds {
		if len(sgitem.SimDrops[mob]) == 0 {
			t.Errorf("%s drops nothing", mob)
		}
	}
	if sgitem.AcceptsEssence(item.NewStack(block.Dirt{}, 1)) {
		t.Error("the simulator takes dirt")
	}
}

// TestFluidLine: an electric crucible melts cobblestone into lava, a fluid
// conduit carries it to a lava generator, which burns it into energy.
func TestFluidLine(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	crucible := machinetest.Place(s, machinetest.Pos, machine.KindCrucible)
	gen := machinetest.Place(s, machinetest.Pos.Add(cube.Pos{2, 0, 0}), machine.KindLavaGenerator)
	gen.Unlimited = false
	_ = crucible.Inv().SetItem(crucible.Def.Input[0], item.NewStack(block.Cobblestone{}, 4))
	coretest.Tick(s, 4*(crucible.Def.Interval+1))
	if crucible.Fluid != fluidnet.Lava || crucible.FluidAmt != 4*machine.CrucibleLava {
		t.Fatalf("crucible holds %v %d, want %d lava", crucible.Fluid, crucible.FluidAmt, 4*machine.CrucibleLava)
	}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(machinetest.Pos.Add(cube.Pos{1, 0, 0}), conduit.Bundle{Fluid: 1}, nil)
		if moved := network.Fluids(s.Session).Step(tx); moved != 4*machine.CrucibleLava {
			t.Errorf("moved %d, want %d", moved, 4*machine.CrucibleLava)
		}
	})
	if gen.Fluid != fluidnet.Lava || gen.FluidAmt != 4*machine.CrucibleLava || crucible.FluidAmt != 0 {
		t.Fatalf("generator %v %d, crucible %d", gen.Fluid, gen.FluidAmt, crucible.FluidAmt)
	}
	coretest.Tick(s, 20)
	if gen.Energy != 20*machine.LavaGenOutput {
		t.Errorf("energy %d after 20 ticks, want %d", gen.Energy, 20*machine.LavaGenOutput)
	}
	if gen.FluidAmt != 4*machine.CrucibleLava-10 {
		t.Errorf("lava %d, want %d (one mB every %d ticks)", gen.FluidAmt, 4*machine.CrucibleLava-10, machine.LavaGenBurn)
	}
	// the generator takes no water, the tank takes both
	m := machine.Machine{Kind: machine.KindLavaGenerator, State: gen}
	if m.FillFluid(nil, cube.Pos{}, fluidnet.Water, 100, true) != 0 {
		t.Error("a lava generator takes water")
	}
	tank := machine.Machine{Kind: machine.KindTank, State: machinetest.Place(s, machinetest.Pos.Add(cube.Pos{0, 0, 2}), machine.KindTank)}
	if tank.FillFluid(nil, cube.Pos{}, fluidnet.Water, 100, false) != 100 || tank.FillFluid(nil, cube.Pos{}, fluidnet.Lava, 100, true) != 0 {
		t.Error("a tank mixes fluids")
	}
}

// TestOreWasher: with water in its tank it washes a raw ore into two dusts
// and uses 250 mB; without water it does nothing; it takes no ore blocks.
func TestOreWasher(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindWasher)
	if st.Accepts(item.NewStack(block.IronOre{}, 1)) || st.Accepts(item.NewStack(sgitem.OreChunk{Ore: sgitem.OreIron}, 1)) {
		t.Fatal("the washer takes ore blocks or chunks")
	}
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(sgitem.RawOre{Metal: sgitem.MetalLead}, 2))
	coretest.Tick(s, 2*(st.Def.Interval+1))
	if n := machine.OutputCount(st, sgitem.MetalDust{Metal: sgitem.MetalLead}); n != 0 {
		t.Fatalf("dust without water = %d", n)
	}
	st.Fluid, st.FluidAmt = fluidnet.Water, 500
	coretest.Tick(s, 2*(st.Def.Interval+1))
	if n := machine.OutputCount(st, sgitem.MetalDust{Metal: sgitem.MetalLead}); n != 4 {
		t.Errorf("dust from two raw ores = %d, want 4", n)
	}
	if st.FluidAmt != 0 || st.Fluid != fluidnet.None {
		t.Errorf("tank %v %d", st.Fluid, st.FluidAmt)
	}
}

// TestTankBuckets: a full bucket pours into a tank (and comes back empty), an empty
// bucket takes a bucket's worth out, a tank refuses the other fluid.
func TestTankBuckets(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindTank)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := tx.Block(machinetest.Pos).(machine.Machine)
		use := func(held item.Stack) {
			p.SetHeldItems(held, item.Stack{})
			machine.UseFluid(s.Session, p, tx, machinetest.Pos, m, held, &item.UseContext{})
		}
		use(machine.BucketOf(fluidnet.Lava))
		if st.Fluid != fluidnet.Lava || st.FluidAmt != machine.BucketMB {
			t.Fatalf("tank %v %d after a bucket of lava", st.Fluid, st.FluidAmt)
		}
		if held, _ := p.HeldItems(); !held.Item().(item.Bucket).Empty() {
			t.Errorf("the bucket stayed full: %v", held)
		}
		use(machine.BucketOf(fluidnet.Water))
		if st.Fluid != fluidnet.Lava || st.FluidAmt != machine.BucketMB {
			t.Errorf("the tank took water into lava: %v %d", st.Fluid, st.FluidAmt)
		}
		use(item.NewStack(item.Bucket{}, 1))
		if st.FluidAmt != 0 {
			t.Errorf("tank still has %d", st.FluidAmt)
		}
		if held, _ := p.HeldItems(); machine.FluidOfBucket(held.Item().(item.Bucket)) != fluidnet.Lava {
			t.Errorf("did not get a bucket of lava: %v", held)
		}
	})
}

func TestMachineFillTakesOnlyWhatFits(t *testing.T) {
	st := machine.New(machine.KindCrusher).State
	in := st.Def.Input[0]
	if n := st.Fill(item.NewStack(item.Stick{}, 5)); n != 0 {
		t.Errorf("filled %d sticks the crusher does not take", n)
	}
	if n := st.Fill(item.NewStack(block.Cobblestone{}, 50)); n != 50 {
		t.Errorf("filled %d of 50 cobblestone", n)
	}
	if n := st.Fill(item.NewStack(block.Cobblestone{}, 30)); n != 14 {
		t.Errorf("filled %d cobblestone onto 50, want 14", n)
	}
	if got, _ := st.Inv().Item(in); got.Count() != 64 {
		t.Errorf("input holds %v", got)
	}
}

func TestQuickMoveIntoMachineGoesToInput(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	st := machine.New(machine.KindCrusher).State
	out, in := st.Def.Output[0], st.Def.Input[0]
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(0, item.NewStack(block.Cobblestone{}, 20))
		_ = p.Inventory().SetItem(1, item.NewStack(item.Stick{}, 5))
		for _, it := range []item.Stack{item.NewStack(block.Cobblestone{}, 20), item.NewStack(item.Stick{}, 5)} {
			ctx := event.C(inventory.Holder(p))
			st.Inv().Handler().HandlePlace(ctx, out, it)
			if !ctx.Cancelled() {
				t.Errorf("%v was placed on the output", it)
			}
		}
	})
	<-h.Do(func(*world.Tx, world.Entity) {}).Done()
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if got, _ := st.Inv().Item(in); got.Count() != 20 {
			t.Errorf("input holds %v, want the 20 cobblestone", got)
		}
		if p.Inventory().ContainsItem(item.NewStack(block.Cobblestone{}, 1)) {
			t.Error("the cobblestone stayed with the player as well")
		}
		if got, _ := p.Inventory().Item(1); got.Count() != 5 {
			t.Errorf("sticks changed: %v", got)
		}
	})
}

// TestMachineInfoWarnsWhenStalled: the info icon of a machine window carries
// the warning badge mark while the machine is stuck, and not otherwise.
func TestMachineInfoWarnsWhenStalled(t *testing.T) {
	st := machine.New(machine.KindCrusher).State
	if _, tier := slotdeco.Of(st.InfoItem().CustomName()); tier != 0 {
		t.Errorf("a working machine shows badge %d", tier)
	}
	st.Stuck = machine.StuckOutput
	if _, tier := slotdeco.Of(st.InfoItem().CustomName()); tier != slotdeco.BadgeWarn {
		t.Errorf("a stuck machine shows badge %d, want the warning", tier)
	}
	if !st.ShownStuck {
		t.Error("the window state was not recorded")
	}
}
