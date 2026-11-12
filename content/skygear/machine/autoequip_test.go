package machine_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machine/machinetest"
)

// TestElectricFurnace: it smelts what a furnace smelts and takes nothing else.
func TestElectricFurnace(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindFurnace)
	if !st.Accepts(item.NewStack(block.Cobblestone{}, 1)) || st.Accepts(item.NewStack(block.Dirt{}, 1)) {
		t.Fatal("the furnace takes the wrong inputs")
	}
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 2))
	coretest.Tick(s, 2*(machine.Kinds[machine.KindFurnace].Interval+2))
	if machine.OutputCount(st, block.Stone{}) == 0 {
		t.Fatal("nothing was smelted")
	}
}

// TestItemVacuum: it takes the dropped items in reach and leaves the ones
// out of reach.
func TestItemVacuum(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindVacuum)
	near, far := machinetest.Pos.Vec3Centre().Add(mgl64.Vec3{3, 0, 0}), machinetest.Pos.Vec3Centre().Add(mgl64.Vec3{30, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: near}, item.NewStack(block.Cobblestone{}, 5)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: far}, item.NewStack(block.Cobblestone{}, 7)))
	})
	coretest.Tick(s, 2*(machine.Kinds[machine.KindVacuum].Interval+2))
	if got := machine.OutputCount(st, block.Cobblestone{}); got != 5 {
		t.Fatalf("collected %d, want the 5 in reach", got)
	}
	left := 0
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if ent, ok := e.(*entity.Ent); ok {
				if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
					left += ib.Item().Count()
				}
			}
		}
	})
	if left != 7 {
		t.Fatalf("%d items left lying, want the 7 out of reach", left)
	}
}

// lyingItems counts the dropped items still in the world.
func lyingItems(s core.Lifecycle) int {
	left := 0
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if ent, ok := e.(*entity.Ent); ok {
				if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
					left += ib.Item().Count()
				}
			}
		}
	})
	return left
}

// TestItemVacuumTakesItemsOnItsFloor: an item lying on the floor of the area
// stands exactly on its edge, and is taken all the same.
func TestItemVacuumTakesItemsOnItsFloor(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindVacuum)
	st.RangeY = 2
	floor := machinetest.Pos.Add(cube.Pos{2, -3, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(floor, block.Stone{}, nil)
		tx.SetBlock(floor.Side(cube.FaceUp), block.Air{}, nil)
		at := floor.Vec3Centre().Add(mgl64.Vec3{0, 0.5, 0})
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: at}, item.NewStack(block.Dirt{}, 4)))
	})
	coretest.Tick(s, 2*(machine.Kinds[machine.KindVacuum].Interval+2))
	if got := machine.OutputCount(st, block.Dirt{}); got != 4 {
		t.Fatalf("collected %d, want the 4 on the floor of the area", got)
	}
}

// TestItemVacuumOffset: a moved area takes the items around its new centre
// and leaves the ones around the machine.
func TestItemVacuumOffset(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindVacuum)
	st.RangeX, st.OffsetX = 2, 4
	in, out := machinetest.Pos.Vec3Centre().Add(mgl64.Vec3{6, 0, 0}), machinetest.Pos.Vec3Centre().Add(mgl64.Vec3{-2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: in}, item.NewStack(block.Cobblestone{}, 5)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: out}, item.NewStack(block.Cobblestone{}, 7)))
	})
	coretest.Tick(s, 2*(machine.Kinds[machine.KindVacuum].Interval+2))
	if got := machine.OutputCount(st, block.Cobblestone{}); got != 5 {
		t.Fatalf("collected %d, want the 5 around the moved centre", got)
	}
	if left := lyingItems(s); left != 7 {
		t.Fatalf("%d items left lying, want the 7 by the machine", left)
	}
}

