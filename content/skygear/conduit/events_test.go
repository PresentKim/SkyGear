package conduit

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/content/skygear/network"
)

// TestPlaceBreakByAnyCause: a plain SetBlock puts a bundle in the grid and
// Air takes it out, with no player and no tick.
func TestPlaceBreakByAnyCause(t *testing.T) {
	s := refreshSession(t)
	pos := cube.Pos{3, 64, 3}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Bundle{Energy: 1, Item: 1, Fluid: 1}, nil)
		network.Energy(s).Step(tx, 4)
		network.Fluids(s).Step(tx)
		network.Items(s).Step(tx)
		if network.Energy(s).Conduits() != 1 || network.Fluids(s).Conduits() != 1 || network.Items(s).Conduits() != 1 {
			t.Fatalf("placed: %d %d %d", network.Energy(s).Conduits(), network.Fluids(s).Conduits(), network.Items(s).Conduits())
		}
		if _, v := LocalOf(s).Visuals[pos]; !v {
			t.Fatal("no visual after SetBlock")
		}
		tx.SetBlock(pos, block.Air{}, nil)
		network.Energy(s).Step(tx, 4)
		if network.Energy(s).Conduits() != 0 || network.Fluids(s).Conduits() != 0 || network.Items(s).Conduits() != 0 {
			t.Fatalf("broken: %d %d %d", network.Energy(s).Conduits(), network.Fluids(s).Conduits(), network.Items(s).Conduits())
		}
		if _, v := LocalOf(s).Visuals[pos]; v {
			t.Fatal("visual after break")
		}
	})
}

// TestConnectionChangeKeepsCellAndGrid: new connection bits do not place or
// remove, and keep the same cell.
func TestConnectionChangeKeepsCellAndGrid(t *testing.T) {
	s := refreshSession(t)
	pos := cube.Pos{3, 64, 3}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Bundle{Energy: 1, Buf: &BufCell{Energy: 150}}, nil)
		cell := tx.Block(pos).(Bundle).Buf
		b := tx.Block(pos).(Bundle)
		b.EConn = 3
		tx.SetBlock(pos, b, nil)
		if tx.Block(pos).(Bundle).Buf != cell {
			t.Fatal("cell changed")
		}
		if a, _, _ := network.Energy(s).Buffer(pos); a != 150 {
			t.Fatalf("buffer %d, want 150", a)
		}
	})
}

// TestCopiedBundleGetsOwnCell: a bundle value copied to a second position
// never shares the cell.
func TestCopiedBundleGetsOwnCell(t *testing.T) {
	s := refreshSession(t)
	p1, p2 := cube.Pos{3, 64, 3}, cube.Pos{8, 64, 8}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(p1, Bundle{Energy: 1, Buf: &BufCell{Energy: 100}}, nil)
		tx.SetBlock(p2, tx.Block(p1), nil)
		c1, c2 := tx.Block(p1).(Bundle).Buf, tx.Block(p2).(Bundle).Buf
		if c1 == nil || c2 == nil || c1 == c2 {
			t.Fatalf("cells %p %p", c1, c2)
		}
		if c2.Energy != 0 {
			t.Fatalf("copy kept %d", c2.Energy)
		}
	})
}

// TestNBTOldSaveZero: a save without EBuf, FKind, FBuf reads as 0.
func TestNBTOldSaveZero(t *testing.T) {
	b := Bundle{}.DecodeNBT(map[string]any{"Energy": int32(1)}).(Bundle)
	if b.Energy != 1 || b.Buf == nil || *b.Buf != (BufCell{}) {
		t.Fatalf("%+v %v", b, b.Buf)
	}
	b.Buf.Energy, b.Buf.FluidKind, b.Buf.Fluid = 7, 2, 9
	got := Bundle{}.DecodeNBT(b.EncodeNBT()).(Bundle)
	if *got.Buf != (BufCell{7, 2, 9}) {
		t.Fatalf("%+v", got.Buf)
	}
}

