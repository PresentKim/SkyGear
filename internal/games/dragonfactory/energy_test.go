package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// run advances n steps of four ticks, calling Fast (which moves the energy)
// after each, as the instance does.
func runSteps(s *session, n int) {
	for range n {
		tick(s, 4)
		s.world.Do(func(tx *world.Tx) { s.Fast(tx) })
	}
}

// TestGeneratorBurnsFuelIntoEnergy: a coal burns for its 80 seconds at 20
// RE/t; a full buffer stops the burning.
func TestGeneratorBurnsFuelIntoEnergy(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindGenerator, "")
	st.unlimited = false
	if !st.insert(item.NewStack(item.Coal{}, 2)) {
		t.Fatal("coal was not taken as fuel")
	}
	if st.insert(item.NewStack(block.Stone{}, 1)) {
		t.Fatal("stone was taken as fuel")
	}
	tick(s, 10)
	if st.energy != 10*generatorOutput {
		t.Fatalf("%d RE after 10 ticks, want %d", st.energy, 10*generatorOutput)
	}
	if st.burn != 1600-10 {
		t.Errorf("burn %d, want %d", st.burn, 1600-10)
	}
	// With a full buffer nothing burns.
	st.energy = st.kind.energyCap
	burn := st.burn
	tick(s, 10)
	if st.energy != st.kind.energyCap || st.burn != burn {
		t.Errorf("a full generator burned: energy %d burn %d", st.energy, st.burn)
	}
	// The generator gives it away and burns on.
	if got := (Machine{Kind: KindGenerator, st: st}).ExtractEnergy(nil, machinePos, cube.FaceEast, 500, false); got != 500 {
		t.Errorf("extracted %d", got)
	}
	tick(s, 5)
	if st.burn >= burn {
		t.Error("the generator did not burn on after room came")
	}
}

// TestPowerLine: a generator, two conduits and a crusher: the crusher gets
// energy and works; without the generator it stalls.
func TestPowerLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	gen := placeTestMachine(s, machinePos, KindGenerator, "")
	gen.unlimited = false
	crusher := placeTestMachine(s, machinePos.Add(cube.Pos{3, 0, 0}), KindCrusher, "")
	crusher.unlimited = false
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(machinePos.Add(cube.Pos{1, 0, 0}), ConduitBundle{Energy: 1}, nil)
		tx.SetBlock(machinePos.Add(cube.Pos{2, 0, 0}), ConduitBundle{Energy: 1}, nil)
	})
	_ = crusher.insert(item.NewStack(block.Cobblestone{}, 3))
	runSteps(s, 30)
	if outputCount(crusher, block.Gravel{}) != 0 || crusher.energy != 0 {
		t.Fatalf("the crusher worked without a generator: %d gravel, %d RE", outputCount(crusher, block.Gravel{}), crusher.energy)
	}
	if !crusher.stalled {
		t.Error("a crusher without energy does not say it is stalled")
	}
	_ = gen.insert(item.NewStack(item.Coal{}, 1))
	runSteps(s, 120)
	if crusher.energy == 0 && outputCount(crusher, block.Gravel{}) == 0 {
		t.Fatal("no energy reached the crusher")
	}
	if outputCount(crusher, block.Gravel{}) == 0 {
		t.Errorf("the powered crusher made no gravel (energy %d, progress %d)", crusher.energy, crusher.progress)
	}
	// The conduits showed their arms.
	s.world.Do(func(tx *world.Tx) {
		c := tx.Block(machinePos.Add(cube.Pos{1, 0, 0})).(ConduitBundle)
		want := 1<<4 | 1<<5 // west (the generator) and east (the next conduit)
		if c.EConn != want || c.IConn != 0 {
			t.Errorf("conduit connections %b %b, want %b", c.EConn, c.IConn, want)
		}
	})
}

