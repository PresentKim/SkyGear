package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
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
		tx.SetBlock(machinePos.Add(cube.Pos{1, 0, 0}), EnergyConduit{Tier: 1}, nil)
		tx.SetBlock(machinePos.Add(cube.Pos{2, 0, 0}), EnergyConduit{Tier: 1}, nil)
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
		c := tx.Block(machinePos.Add(cube.Pos{1, 0, 0})).(EnergyConduit)
		want := 1<<4 | 1<<5 // west (the generator) and east (the next conduit)
		if c.Conn != want {
			t.Errorf("conduit connections %b, want %b", c.Conn, want)
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

// TestConduitStateRoundTrip: every set of sides has a block of its own.
func TestConduitStateRoundTrip(t *testing.T) {
	for tier := 1; tier <= conduitTiers; tier++ {
		seen := map[uint64]bool{}
		for conn := range 64 {
			c := EnergyConduit{Tier: tier, Conn: conn}
			name, props := c.EncodeBlock()
			if props[conduitConnState] != conduitConnValue(conn) || name == "" {
				t.Fatalf("encoding %v %v", name, props)
			}
			_, h := c.Hash()
			if seen[h] {
				t.Fatalf("tier %d: the set %d shares a hash", tier, conn)
			}
			seen[h] = true
		}
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
		c := ItemConduit{Tier: 1}
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
		c := tx.Block(mid).(ItemConduit)
		if c.Modes[cube.FaceWest] != itemnet.Input || c.Modes[cube.FaceEast] != itemnet.Output {
			t.Errorf("modes after refreshing: %v", c.Modes)
		}
		if c.Conn != 1<<4|1<<5 {
			t.Errorf("connections %b", c.Conn)
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
		tx.SetBlock(pos, ItemConduit{Tier: 2}, nil)
		c := tx.Block(pos).(ItemConduit)
		p.SetHeldItems(item.Stack{}, item.Stack{})
		if c.Activate(pos, cube.FaceWest, tx, p, nil) {
			t.Fatal("a standing player changed a mode")
		}
		p.StartSneaking()
		want := []itemnet.Mode{itemnet.Input, itemnet.Output, itemnet.Off, itemnet.Both}
		for i, w := range want {
			c = tx.Block(pos).(ItemConduit)
			if !c.Activate(pos, cube.FaceWest, tx, p, nil) {
				t.Fatalf("click %d was not handled", i)
			}
			if got := tx.Block(pos).(ItemConduit).Modes[cube.FaceWest]; got != w {
				t.Fatalf("click %d: mode %d, want %d", i, got, w)
			}
			if off := tx.Block(pos).(ItemConduit).Conn&(1<<4) == 0; off != (w == itemnet.Off) {
				t.Fatalf("click %d: arm hidden = %v for mode %d", i, off, w)
			}
		}
		c = tx.Block(pos).(ItemConduit)
		before := c.Modes
		c.Activate(pos, cube.FaceNorth, tx, p, nil) // nothing there
		if tx.Block(pos).(ItemConduit).Modes != before {
			t.Error("a side without items changed its mode")
		}
	})
}