// TestUnloadReloadConserves: the buffer of a network survives its column
// going away and coming back.
func TestUnloadReloadConserves(t *testing.T) {
	s := refreshSession(t)
	s.World.Do(func(tx *world.Tx) {
		entities := map[cube.Pos]world.Block{}
		for x := 3; x < 6; x++ {
			pos := cube.Pos{x, 64, 3}
			tx.SetBlock(pos, Bundle{Energy: 1, Fluid: 1, Buf: &BufCell{Energy: 200, FluidKind: int(fluidnet.Water), Fluid: 300}}, nil)
			entities[pos] = tx.Block(pos)
		}
		if a, c, _ := network.Energy(s).Buffer(cube.Pos{4, 64, 3}); a != 600 || c != 600 {
			t.Fatalf("before: %d/%d", a, c)
		}
		UnloadColumn(s, tx, entities)
		if network.Energy(s).Conduits() != 0 {
			t.Fatal("still in the grid")
		}
		e, f := 0, 0
		for _, b := range entities {
			e += b.(Bundle).Buf.Energy
			f += b.(Bundle).Buf.Fluid
		}
		if e != 600 || f != 900 {
			t.Fatalf("written %d %d", e, f)
		}
		LoadColumn(s, tx, entities)
		if a, c, _ := network.Energy(s).Buffer(cube.Pos{4, 64, 3}); a != 600 || c != 600 {
			t.Fatalf("after: %d/%d", a, c)
		}
		if k, a, _, _ := network.Fluids(s).Buffer(cube.Pos{4, 64, 3}); k != fluidnet.Water || a != 900 {
			t.Fatalf("after fluid: %v %d", k, a)
		}
		if _, v := LocalOf(s).Visuals[cube.Pos{4, 64, 3}]; !v {
			t.Fatal("no visual after load")
		}
	})
}

// TestWriteSharesMarksChanged: WriteShares copies the grid shares to the cells.
func TestWriteSharesMarksChanged(t *testing.T) {
	s := refreshSession(t)
	s.World.Do(func(tx *world.Tx) {
		pos := cube.Pos{3, 64, 3}
		tx.SetBlock(pos, Bundle{Energy: 1, Buf: &BufCell{Energy: 50}}, nil)
		tx.Block(pos).(Bundle).Buf.Energy = 0
		WriteShares(s, tx)
		if got := tx.Block(pos).(Bundle).Buf.Energy; got != 50 {
			t.Fatalf("cell %d, want 50", got)
		}
	})
}

// TestNoTicker: a bundle is not ticked.
func TestNoTicker(t *testing.T) {
	if _, ok := any(Bundle{}).(world.TickerBlock); ok {
		t.Fatal("Bundle is a world.TickerBlock")
	}
}

// TestLoadColumnDoesNotLoadNeighbour: loading a column whose bundle sits on the
// chunk border reads no neighbour column, so it cannot cascade; when the
// neighbour column loads later, the arms connect on both sides.
func TestLoadColumnDoesNotLoadNeighbour(t *testing.T) {
	s := refreshSession(t)
	a, b := cube.Pos{1599, 64, 3}, cube.Pos{1600, 64, 3} // chunks 99 and 100
	east, west := FaceIndex(cube.FaceEast), FaceIndex(cube.FaceWest)
	s.World.Do(func(tx *world.Tx) {
		// A plain SetBlock of a conduit would read (and load) the neighbour, so
		// the bundle is stored empty and its kinds are set without the hook.
		tx.SetBlock(a, Bundle{}, nil)
		tx.SetBlockEntity(a, Bundle{Energy: 1, Buf: &BufCell{}})
		if _, loaded := tx.BlockLoaded(b); loaded {
			t.Fatal("neighbour column loaded before the test")
		}
		LoadColumn(s, tx, map[cube.Pos]world.Block{a: tx.Block(a)})
		if _, loaded := tx.BlockLoaded(b); loaded {
			t.Fatal("LoadColumn loaded the neighbour column")
		}
		if _, v := LocalOf(s).Visuals[a]; !v {
			t.Fatal("no visual after load")
		}

		tx.SetBlock(b, Bundle{}, nil)
		tx.SetBlockEntity(b, Bundle{Energy: 1, Buf: &BufCell{}})
		LoadColumn(s, tx, map[cube.Pos]world.Block{b: tx.Block(b)})
		if got := tx.Block(b).(Bundle).EConn; got&(1<<west) == 0 {
			t.Fatalf("new bundle EConn %b, want the west arm", got)
		}
		if got := tx.Block(a).(Bundle).EConn; got&(1<<east) == 0 {
			t.Fatalf("border bundle EConn %b, want the east arm", got)
		}
	})
}