// TestConduitsOnlyJoinEnergyBlocks: a conduit next to a sieve (no energy
// role) or a crusher (a sink) connects only to the crusher; a conduit item
// (no state) does not.
func TestConduitsOnlyJoinEnergyBlocks(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	placeTestMachine(s, machinePos.Add(cube.Pos{1, 0, 0}), KindCrusher, "")
	placeTestMachine(s, machinePos.Add(cube.Pos{-1, 0, 0}), KindSieve, "")
	s.world.Do(func(tx *world.Tx) {
		if joinsEnergy(tx.Block(machinePos.Add(cube.Pos{-1, 0, 0}))) {
			t.Error("a conduit joins a hand sieve")
		}
		if !joinsEnergy(tx.Block(machinePos.Add(cube.Pos{1, 0, 0}))) {
			t.Error("a conduit does not join a crusher")
		}
		if joinsEnergy(block.Stone{}) {
			t.Error("a conduit joins stone")
		}
		if src, snk := energy.RolesOf(Machine{Kind: KindCrusher}); src != nil || snk != nil {
			t.Error("a machine item has energy ports")
		}
	})
}

// TestConduitLookCodes: the look of a bundle in the two numbers of the display
// entity is told apart by every kind, tier and set of sides, and fits the
// client's float32 molang exactly.
func TestConduitLookCodes(t *testing.T) {
	seen := map[[2]int32]bool{}
	for e := 0; e <= conduitTiers; e++ {
		for i := 0; i <= conduitTiers; i++ {
			for ec := range 64 {
				for ic := range 64 {
					b := ConduitBundle{Energy: e, Item: i}
					if e > 0 {
						b.EConn = ec
					}
					if i > 0 {
						b.IConn = ic
					}
					v, m := conduitLook(b)
					if v < 0 || v >= conduitCodes*conduitCodes || m != 0 {
						t.Fatalf("look %d %d out of range", v, m)
					}
					// the render controller's way of taking it apart
					ecode, icode := v%conduitCodes, v/conduitCodes
					if (ecode > 0) != (e > 0) || (icode > 0) != (i > 0) {
						t.Fatalf("presence lost in %d for %d %d", v, e, i)
					}
					if e > 0 && ((ecode-1)/64+1 != int32(e) || int((ecode-1)%64) != b.EConn) {
						t.Fatalf("energy code %d for tier %d sides %d", ecode, e, b.EConn)
					}
					if i > 0 && ((icode-1)/64+1 != int32(i) || int((icode-1)%64) != b.IConn) {
						t.Fatalf("item code %d for tier %d sides %d", icode, i, b.IConn)
					}
					if float32(v) != float32(int32(float32(v))) || float32(v)/193 < 0 {
						t.Fatal("not exact in float32")
					}
					seen[[2]int32{v, m}] = true
				}
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("nothing")
	}
}

// TestItemConduitMovesItems: a chest gives over a conduit into another chest
// (tier 1: one item per 0.4 seconds), and the modes survive the arms being
// refreshed.
func TestItemConduitMovesItems(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	from, mid, to := machinePos, machinePos.Add(cube.Pos{1, 0, 0}), machinePos.Add(cube.Pos{2, 0, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(from, block.NewChest(), nil)
		tx.SetBlock(to, block.NewChest(), nil)
		_, _ = tx.Block(from).(block.Chest).Inventory(tx, from).AddItem(item.NewStack(block.Stone{}, 40))
		c := ConduitBundle{Item: 1}
		c.Modes[cube.FaceWest] = itemnet.Input
		c.Modes[cube.FaceEast] = itemnet.Output
		tx.SetBlock(mid, c, nil)
	})
	runSteps(s, 20)
	s.world.Do(func(tx *world.Tx) {
		got := 0
		for _, st := range tx.Block(to).(block.Chest).Inventory(tx, to).Slots() {
			got += st.Count()
		}
		if got < 8 || got > 12 {
			t.Errorf("%d stone moved in 20 steps (4 seconds), want about 10", got)
		}
		c := tx.Block(mid).(ConduitBundle)
		if c.Modes[cube.FaceWest] != itemnet.Input || c.Modes[cube.FaceEast] != itemnet.Output {
			t.Errorf("modes after refreshing: %v", c.Modes)
		}
		if c.IConn != 1<<4|1<<5 || c.EConn != 0 {
			t.Errorf("connections %b %b", c.EConn, c.IConn)
		}
	})
}

// TestConduitsShareABundle: placing an energy and then an item conduit on the
// same block puts both in one bundle; a second conduit of a kind needs a block
// of its own; breaking the bundle gives both back; and the two kinds of
// conduit keep to their own networks.
func TestConduitsShareABundle(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func(it world.Item, at cube.Pos, face cube.Face) bool {
			_ = p.Inventory().SetItem(0, item.NewStack(it, 4))
			p.SetHeldItems(item.NewStack(it, 4), item.Stack{})
			ctx := &item.UseContext{}
			return it.(item.UsableOnBlock).UseOnBlock(at, face, mgl64.Vec3{}, tx, p, ctx)
		}
		if !use(EnergyConduit{Tier: 2}, pos.Side(cube.FaceWest), cube.FaceEast) {
			t.Fatal("the energy conduit was not placed")
		}
		b, ok := tx.Block(pos).(ConduitBundle)
		if !ok || b.Energy != 2 || b.Item != 0 {
			t.Fatalf("placed %#v", tx.Block(pos))
		}
		if !use(ItemConduit{Tier: 3}, pos, cube.FaceUp) {
			t.Fatal("the item conduit was not put into the bundle")
		}
		b = tx.Block(pos).(ConduitBundle)
		if b.Energy != 2 || b.Item != 3 || tx.Block(pos.Side(cube.FaceUp)) != (block.Air{}) {
			t.Fatalf("bundle %#v, above %#v", b, tx.Block(pos.Side(cube.FaceUp)))
		}
		if use(EnergyConduit{Tier: 1}, pos, cube.FaceUp) {
			t.Error("a second energy conduit went into a full bundle")
		}
		if _, ok := energy.IsConduit(b); !ok {
			t.Error("the bundle is no energy conduit")
		}
		if _, ok := itemnet.IsConduit(b); !ok {
			t.Error("the bundle is no item conduit")
		}
		if _, ok := energy.IsConduit(ConduitBundle{Item: 1}); ok {
			t.Error("a bundle with only an item conduit carries energy")
		}
		if _, ok := itemnet.IsConduit(ConduitBundle{Energy: 1}); ok {
			t.Error("a bundle with only an energy conduit carries items")
		}
		if drops := b.BreakInfo().Drops(item.Tool(nil), nil); len(drops) != 2 {
			t.Errorf("drops %v", drops)
		}
	})
}

