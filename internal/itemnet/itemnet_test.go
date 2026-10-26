package itemnet

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// testConduit is a conduit block (a block entity, so the world keeps its
// fields).
type testConduit struct {
	tier    int
	modes   [6]Mode
	extract [6]int
	insert  [6]int
}

var testHash = block.NextHash()

func (testConduit) EncodeBlock() (string, map[string]any) { return "test:conduit", nil }
func (testConduit) Hash() (uint64, uint64)                { return testHash, 0 }
func (testConduit) Model() world.BlockModel               { return model.Solid{} }
func (testConduit) Properties() customblock.Properties    { return customblock.Properties{Cube: true} }
func (testConduit) EncodeNBT() map[string]any             { return map[string]any{"id": "TestConduit"} }
func (c testConduit) DecodeNBT(map[string]any) any        { return c }
func (c testConduit) ItemTier() int                       { return c.tier }
func (c testConduit) ItemMode(f cube.Face) Mode           { return c.modes[f] }
func (c testConduit) ItemChannels(f cube.Face) (int, int) { return c.extract[f], c.insert[f] }

func init() {
	world.RegisterBlock(testConduit{})
	world.DefaultBlockRegistry.Finalize()
}

func withWorld(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	w.Do(f)
}

func chestAt(tx *world.Tx, pos cube.Pos) block.Container {
	tx.SetBlock(pos, block.NewChest(), nil)
	return tx.Block(pos).(block.Container)
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

// TestItemsFlowBetweenChestsOverConduits: a chest beside the first conduit
// gives, a chest beside the last takes, at the rate of the tier per step.
func TestItemsFlowBetweenChestsOverConduits(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	for tier := 1; tier <= 3; tier++ {
		withWorld(t, func(tx *world.Tx) {
			from, to := cube.Pos{0, 64, 0}, cube.Pos{4, 64, 0}
			src := chestAt(tx, from)
			chestAt(tx, to)
			_, _ = src.Inventory(tx, from).AddItem(item.NewStack(block.Stone{}, 640))
			for x := 1; x <= 3; x++ {
				c := testConduit{tier: tier}
				if x == 1 {
					c.modes[cube.FaceWest] = Input // takes from the chest on its west
				}
				if x == 3 {
					c.modes[cube.FaceEast] = Output // gives to the chest on its east
				}
				tx.SetBlock(cube.Pos{x, 64, 0}, c, nil)
			}
			g := NewGrid()
			g.AddConduit(cube.Pos{1, 64, 0})
			if got := g.Step(tx); got != RateOf(tier) {
				t.Fatalf("tier %d moved %d per step, want %d", tier, got, RateOf(tier))
			}
			g.AddConduit(cube.Pos{1, 64, 0})
			g.Step(tx)
			if count(tx, to, stone) != 2*RateOf(tier) || count(tx, from, stone) != 640-2*RateOf(tier) {
				t.Errorf("tier %d: %d in the target, %d left", tier, count(tx, to, stone), count(tx, from, stone))
			}
		})
	}
}

func TestSlowestConduitSetsTheRate(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		chestAt(tx, cube.Pos{0, 64, 0})
		chestAt(tx, cube.Pos{3, 64, 0})
		inv := tx.Block(cube.Pos{0, 64, 0}).(block.Container).Inventory(tx, cube.Pos{0, 64, 0})
		_, _ = inv.AddItem(item.NewStack(block.Stone{}, 64))
		a := testConduit{tier: 3}
		a.modes[cube.FaceWest] = Input
		b := testConduit{tier: 1}
		c := testConduit{tier: 3}
		c.modes[cube.FaceEast] = Output
		tx.SetBlock(cube.Pos{1, 64, 0}, a, nil)
		tx.SetBlock(cube.Pos{2, 64, 0}, b, nil)
		n := Discover(tx, cube.Pos{1, 64, 0}, 10)
		if n.Tier != 1 || len(n.Conduits) != 2 {
			t.Fatalf("network %+v", n)
		}
	})
}

