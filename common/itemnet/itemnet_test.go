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
			for x := 1; x <= 3; x++ {
				g.Place(tx, cube.Pos{x, 64, 0})
			}
			if got := g.Step(tx); got != RateOf(tier) {
				t.Fatalf("tier %d moved %d per step, want %d", tier, got, RateOf(tier))
			}
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
		g.Place(tx, cube.Pos{1, 64, 0})
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
			g.Place(tx, cube.Pos{1, 64, 0})
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

func itemLine(tx *world.Tx) (from, to cube.Pos) {
	from, to = cube.Pos{0, 64, 0}, cube.Pos{2, 64, 0}
	src := chestAt(tx, from)
	_, _ = src.Inventory(tx, from).AddItem(item.NewStack(block.Stone{}, 640))
	c := testConduit{tier: 3}
	c.modes[cube.FaceWest] = Input
	c.modes[cube.FaceEast] = Output
	c.modes[cube.FaceUp] = Output
	tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
	return from, to
}

func TestPlacedChestJoinsNextStep(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		from, _ := itemLine(tx)
		_ = from
		g := NewGrid()
		g.Place(tx, cube.Pos{1, 64, 0})
		g.Step(tx)
		up := cube.Pos{1, 65, 0}
		chestAt(tx, up)
		g.Changed(tx, up)
		g.Step(tx)
		if count(tx, up, stone) == 0 {
			t.Error("the placed chest got nothing")
		}
	})
}

func TestBrokenChestLeavesNextStep(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		itemLine(tx)
		to := cube.Pos{2, 64, 0}
		chestAt(tx, to)
		g := NewGrid()
		g.Place(tx, cube.Pos{1, 64, 0})
		g.Step(tx)
		tx.SetBlock(to, block.Air{}, nil)
		g.Changed(tx, to)
		g.Step(tx)
	})
}

func TestStaleCacheNoPanic(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		itemLine(tx)
		to := cube.Pos{2, 64, 0}
		chestAt(tx, to)
		g := NewGrid()
		g.Place(tx, cube.Pos{1, 64, 0})
		g.Step(tx)
		tx.SetBlock(to, block.Stone{}, nil)
		tx.SetBlock(cube.Pos{0, 64, 0}, block.Stone{}, nil)
		g.Step(tx)
	})
}

func TestCachedEqualsFresh(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		itemLine(tx)
		chestAt(tx, cube.Pos{2, 64, 0})
		g := NewGrid()
		g.Place(tx, cube.Pos{1, 64, 0})
		g.Step(tx)
		fresh := Discover(tx, cube.Pos{1, 64, 0}, 2048)
		if len(g.nets) != 1 {
			t.Fatalf("%d cached networks", len(g.nets))
		}
		for _, n := range g.nets {
			if len(n.Conduits) != len(fresh.Conduits) || len(n.Ends) != len(fresh.Ends) {
				t.Errorf("cached %+v, fresh %+v", n, fresh)
			}
		}
	})
}

// TestModeChangeReadsSettings: a side set to Off and a Changed(conduit) later,
// nothing goes that way.
func TestModeChangeReadsSettings(t *testing.T) {
	stone := item.NewStack(block.Stone{}, 1)
	withWorld(t, func(tx *world.Tx) {
		itemLine(tx)
		to, up := cube.Pos{2, 64, 0}, cube.Pos{1, 65, 0}
		chestAt(tx, to)
		chestAt(tx, up)
		g := NewGrid()
		g.Place(tx, cube.Pos{1, 64, 0})
		g.Step(tx)
		if count(tx, up, stone) == 0 {
			t.Fatal("the up chest got nothing before the change")
		}
		c := testConduit{tier: 3}
		c.modes[cube.FaceWest] = Input
		c.modes[cube.FaceEast] = Output
		c.modes[cube.FaceUp] = Off
		tx.SetBlock(cube.Pos{1, 64, 0}, c, nil)
		g.Changed(tx, cube.Pos{1, 64, 0})
		before := count(tx, up, stone)
		g.Step(tx)
		if count(tx, up, stone) != before {
			t.Error("the Off side still got items")
		}
	})
}

// TestRemoveKeepsTurnOfOtherNets: removing a conduit of one network leaves the
// turn of another network alone.
func TestRemoveKeepsTurnOfOtherNets(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		g := NewGrid()
		a, b := cube.Pos{1, 64, 0}, cube.Pos{1, 64, 10}
		for _, p := range []cube.Pos{a, b, {1, 64, 11}} {
			tx.SetBlock(p, testConduit{tier: 1}, nil)
			g.Place(tx, p)
		}
		g.Step(tx)
		idB, _ := g.graph.NetOf(b)
		g.nets[idB].turn = 7
		tx.SetBlock(a, block.Air{}, nil)
		g.Remove(tx, a)
		g.Step(tx)
		if g.Conduits() != 2 || g.nets[idB].turn != 7 {
			t.Errorf("%d conduits, turn %d", g.Conduits(), g.nets[idB].turn)
		}
	})
}

