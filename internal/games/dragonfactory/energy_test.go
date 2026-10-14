package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
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

// TestItemConduitModeCycles: sneaking with an empty hand cycles the mode of
// the clicked side; a side without items does nothing; a side that is off
// shows no arm.
func TestItemConduitModeCycles(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(machinePos, block.NewChest(), nil)
		tx.SetBlock(pos, ConduitBundle{Item: 2}, nil)
		c := tx.Block(pos).(ConduitBundle)
		p.SetHeldItems(item.Stack{}, item.Stack{})
		if c.Activate(pos, cube.FaceWest, tx, p, nil) {
			t.Fatal("a standing player changed a mode")
		}
		p.StartSneaking()
		want := []itemnet.Mode{itemnet.Input, itemnet.Output, itemnet.Off, itemnet.Both}
		for i, w := range want {
			c = tx.Block(pos).(ConduitBundle)
			if !c.Activate(pos, cube.FaceWest, tx, p, nil) {
				t.Fatalf("click %d was not handled", i)
			}
			if got := tx.Block(pos).(ConduitBundle).Modes[cube.FaceWest]; got != w {
				t.Fatalf("click %d: mode %d, want %d", i, got, w)
			}
			if off := tx.Block(pos).(ConduitBundle).IConn&(1<<4) == 0; off != (w == itemnet.Off) {
				t.Fatalf("click %d: arm hidden = %v for mode %d", i, off, w)
			}
		}
		c = tx.Block(pos).(ConduitBundle)
		before := c.Modes
		c.Activate(pos, cube.FaceNorth, tx, p, nil) // nothing there
		if tx.Block(pos).(ConduitBundle).Modes != before {
			t.Error("a side without items changed its mode")
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

// TestOldConduitsBecomeBundles: a conduit block of the earlier versions (one
// block per conduit) turns into a bundle of its kind and keeps its item modes.
func TestOldConduitsBecomeBundles(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	e, i := machinePos.Add(cube.Pos{1, 0, 0}), machinePos.Add(cube.Pos{3, 0, 0})
	old := legacyConduit{Kind: 1, Tier: 2, Conn: 1 << 4}
	old.Modes[cube.FaceWest] = itemnet.Input
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(e, legacyConduit{Kind: 0, Tier: 3, Conn: 1 << 5}, nil)
		tx.SetBlock(i, old, nil)
	})
	runSteps(s, 10)
	s.world.Do(func(tx *world.Tx) {
		if b, ok := tx.Block(e).(ConduitBundle); !ok || b.Energy != 3 || b.Item != 0 {
			t.Errorf("old energy conduit is %#v", tx.Block(e))
		}
		if b, ok := tx.Block(i).(ConduitBundle); !ok || b.Item != 2 || b.Energy != 0 || b.Modes[cube.FaceWest] != itemnet.Input {
			t.Errorf("old item conduit is %#v", tx.Block(i))
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

// TestOldConduitItemsWork: an old conduit item in an inventory is turned into
// the new item when the player joins, and still places its conduit if used.
func TestOldConduitItemsWork(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	pos := machinePos.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(0, item.NewStack(legacyConduit{Kind: 0, Tier: 1}, 3))
		_ = p.Inventory().SetItem(1, item.NewStack(legacyConduit{Kind: 1, Tier: 1}, 2))
		migrateConduitItems(p)
		if st, _ := p.Inventory().Item(0); st.Item() != (EnergyConduit{Tier: 1}) || st.Count() != 3 {
			t.Errorf("slot 0 holds %v", st)
		}
		if st, _ := p.Inventory().Item(1); st.Item() != (ItemConduit{Tier: 1}) || st.Count() != 2 {
			t.Errorf("slot 1 holds %v", st)
		}
		old := legacyConduit{Kind: 0, Tier: 1}
		p.SetHeldItems(item.NewStack(old, 2), item.Stack{})
		old.UseOnBlock(pos.Side(cube.FaceWest), cube.FaceEast, mgl64.Vec3{}, tx, p, &item.UseContext{})
		if b, ok := tx.Block(pos).(ConduitBundle); !ok || b.Energy != 1 {
			t.Errorf("an old item placed %#v", tx.Block(pos))
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