// TestModes: Off and Output sides give nothing, Input sides take nothing, Both
// does either; a chest is never given its own items.
func TestModes(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		left, right := cube.Pos{0, 64, 0}, cube.Pos{2, 64, 0}
		chestAt(tx, left)
		chestAt(tx, right)
		_, _ = tx.Block(left).(block.Container).Inventory(tx, left).AddItem(item.NewStack(block.Stone{}, 10))
		c := testConduit{tier: 3}
		c.modes[cube.FaceWest], c.modes[cube.FaceEast] = Output, Output // nobody gives
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		if got := Discover(tx, cube.Pos{1, 64, 0}, 10).Step(tx); got != 0 {
			t.Fatalf("moved %d with no giving side", got)
		}
		c.modes[cube.FaceWest], c.modes[cube.FaceEast] = Off, Both
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		n := Discover(tx, cube.Pos{1, 64, 0}, 10)
		if len(n.Ends) != 1 {
			t.Fatalf("an Off side is an endpoint: %+v", n.Ends)
		}
		if got := n.Step(tx); got != 0 {
			t.Fatalf("moved %d between one chest and itself", got)
		}
		c.modes[cube.FaceWest] = Both
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		if got := Discover(tx, cube.Pos{1, 64, 0}, 10).Step(tx); got != 10 {
			t.Fatalf("moved %d", got)
		}
		if count(tx, left, stone)+count(tx, right, stone) != 10 {
			t.Error("items were lost or made")
		}
	})
}

// TestDestinationsTakeTurns: with two chests to fill they take turns, each
// getting half of every step.
func TestDestinationsTakeTurns(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		src, a, b := cube.Pos{0, 64, 0}, cube.Pos{1, 64, -1}, cube.Pos{1, 64, 1}
		chestAt(tx, src)
		chestAt(tx, a)
		chestAt(tx, b)
		_, _ = tx.Block(src).(block.Container).Inventory(tx, src).AddItem(item.NewStack(block.Stone{}, 64))
		c := testConduit{tier: 1}
		c.modes[cube.FaceWest] = Input
		c.modes[cube.FaceNorth], c.modes[cube.FaceSouth] = Output, Output
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		g := NewGrid()
		g.AddConduit(cube.Pos{1, 64, 0})
		g.Step(tx)
		half := RateOf(1) / 2
		if count(tx, a, stone) != half || count(tx, b, stone) != half {
			t.Errorf("the chests got %d and %d, want %d each", count(tx, a, stone), count(tx, b, stone), half)
		}
	})
}

// TestChannelsKeepLinesApart: items taken on one channel go only to the sides
// that insert on that channel.
func TestChannelsKeepLinesApart(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		src, a, b := cube.Pos{0, 64, 0}, cube.Pos{1, 64, -1}, cube.Pos{1, 64, 1}
		chestAt(tx, src)
		chestAt(tx, a)
		chestAt(tx, b)
		_, _ = tx.Block(src).(block.Container).Inventory(tx, src).AddItem(item.NewStack(block.Stone{}, 640))
		c := testConduit{tier: 3}
		c.modes[cube.FaceWest] = Input
		c.extract[cube.FaceWest] = 5
		c.modes[cube.FaceNorth], c.modes[cube.FaceSouth] = Output, Output
		c.insert[cube.FaceNorth], c.insert[cube.FaceSouth] = 5, 7 // only the north chest listens on channel 5
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		g := NewGrid()
		for range 3 {
			g.AddConduit(cube.Pos{1, 64, 0})
			g.Step(tx)
		}
		if count(tx, a, stone) != 3*RateOf(3) || count(tx, b, stone) != 0 {
			t.Errorf("channel 5 chest got %d, channel 7 chest got %d", count(tx, a, stone), count(tx, b, stone))
		}
	})
}

// TestRoundRobinWithinAStep: a fast network hands out one item at a time to
// each destination in turn, not a whole step's worth to the first.
func TestRoundRobinWithinAStep(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		src, a, b := cube.Pos{0, 64, 0}, cube.Pos{1, 64, -1}, cube.Pos{1, 64, 1}
		chestAt(tx, src)
		chestAt(tx, a)
		chestAt(tx, b)
		_, _ = tx.Block(src).(block.Container).Inventory(tx, src).AddItem(item.NewStack(block.Stone{}, 640))
		c := testConduit{tier: 3}
		c.modes[cube.FaceWest] = Input
		c.modes[cube.FaceNorth], c.modes[cube.FaceSouth] = Output, Output
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		if got := Discover(tx, cube.Pos{1, 64, 0}, 10).Step(tx); got != RateOf(3) {
			t.Fatalf("moved %d, want %d", got, RateOf(3))
		}
		if count(tx, a, stone) != RateOf(3)/2 || count(tx, b, stone) != RateOf(3)/2 {
			t.Errorf("the chests got %d and %d, want %d each", count(tx, a, stone), count(tx, b, stone), RateOf(3)/2)
		}
	})
}