// TestConduitVisualFollowsTheBundle: a placed conduit gets a display entity
// that carries its look; putting another conduit into the bundle or
// connecting it changes the look; the entity goes with the bundle.
func TestConduitVisualFollowsTheBundle(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	look := func(tx *world.Tx) (int32, bool) {
		handle, ok := s.visuals[pos]
		if !ok {
			return 0, false
		}
		e, ok := handle.Entity(tx)
		if !ok {
			return 0, false
		}
		return e.(*ConduitVisual).Variant(), true
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func(it world.Item, at cube.Pos, face cube.Face) {
			p.SetHeldItems(item.NewStack(it, 4), item.Stack{})
			ctx := &item.UseContext{}
			it.(item.UsableOnBlock).UseOnBlock(at, face, mgl64.Vec3{}, tx, p, ctx)
		}
		use(EnergyConduit{Tier: 2}, pos.Side(cube.FaceWest), cube.FaceEast)
		if v, ok := look(tx); !ok || v != conduitCode(2, 0) {
			t.Fatalf("look %d (%v) after placing an energy conduit, want %d", v, ok, conduitCode(2, 0))
		}
		use(ItemConduit{Tier: 1}, pos, cube.FaceUp)
		if v, ok := look(tx); !ok || v != conduitCode(2, 0)+conduitCodes*conduitCode(1, 0) {
			t.Fatalf("look %d (%v) after adding an item conduit", v, ok)
		}
		// A chest beside it: the item conduit reaches for it, the energy one does not.
		tx.SetBlock(pos.Side(cube.FaceEast), block.NewChest(), nil)
		b := tx.Block(pos).(ConduitBundle)
		b.refresh(pos, tx)
		b = tx.Block(pos).(ConduitBundle)
		if b.Modes[cube.FaceEast] != itemnet.Off || b.IConn != 0 {
			t.Fatalf("a new item conduit reaches the chest: mode %v, connections %b", b.Modes[cube.FaceEast], b.IConn)
		}
		b.Modes[cube.FaceEast] = itemnet.Both
		tx.SetBlock(pos, b, nil)
		b.refresh(pos, tx)
		b = tx.Block(pos).(ConduitBundle)
		if b.EConn != 0 || b.IConn != 1<<5 {
			t.Fatalf("connections energy %b item %b, want only the item conduit towards the chest", b.EConn, b.IConn)
		}
		if v, _ := look(tx); v != conduitCode(2, 0)+conduitCodes*conduitCode(1, 1<<5) {
			t.Errorf("look %d after the chest came", v)
		}
		s.removeVisual(tx, pos)
		if _, ok := look(tx); ok {
			t.Error("the entity stayed after the bundle was removed")
		}
	})
}

