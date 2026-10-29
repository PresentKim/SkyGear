package itemio

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func withWorld(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	w.Do(f)
}

func count(tx *world.Tx, pos cube.Pos, it item.Stack) int {
	n := 0
	for _, st := range tx.Block(pos).(block.Container).Inventory(tx, pos).Slots() {
		if !st.Empty() && st.Comparable(it) {
			n += st.Count()
		}
	}
	return n
}

// TestTransferBetweenChests: items move up to the maximum, one kind at a
// time, and the want filter picks the kind.
func TestTransferBetweenChests(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		from, to := cube.Pos{0, 64, 0}, cube.Pos{0, 63, 0}
		tx.SetBlock(from, block.NewChest(), nil)
		tx.SetBlock(to, block.NewChest(), nil)
		stone, wheat := item.NewStack(block.Stone{}, 5), item.NewStack(item.Wheat{}, 4)
		inv := tx.Block(from).(block.Container).Inventory(tx, from)
		_, _ = inv.AddItem(stone)
		_, _ = inv.AddItem(wheat)
		if n := Transfer(tx, from, cube.FaceDown, to, cube.FaceUp, 2, nil); n != 2 {
			t.Fatalf("moved %d, want 2", n)
		}
		if n := Transfer(tx, from, cube.FaceDown, to, cube.FaceUp, 64, func(st item.Stack) bool { return st.Comparable(wheat) }); n != 4 {
			t.Fatalf("moved %d wheat, want 4", n)
		}
		if count(tx, to, stone) != 2 || count(tx, to, wheat) != 4 || count(tx, from, stone) != 3 || count(tx, from, wheat) != 0 {
			t.Errorf("stone %d→%d wheat %d→%d", count(tx, from, stone), count(tx, to, stone), count(tx, from, wheat), count(tx, to, wheat))
		}
	})
}

// TestTransferSkipsRefusedKinds: a kind the receiver refuses does not block
// the kinds behind it (a furnace's side takes only fuel).
func TestTransferSkipsRefusedKinds(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		from, to := cube.Pos{0, 64, 0}, cube.Pos{1, 64, 0}
		tx.SetBlock(from, block.NewChest(), nil)
		tx.SetBlock(to, block.NewFurnace(cube.North), nil)
		inv := tx.Block(from).(block.Container).Inventory(tx, from)
		_, _ = inv.AddItem(item.NewStack(block.Stone{}, 3))
		_, _ = inv.AddItem(item.NewStack(item.Charcoal{}, 2))
		if n := Transfer(tx, from, cube.FaceEast, to, cube.FaceWest, 8, nil); n != 2 {
			t.Fatalf("moved %d, want the 2 charcoal (the stone was refused and skipped)", n)
		}
		fuel, _ := tx.Block(to).(block.Container).Inventory(tx, to).Item(1)
		if fuel.Count() != 2 {
			t.Errorf("fuel slot %v", fuel)
		}
		if count(tx, from, item.NewStack(block.Stone{}, 1)) != 3 {
			t.Error("the stone left the chest")
		}
	})
}

// TestFurnaceTopFeedsInputSideFeedsFuel: the face the items come through
// picks the slot, as a hopper above or beside a furnace does.
func TestFurnaceTopFeedsInputSideFeedsFuel(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 64, 0}
		tx.SetBlock(pos, block.NewFurnace(cube.North), nil)
		if Insert(tx, pos, cube.FaceUp, item.NewStack(block.Stone{}, 2)) < 1 {
			t.Fatal("the top took nothing")
		}
		if Insert(tx, pos, cube.FaceNorth, item.NewStack(item.Charcoal{}, 1)) != 1 {
			t.Fatal("the side did not take fuel")
		}
		inv := tx.Block(pos).(block.Container).Inventory(tx, pos)
		if in, _ := inv.Item(0); in.Empty() {
			t.Error("the input slot is empty")
		}
		if fuel, _ := inv.Item(1); fuel.Empty() {
			t.Error("the fuel slot is empty")
		}
	})
}

// TestFurnaceGivesOutputBelowOnly: the product leaves through the bottom
// face only.
func TestFurnaceGivesOutputBelowOnly(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 64, 0}
		tx.SetBlock(pos, block.NewFurnace(cube.North), nil)
		inv := tx.Block(pos).(block.Container).Inventory(tx, pos)
		_ = inv.SetItem(0, item.NewStack(block.Stone{}, 4))
		_ = inv.SetItem(2, item.NewStack(item.IronIngot{}, 3))
		if !Peek(tx, pos, cube.FaceUp, nil).Empty() {
			t.Error("the top gave something")
		}
		got := Peek(tx, pos, cube.FaceDown, nil)
		if got.Empty() || got.Item() != (item.IronIngot{}) {
			t.Fatalf("the bottom peeked %v", got)
		}
		if out := Extract(tx, pos, cube.FaceDown, got, 2); out.Count() != 2 {
			t.Errorf("took %d, want 2", out.Count())
		}
		if left, _ := inv.Item(2); left.Count() != 1 {
			t.Errorf("%d ingots left, want 1", left.Count())
		}
		if in, _ := inv.Item(0); in.Count() != 4 {
			t.Error("the input slot was touched")
		}
	})
}

// fakeBox is a block-less Inserter/Extractor used through an Adapter.
type fakeBox struct{ held item.Stack }

func (f *fakeBox) InsertItems(_ *world.Tx, _ cube.Pos, _ cube.Face, st item.Stack) int {
	if !f.held.Empty() {
		return 0
	}
	f.held = st
	return st.Count()
}
func (f *fakeBox) PeekItems(_ *world.Tx, _ cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	if f.held.Empty() || (want != nil && !want(f.held)) {
		return item.Stack{}
	}
	return f.held
}
func (f *fakeBox) ExtractItems(_ *world.Tx, _ cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	if f.held.Empty() || !f.held.Comparable(st) {
		return item.Stack{}
	}
	n = min(n, f.held.Count())
	out := f.held.Grow(n - f.held.Count())
	f.held = f.held.Grow(-n)
	return out
}

// TestAdapterServesABlockWithoutMethods: a registered adapter makes a block
// that has no interface methods an item source and sink.
func TestAdapterServesABlockWithoutMethods(t *testing.T) {
	box := &fakeBox{}
	RegisterAdapter(func(_ *world.Tx, _ cube.Pos, b world.Block) (Inserter, Extractor) {
		if _, ok := b.(block.Glass); ok {
			return box, box
		}
		return nil, nil
	})
	t.Cleanup(func() { adapters = nil })
	withWorld(t, func(tx *world.Tx) {
		chest, glass := cube.Pos{0, 64, 0}, cube.Pos{0, 65, 0}
		tx.SetBlock(chest, block.NewChest(), nil)
		tx.SetBlock(glass, block.Glass{}, nil)
		_, _ = tx.Block(chest).(block.Container).Inventory(tx, chest).AddItem(item.NewStack(item.Wheat{}, 3))
		if n := Transfer(tx, chest, cube.FaceUp, glass, cube.FaceDown, 8, nil); n != 3 || box.held.Count() != 3 {
			t.Fatalf("moved %d into the adapter box (holds %d)", n, box.held.Count())
		}
		if n := Transfer(tx, glass, cube.FaceDown, chest, cube.FaceUp, 2, nil); n != 2 || box.held.Count() != 1 {
			t.Fatalf("moved %d back (box holds %d)", n, box.held.Count())
		}
	})
}
