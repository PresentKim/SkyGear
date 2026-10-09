package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
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

func TestAutoSieveFillsChestBeside(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindAutoSieve, "")
	chest := block.NewChest()
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(machinePos.Side(cube.FaceEast), chest, nil) })
	_ = st.inv().SetItem(st.kind.input[0], item.NewStack(block.Gravel{}, 64))
	tick(s, 64*(st.kind.interval+1))
	if left, _ := st.inv().Item(st.kind.input[0]); !left.Empty() {
		t.Fatalf("gravel left: %v", left)
	}
	flint := 0
	s.world.Do(func(tx *world.Tx) {
		inv := tx.Block(machinePos.Side(cube.FaceEast)).(block.Chest).Inventory(tx, machinePos.Side(cube.FaceEast))
		for _, it := range inv.Slots() {
			if !it.Empty() && itemKey(it.Item()) == itemKey(item.Flint{}) {
				flint += it.Count()
			}
		}
	})
	if flint == 0 || flint == 64 {
		t.Fatalf("flint in the chest from 64 gravel = %d, want some but not all", flint)
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
		{block.Air{}, false}, {block.Bedrock{}, false}, {Core{Type: "t1"}, false},
		{Machine{Kind: KindCrusher}, false}, {Drawer{}, false}, {block.NewChest(), false},
	} {
		if _, got := mineable(c.b); got != c.want {
			t.Errorf("mineable(%T) = %v, want %v", c.b, got, c.want)
		}
	}
}

// TestHandsFreeLine runs the M4 goal without a player: core → auto miner →
// crusher → hopper → chest, and a second core → auto miner → sell box.
func TestHandsFreeLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	minerPos := startCorePos.Add(cube.Pos{0, 2, 0})
	crusherPos := minerPos.Side(cube.FaceEast)
	core2Pos := startCorePos.Add(cube.Pos{-3, 0, 0})
	miner2Pos := core2Pos.Add(cube.Pos{0, 2, 0})
	sellPos := miner2Pos.Side(cube.FaceWest)
	hopperPos, chestPos := crusherPos.Side(cube.FaceDown), crusherPos.Add(cube.Pos{0, -2, 0})
	s.world.Do(func(tx *world.Tx) {
		placeCore(tx, startCorePos, "t1", block.Cobblestone{})
		placeCore(tx, core2Pos, "t1", block.Cobblestone{})
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(hopperPos, h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
	})
	placeTestMachine(s, minerPos, KindMiner, "")
	placeTestMachine(s, crusherPos, KindCrusher, "")
	placeTestMachine(s, miner2Pos, KindMiner, "")
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