// TestConduitPlacementWays: a conduit goes into the bundle that was clicked if
// it has room; otherwise it does nothing, but while sneaking the bundle counts
// as the ground and the conduit goes into a bundle (or a new one) on the
// clicked side; a block behind a thin conduit that was clicked puts the
// conduit into the bundle in front of it if it has room.
func TestConduitPlacementWays(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func(it world.Item, at cube.Pos, face cube.Face) bool {
			p.SetHeldItems(item.NewStack(it, 8), item.Stack{})
			return it.(item.UsableOnBlock).UseOnBlock(at, face, mgl64.Vec3{}, tx, p, &item.UseContext{})
		}
		tx.SetBlock(pos, ConduitBundle{Energy: 1}, nil)
		// the same kind on the bundle does nothing
		if use(EnergyConduit{Tier: 1}, pos, cube.FaceUp) {
			t.Fatal("an energy conduit went into a bundle that has one")
		}
		if _, ok := tx.Block(pos.Side(cube.FaceUp)).(ConduitBundle); ok {
			t.Fatal("a conduit was placed above while standing")
		}
		// sneaking: the bundle is the ground
		p.StartSneaking()
		if !use(EnergyConduit{Tier: 1}, pos, cube.FaceUp) {
			t.Fatal("a sneaking player could not place on a bundle")
		}
		if b, ok := tx.Block(pos.Side(cube.FaceUp)).(ConduitBundle); !ok || b.Energy != 1 {
			t.Fatalf("above the bundle: %#v", tx.Block(pos.Side(cube.FaceUp)))
		}
		// sneaking with a kind the bundle lacks still places beside it
		if !use(ItemConduit{Tier: 1}, pos, cube.FaceNorth) {
			t.Fatal("could not place beside while sneaking")
		}
		if b, ok := tx.Block(pos.Side(cube.FaceNorth)).(ConduitBundle); !ok || b.Item != 1 || b.Energy != 0 {
			t.Fatalf("beside the bundle: %#v", tx.Block(pos.Side(cube.FaceNorth)))
		}
		p.StopSneaking()
		// a block behind a thin conduit: a stone east of the bundle, clicked
		// on its west face, puts the conduit into the bundle in front of it
		stone := pos.Side(cube.FaceEast)
		tx.SetBlock(stone, block.Stone{}, nil)
		if !use(FluidConduit{Tier: 1}, stone, cube.FaceWest) {
			t.Fatal("clicking the block behind a conduit placed nothing")
		}
		if b := tx.Block(pos).(ConduitBundle); b.Fluid != 1 || b.Energy != 1 {
			t.Fatalf("the bundle in front of the block: %#v", b)
		}
		// and with no room there nothing is placed
		if use(FluidConduit{Tier: 1}, stone, cube.FaceWest) {
			t.Error("a second fluid conduit went into the bundle")
		}
	})
}

