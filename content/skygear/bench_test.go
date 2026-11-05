package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// benchSession returns a session with the area around the origin loaded.
func benchSession(b *testing.B) *session {
	b.Helper()
	s := testSession(b)
	loadAround(s)
	return s
}

// BenchmarkMachineTick ticks a powered crusher that always has input.
func BenchmarkMachineTick(b *testing.B) {
	s := benchSession(b)
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, _, ok := st.FirstInput(); !ok {
			_ = st.Insert(item.NewStack(block.Cobblestone{}, 64))
		}
		for _, slot := range st.Def.Output {
			_ = st.Inv().SetItem(slot, item.Stack{})
		}
		s.World.AdvanceTick()
	}
}

// BenchmarkMachineTickStuck ticks a crusher whose output is full.
func BenchmarkMachineTickStuck(b *testing.B) {
	s := benchSession(b)
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	for i, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, 64).WithValue("fill", i))
	}
	_ = st.Insert(item.NewStack(block.Cobblestone{}, 64))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.World.AdvanceTick()
	}
}

// BenchmarkMarshalSaved flushes the saved state of a session with 2000 crops.
func BenchmarkMarshalSaved(b *testing.B) {
	s := benchSession(b)
	fs := farm.StateOf(s.Session)
	fs.Crops = map[string]*farm.CropRecord{}
	for i := range 2000 {
		fs.Crops[farm.CropKey(cube.Pos{i % 50, 60, i / 50})] = &farm.CropRecord{Kind: "wheat"}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.Flush()
	}
}

// BenchmarkBundleTick ticks the world with 100 conduit bundles placed.
func BenchmarkBundleTick(b *testing.B) {
	s := benchSession(b)
	s.World.Do(func(tx *world.Tx) {
		for i := range 100 {
			tx.SetBlock(machinePos.Add(cube.Pos{i, 0, 0}), conduit.Bundle{Energy: 1, Item: 1}, nil)
		}
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.World.AdvanceTick()
	}
}

// BenchmarkEnergyStep runs the energy step of a network of 50 conduits, a
// generator and 10 machines.
func BenchmarkEnergyStep(b *testing.B) {
	s := benchSession(b)
	gen := placeTestMachine(s, machinePos, machine.KindGenerator, "")
	gen.Unlimited = false
	s.World.Do(func(tx *world.Tx) {
		for i := range 50 {
			tx.SetBlock(machinePos.Add(cube.Pos{1 + i, 0, 0}), conduit.Bundle{Energy: 1}, nil)
		}
	})
	for i := range 10 {
		m := placeTestMachine(s, machinePos.Add(cube.Pos{1 + i*5, 0, 1}), machine.KindCrusher, "")
		m.Unlimited = false
	}
	_ = gen.Insert(item.NewStack(item.Coal{}, 64))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
	}
}

// BenchmarkControllerPeek peeks into a drawer controller with 50 drawers.
func BenchmarkControllerPeek(b *testing.B) {
	s := benchSession(b)
	ctl := machinePos
	placeTestMachine(s, ctl, machine.KindDrawerController, "")
	s.World.Do(func(tx *world.Tx) {
		n := 0
		for dy := 0; dy < 2 && n < 50; dy++ {
			for dz := 0; dz < 5 && n < 50; dz++ {
				for dx := 1; dx <= 5 && n < 50; dx++ {
					pos := ctl.Add(cube.Pos{dx, dy, dz})
					if pos == ctl {
						continue
					}
					tx.SetBlock(pos, storage.Drawer{Facing: cube.South}, nil)
					n++
				}
			}
		}
		for dx := 1; n < 50; dx++ {
			tx.SetBlock(ctl.Add(cube.Pos{dx, 0, 5}), storage.Drawer{Facing: cube.South}, nil)
			n++
		}
		if got := len(storage.Network(tx, ctl)); got != 50 {
			b.Fatalf("%d drawers in the network, want 50", got)
		}
		storage.ControllerInsert(tx, ctl, item.NewStack(block.Cobblestone{}, 100))
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.World.Do(func(tx *world.Tx) { storage.ControllerPeek(tx, ctl, nil) })
	}
}
