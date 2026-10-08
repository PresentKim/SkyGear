package dragonfactory

import (
	"testing"

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

func TestSieve(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSieve, "")
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 64))
	tick(s, 64*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); !left.Empty() {
		t.Fatalf("gravel left: %v", left)
	}
	flint := outputCount(st, item.Flint{})
	if flint == 0 || flint == 64 {
		t.Fatalf("flint from 64 gravel = %d, want some but not all", flint)
	}
}

func TestSieveWaitsWhenFull(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSieve, "")
	for _, slot := range st.kind.output {
		_ = st.inv().SetItem(slot, item.NewStack(item.Flint{}, 64))
	}
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 3))
	tick(s, 3*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); left.Count() != 3 {
		t.Fatalf("sieve worked with full outputs: %v left", left)
	}
}

func TestAutoMinerPushesIntoChest(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	below, above := machinePos.Side(cube.FaceDown), machinePos.Side(cube.FaceUp)
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(below, block.Stone{}, nil)
		tx.SetBlock(above, block.NewChest(), nil)
	})
	placeTestMachine(s, machinePos, KindMiner, "")
	tick(s, machineKinds[KindMiner].interval+10)
	s.world.Do(func(tx *world.Tx) {
		if _, air := tx.Block(below).(block.Air); !air {
			t.Fatalf("stone not mined: %v", tx.Block(below))
		}
		chest := tx.Block(above).(block.Chest)
		if !chest.Inventory(tx, above).ContainsItem(item.NewStack(block.Cobblestone{}, 1)) {
			t.Fatal("cobblestone not pushed into the chest")
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

func TestSellBoxPaysOfflineOwner(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	if _, err := s.g.db.TouchAccount("g", "", "g"); err != nil {
		t.Fatal(err)
	}
	_ = s.g.db.SetMoney(s.saveID, "g", 100)
	st := placeTestMachine(s, machinePos, KindSellBox, "g")
	if st.insert(item.NewStack(block.Dirt{}, 1)) == false {
		t.Fatal("sell box refused dirt")
	}
	if st.insert(item.NewStack(item.Apple{}, 1)) {
		t.Fatal("sell box took an item the shop does not buy")
	}
	_ = st.inv().SetItem(st.kind.input[1], item.NewStack(block.Cobblestone{}, 10))
	tick(s, st.kind.interval+2)
	if m, _ := s.g.db.Money(s.saveID, "g"); m != 100+1+10 {
		t.Fatalf("owner money = %d, want 111", m)
	}
	if _, _, ok := st.firstInput(); ok {
		t.Fatal("sold items left in the box")
	}
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
		{block.Air{}, false}, {block.Bedrock{}, false}, {Core{Type: "earth"}, false},
		{Machine{Kind: KindCrusher}, false}, {Drawer{}, false}, {block.NewChest(), false},
	} {
		if _, got := mineable(c.b); got != c.want {
			t.Errorf("mineable(%T) = %v, want %v", c.b, got, c.want)
		}
	}
}

// TestHandsFreeLine runs the M4 goal: core → auto miner → crusher / sell
// box → hopper → chest, without a player.
func TestHandsFreeLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	minerPos := startCorePos.Add(cube.Pos{0, 2, 0})
	crusherPos, sellPos := minerPos.Side(cube.FaceEast), minerPos.Side(cube.FaceWest)
	hopperPos, chestPos := crusherPos.Side(cube.FaceDown), crusherPos.Add(cube.Pos{0, -2, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(hopperPos, h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
	})
	placeTestMachine(s, minerPos, KindMiner, "")
	placeTestMachine(s, crusherPos, KindCrusher, "")
	placeTestMachine(s, sellPos, KindSellBox, s.hostID)

	tick(s, 20*60) // one minute
	bb := s.buyback()
	if bb.Budget == buybackBudget(s.state.Level) {
		t.Fatal("sell box sold nothing")
	}
	s.world.Do(func(tx *world.Tx) {
		inv := tx.Block(chestPos).(block.Chest).Inventory(tx, chestPos)
		if inv.Empty() {
			t.Fatal("crusher output never reached the chest")
		}
	})
	if m, _ := s.g.db.Money(s.saveID, s.hostID); m <= 0 {
		t.Fatalf("host money = %d", m)
	}
}