// TestConduitPlates: an item conduit beside a chest gets a connector plate on
// that side only (not on the sides with nothing), its look carries the side, the
// kind and the mode and channels; the plate's square on the block it touches is
// where a click counts; and the plates go with the bundle.
func TestConduitPlates(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(machinePos, block.NewChest(), nil)
		b := ConduitBundle{Item: 1, Energy: 1}
		b.Modes[cube.FaceWest] = itemnet.Input
		b.Extract[cube.FaceWest], b.Insert[cube.FaceWest] = 5, 9
		tx.SetBlock(pos, b, nil)
		b.refresh(pos, tx)
		west := plateKey{pos, 4}
		h, ok := s.plates[west]
		if !ok || len(s.plates) != 1 {
			t.Fatalf("plates %v, want one on the west side", s.plates)
		}
		e, alive := h.Entity(tx)
		if !alive {
			t.Fatal("the plate entity is gone")
		}
		plate := e.(*ConduitPlate)
		want := int32(4 + 6*2 + 48*int(itemnet.Input) + 192*5 + 3072*9)
		if plate.Variant() != want || plate.MarkVariant() != 1+4*1 {
			t.Errorf("look %d %d, want %d %d", plate.Variant(), plate.MarkVariant(), want, 5)
		}
		if plate.Position().X() != float64(pos[0])+0.5 {
			t.Errorf("the plate stands at %v", plate.Position())
		}
		// a click on the chest's east face in the middle is on the plate, one near the edge is not
		if _, _, side, ok := plateHit(tx, machinePos, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.55}); !ok || side != 4 {
			t.Errorf("a click in the middle missed the plate (side %d, %v)", side, ok)
		}
		if _, _, _, ok := plateHit(tx, machinePos, cube.FaceEast, mgl64.Vec3{1, 0.5, 0.95}); ok {
			t.Error("a click near the edge hit the plate")
		}
		if _, _, _, ok := plateHit(tx, machinePos, cube.FaceNorth, mgl64.Vec3{0.5, 0.5, 0}); ok {
			t.Error("a click on a face with no conduit hit a plate")
		}
		// the settings survive the NBT
		back := ConduitBundle{}.DecodeNBT(b.EncodeNBT()).(ConduitBundle)
		if back.Extract[cube.FaceWest] != 5 || back.Insert[cube.FaceWest] != 9 || back.Modes[cube.FaceWest] != itemnet.Input {
			t.Errorf("after NBT: %v %v %v", back.Extract, back.Insert, back.Modes)
		}
		// switching the side off keeps its plate
		b.Modes[cube.FaceWest] = itemnet.Off
		tx.SetBlock(pos, b, nil)
		b.refresh(pos, tx)
		if _, ok := s.plates[west]; !ok {
			t.Error("an off side lost its plate")
		}
		s.removeVisual(tx, pos)
		if len(s.plates) != 0 {
			t.Errorf("plates left after the bundle went: %v", s.plates)
		}
	})
}

// TestExtractAndInsertFlags: the settings form's two switches make the modes.
func TestExtractAndInsertFlags(t *testing.T) {
	for _, c := range []struct {
		extract, insert bool
		want            itemnet.Mode
	}{{true, true, itemnet.Both}, {true, false, itemnet.Input}, {false, true, itemnet.Output}, {false, false, itemnet.Off}} {
		m := itemnet.ModeOf(c.extract, c.insert)
		if m != c.want || m.Gives() != c.extract || m.Takes() != c.insert {
			t.Errorf("ModeOf(%v, %v) = %v", c.extract, c.insert, m)
		}
	}
}