func TestVerifyDropsVanishedLoneConduit(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		p := cube.Pos{1, 64, 0}
		tx.SetBlock(p, testConduit{tier: 1}, nil)
		g := NewGrid()
		g.Place(tx, p)
		tx.SetBlock(p, block.Air{}, nil) // no event
		for range verifySteps + 2 {
			g.Step(tx)
		}
		if g.Conduits() != 0 {
			t.Errorf("%d conduits left", g.Conduits())
		}
	})
}

// sameAsFresh checks that the one cached network of g is what Discover finds
// from start now: the same conduits, tier and endpoints (as sets).
func sameAsFresh(t *testing.T, tx *world.Tx, g *Grid, start cube.Pos) {
	t.Helper()
	fresh := Discover(tx, start, 2048)
	if len(g.nets) != 1 {
		t.Fatalf("%d cached networks", len(g.nets))
	}
	for id, n := range g.nets {
		if g.dirty[id] {
			t.Fatalf("network %d is dirty, the edit should have been in place", id)
		}
		if n.Tier != fresh.Tier {
			t.Errorf("tier %d, fresh %d", n.Tier, fresh.Tier)
		}
		if len(n.Conduits) != len(fresh.Conduits) {
			t.Errorf("%d conduits, fresh %d", len(n.Conduits), len(fresh.Conduits))
		}
		have := map[cube.Pos]bool{}
		for _, p := range n.Conduits {
			have[p] = true
		}
		for _, p := range fresh.Conduits {
			if !have[p] {
				t.Errorf("conduit %v missing", p)
			}
		}
		ends := map[Endpoint]bool{}
		for _, e := range n.Ends {
			if ends[e] {
				t.Errorf("endpoint %+v twice", e)
			}
			ends[e] = true
		}
		want := map[Endpoint]bool{}
		for _, e := range fresh.Ends {
			want[e] = true
			if !ends[e] {
				t.Errorf("endpoint %+v missing", e)
			}
		}
		for e := range ends {
			if !want[e] {
				t.Errorf("endpoint %+v should not be there", e)
			}
		}
	}
}

// TestPlaceBreakEditsInPlace: a conduit placed beside one network and broken
// again leaves the cached endpoints equal to a fresh Discover, keeps the round
// robin, and the place, the break and the Step read at most 12 positions.
func TestPlaceBreakEditsInPlace(t *testing.T) {
	withWorld(t, func(tx *world.Tx) {
		chestAt(tx, cube.Pos{0, 64, 0})
		chestAt(tx, cube.Pos{5, 65, 0})
		chestAt(tx, cube.Pos{5, 64, 2})
		chestAt(tx, cube.Pos{21, 64, 0})
		g := NewGrid()
		for x := 1; x <= 20; x++ {
			tx.SetBlock(cube.Pos{x, 64, 0}, testConduit{tier: 3}, nil)
			g.Place(tx, cube.Pos{x, 64, 0})
		}
		start := cube.Pos{1, 64, 0}
		g.Step(tx)
		var turn int
		for _, n := range g.nets {
			turn = n.turn
		}
		g.trace = map[cube.Pos]struct{}{}

		// a tier 1 branch beside a chest
		branch := cube.Pos{5, 64, 1}
		tx.SetBlock(branch, testConduit{tier: 1}, nil)
		g.Place(tx, branch)
		g.Step(tx)
		for _, n := range g.nets {
			if n.turn != turn+1 {
				t.Errorf("turn %d, want %d", n.turn, turn+1)
			}
		}
		sameAsFresh(t, tx, g, start)
		for _, n := range g.nets {
			if n.Tier != 1 {
				t.Errorf("tier %d after the tier 1 branch", n.Tier)
			}
		}

		// break it: the tier and the chest beside it go
		tx.SetBlock(branch, block.Air{}, nil)
		g.Remove(tx, branch)
		g.Step(tx)
		sameAsFresh(t, tx, g, start)
		for _, n := range g.nets {
			if n.Tier != 3 {
				t.Errorf("tier %d after the branch is gone", n.Tier)
			}
		}
		if len(g.trace) > 12 {
			t.Errorf("place and break read %d positions, want at most 12", len(g.trace))
		}

		// a chest replaces a conduit at the end of the line: it is an endpoint
		last := cube.Pos{20, 64, 0}
		chestAt(tx, last)
		g.Remove(tx, last)
		sameAsFresh(t, tx, g, start)
		// and a conduit replaces it
		tx.SetBlock(last, testConduit{tier: 3}, nil)
		g.Place(tx, last)
		sameAsFresh(t, tx, g, start)
	})
}