// TestTreeFeller: it fells the highest log first, then plants a sapling
// when no log is left.
func TestTreeFeller(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindFeller)
	log := func() world.Block { return block.Log{Wood: block.OakWood()} }
	trunk := machinetest.Pos.Add(cube.Pos{2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		for dy := range 3 {
			tx.SetBlock(trunk.Add(cube.Pos{0, dy, 0}), log(), nil)
		}
		for dx := -3; dx <= 3; dx += 3 {
			for dz := -3; dz <= 3; dz += 3 {
				at := machinetest.Pos.Add(cube.Pos{dx, -1, dz})
				if dx != 0 || dz != 0 {
					tx.SetBlock(at, block.Grass{}, nil)
					tx.SetBlock(at.Side(cube.FaceUp), block.Air{}, nil)
				}
			}
		}
	})
	_ = st.Inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	coretest.Tick(s, 3*(machine.Kinds[machine.KindFeller].Interval+2))
	s.World.Do(func(tx *world.Tx) {
		if _, ok := tx.Block(trunk.Add(cube.Pos{0, 2, 0})).(block.Log); ok {
			t.Error("the top log was not felled first")
		}
	})
	if machine.OutputCount(st, block.Log{Wood: block.OakWood()}) < 2 {
		t.Fatalf("logs collected: %d", machine.OutputCount(st, block.Log{Wood: block.OakWood()}))
	}
	coretest.Tick(s, 6*(machine.Kinds[machine.KindFeller].Interval+2))
	planted := 0
	s.World.Do(func(tx *world.Tx) {
		for dx := -3; dx <= 3; dx += 3 {
			for dz := -3; dz <= 3; dz += 3 {
				if _, ok := tx.Block(machinetest.Pos.Add(cube.Pos{dx, 0, dz})).(block.Sapling); ok {
					planted++
				}
			}
		}
	})
	if planted == 0 {
		t.Fatal("no sapling was planted once the logs were gone")
	}
}

func TestTreeFellerBoneMealsSaplings(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindFeller)
	sapling := machinetest.Pos.Add(cube.Pos{2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(sapling.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(sapling, block.Sapling{Type: block.OakSapling()}, nil)
	})
	if !st.Insert(item.NewStack(item.BoneMeal{}, 1)) {
		t.Fatal("feller took no bone meal")
	}
	if got, _ := st.Inv().Item(st.Def.Charge[0]); got.Count() != 1 {
		t.Fatalf("bone meal slot holds %v", got)
	}
	_ = st.Inv().SetItem(st.Def.Charge[0], item.NewStack(item.BoneMeal{}, 64))
	coretest.Tick(s, 40*(machine.Kinds[machine.KindFeller].Interval+2))
	if got, _ := st.Inv().Item(st.Def.Charge[0]); got.Count() == 64 {
		t.Fatal("no bone meal was used")
	}
	if machine.OutputCount(st, block.Log{Wood: block.OakWood()}) == 0 {
		t.Fatal("the sapling did not grow into a tree that was felled")
	}
}

func TestTreeFellerPlantsOnAnySoilInReach(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindFeller)
	grassy := machinetest.Pos.Add(cube.Pos{2, 1, 1})
	bare := machinetest.Pos.Add(cube.Pos{-4, -1, 2})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(grassy.Side(cube.FaceDown), block.Grass{}, nil)
		tx.SetBlock(grassy, block.ShortGrass{}, nil)
		tx.SetBlock(bare.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(bare, block.Air{}, nil)
	})
	_ = st.Inv().SetItem(10, item.NewStack(block.Sapling{Type: block.OakSapling()}, 2))
	coretest.Tick(s, 3*(machine.Kinds[machine.KindFeller].Interval+2))
	s.World.Do(func(tx *world.Tx) {
		for _, at := range []cube.Pos{grassy, bare} {
			if _, ok := tx.Block(at).(block.Sapling); !ok {
				t.Errorf("no sapling at %v", at)
			}
		}
	})
}

// rangeBoxAt is where the range box of the machine at pos stands, and false
// with no box in the world.
func rangeBoxAt(s core.Lifecycle, pos cube.Pos) (at mgl64.Vec3, ok bool) {
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if b, isBox := e.(*machine.RangeBox); isBox {
				at, ok = b.Position(), true
			}
		}
	})
	return at, ok
}

// TestRangeBoxFollowsAMovedCentre: moving the centre sideways keeps the
// area's size, and the box shown moves with it at once.
func TestRangeBoxFollowsAMovedCentre(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	st := machinetest.Place(s, machinetest.Pos, machine.KindVacuum)
	peek := func() {
		s.World.Do(func(tx *world.Tx) { machine.PeekRange(tx, machinetest.Pos, tx.Block(machinetest.Pos)) })
		coretest.Tick(s, 2)
	}
	peek()
	before, ok := rangeBoxAt(s, machinetest.Pos)
	if !ok {
		t.Fatal("no range box while peeking")
	}
	st.OffsetX = 3
	peek()
	after, ok := rangeBoxAt(s, machinetest.Pos)
	if !ok || after[0]-before[0] != 3 || after[2] != before[2] {
		t.Fatalf("box at %v after moving the centre 3 east, was at %v", after, before)
	}
}