// TestPlateFormsOpen: the settings form and the channel picker open for a side
// with a plate (and nothing happens for a side without).
func TestPlateFormsOpen(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(machinePos, block.NewChest(), nil)
		tx.SetBlock(pos, ConduitBundle{Item: 1, Energy: 1}, nil)
		b := tx.Block(pos).(ConduitBundle)
		b.refresh(pos, tx)
		s.showPlateForm(p, tx, pos, 4)
		s.showConduitSides(p, tx, pos, 4)
		board := s.conduitSideBoard(p, tx, pos, 4)
		west := board.Sides[cube.FaceWest].Ports
		if len(west) != 2 || len(west[1].Choices) != itemnet.Channels+1 || west[1].Choices[0].Colour != "" || west[1].Choices[1].Colour == "" {
			t.Fatalf("west side ports %v", west)
		}
		west[1].Choices[2].Run(p, tx)
		if got := tx.Block(pos).(ConduitBundle); !got.Modes[cube.FaceWest].Gives() || got.Extract[cube.FaceWest] != 1 {
			t.Errorf("chose extract channel: %v %v", got.Modes, got.Extract)
		}
		s.showPlateForm(p, tx, pos, 0) // no plate below
		s.setPlateState(tx, pos, cube.FaceWest, true, 2)
		if got := tx.Block(pos).(ConduitBundle); !got.Modes[cube.FaceWest].Gives() || got.Extract[cube.FaceWest] != 1 {
			t.Errorf("set extract channel: %v %v", got.Modes, got.Extract)
		}
		// the form's changes are stored as modes and channels
		b = tx.Block(pos).(ConduitBundle)
		b.Modes[cube.FaceWest] = itemnet.ModeOf(false, true)
		b.Insert[cube.FaceWest] = 3
		s.storeConduitSide(tx, pos, b)
		got := tx.Block(pos).(ConduitBundle)
		if got.Modes[cube.FaceWest] != itemnet.Output || got.Insert[cube.FaceWest] != 3 {
			t.Errorf("stored %v %v", got.Modes, got.Insert)
		}
	})
}

// TestConduitBlockSizes: the conduit block collides, is pointed at and is
// shaped like the 9 pixels its conduits take, not a full block; fluid conduits
// move EnderIO's 200, 1000 and 8000 mB a tick.
func TestConduitBlockSizes(t *testing.T) {
	for name, props := range map[string]customblock.Properties{"bundle": ConduitBundle{}.Properties()} {
		if props.CollisionBox != bundleSelection || props.SelectionBox != bundleSelection {
			t.Errorf("%s: collision %v, selection %v, want %v", name, props.CollisionBox, props.SelectionBox, bundleSelection)
		}
	}
	box := bundleSelection
	if w := (box.Max()[0] - box.Min()[0]) * 16; w < 8.99 || w > 9.01 {
		t.Errorf("the box is %v pixels wide", w)
	}
	for tier, want := range map[int]int{1: 200, 2: 1000, 3: 8000} {
		if fluidRate(tier) != want {
			t.Errorf("fluid tier %d moves %d mB/t, want %d", tier, fluidRate(tier), want)
		}
	}
}

// TestPlateClicksWithAnythingInHand: a sneaking player holding a tool opens the
// settings by clicking the plate's square on the block it touches, or the side
// of the conduit; one holding a conduit item places instead; standing does
// nothing.
func TestPlateClicksWithAnythingInHand(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(machinePos, block.NewChest(), nil)
		tx.SetBlock(pos, ConduitBundle{Item: 1}, nil)
		b := tx.Block(pos).(ConduitBundle)
		b.refresh(pos, tx)
		click := func(at cube.Pos, face cube.Face, held item.Stack, sneak bool) bool {
			p.SetHeldItems(held, item.Stack{})
			if sneak {
				p.StartSneaking()
			} else {
				p.StopSneaking()
			}
			ctx := player.NewEventContext(tx, p)
			(&handler{s: s}).HandleItemUseOnBlock(ctx, at, face, mgl64.Vec3{1, 0.5, 0.5})
			return ctx.Cancelled()
		}
		pick := item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)
		if !click(machinePos, cube.FaceEast, pick, true) {
			t.Error("a pickaxe in hand: the chest face with the plate did not open the settings")
		}
		if !click(machinePos, cube.FaceEast, item.Stack{}, true) {
			t.Error("an empty hand: the chest face did not open the settings")
		}
		if !click(pos, cube.FaceWest, pick, true) {
			t.Error("a pickaxe in hand: the conduit's side with the plate did not open the settings")
		}
		if click(machinePos, cube.FaceEast, pick, false) {
			t.Error("standing: the click was taken")
		}
		if click(machinePos, cube.FaceEast, item.NewStack(ItemConduit{Tier: 1}, 1), true) {
			t.Error("a conduit item in hand did not place")
		}
	})
}

// TestChannelItems: a side state shows a barrier when off and concrete in the
// channel's colour when on; every state has an icon and every channel its own
// colour square.
func TestChannelItems(t *testing.T) {
	if _, ok := channelItem(0).(block.Barrier); !ok {
		t.Errorf("off shows %T", channelItem(0))
	}
	colours := map[string]bool{}
	for c := range itemnet.Channels {
		got, ok := channelItem(1 + c).(block.Concrete)
		if !ok || got.Colour != channelColours[c] {
			t.Errorf("channel %d shows %v", c, channelItem(1+c))
		}
		if ui.ItemIcon(channelItem(1+c)) == "" {
			t.Errorf("channel %d has no icon", c)
		}
		colours[channelColour(c)] = true
	}
	if len(colours) != itemnet.Channels {
		t.Errorf("%d colour squares for %d channels", len(colours), itemnet.Channels)
	}
}

// TestNextPlateSide: clicking a conduit where no plate sits opens the plates in
// turn, one per click, and starts over after the last.
func TestNextPlateSide(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceWest), block.NewChest(), nil)
		tx.SetBlock(pos.Side(cube.FaceEast), block.NewChest(), nil)
		tx.SetBlock(pos, ConduitBundle{Item: 1}, nil)
		b := tx.Block(pos).(ConduitBundle)
		b.refresh(pos, tx)
		b = tx.Block(pos).(ConduitBundle)
		var got []int
		for range 3 {
			side, ok := s.nextPlateSide(p, tx, pos, b)
			if !ok {
				t.Fatal("no plate side")
			}
			got = append(got, side)
		}
		west, east := faceIndex(cube.FaceWest), faceIndex(cube.FaceEast)
		first, second := min(west, east), max(west, east)
		if got[0] != first || got[1] != second || got[2] != first {
			t.Errorf("sides %v, want %d %d %d", got, first, second, first)
		}
		tx.SetBlock(pos, ConduitBundle{Item: 1}, nil)
		tx.SetBlock(pos.Side(cube.FaceWest), block.Air{}, nil)
		tx.SetBlock(pos.Side(cube.FaceEast), block.Air{}, nil)
		if _, ok := s.nextPlateSide(p, tx, pos, ConduitBundle{Item: 1}); ok {
			t.Error("a side without targets was found")
		}
	})
}

// TestChestOpensUnderConduit: a chest with a conduit bundle above it opens (the
// block above must diffuse little light).
func TestChestOpensUnderConduit(t *testing.T) {
	var _ block.LightDiffuser = ConduitBundle{}
	if (ConduitBundle{}).LightDiffusionLevel() > 2 {
		t.Error("a bundle blocks light")
	}
}
