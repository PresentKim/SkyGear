package fluidnet

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func line(w fakeWorld, g *Grid, from, to int) {
	for x := from; x <= to; x++ {
		w[cube.Pos{x, 0, 0}] = pipe{}
		g.Place(w, cube.Pos{x, 0, 0}, None, 0)
	}
}

func total(g *Grid) (sum int) {
	g.Shares(func(_ cube.Pos, _ Kind, mB int) { sum += mB })
	return
}

func TestBufferOf(t *testing.T) {
	if BufferOf(1) != 1000 || BufferOf(2) != 5000 || BufferOf(3) != 40000 {
		t.Errorf("%d %d %d", BufferOf(1), BufferOf(2), BufferOf(3))
	}
}

func TestSourceFillsBuffer(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 1, 2)
	w[cube.Pos{1, 0, 1}] = &tank{kind: Water, amount: 100000, cap: 100000, source: true}
	g.Changed(w, cube.Pos{1, 0, 1})
	for range 20 {
		g.Step(w)
	}
	k, a, c, ok := g.Buffer(cube.Pos{1, 0, 0})
	if k != Water || a != 2000 || c != 2000 || !ok {
		t.Errorf("buffer %v %d %d %v", k, a, c, ok)
	}
	if total(g) != 2000 {
		t.Errorf("shares %d", total(g))
	}
}

func TestBufferFeedsSink(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 1, 2)
	w[cube.Pos{1, 0, 1}] = &tank{cap: 100000, sink: true}
	g.Changed(w, cube.Pos{1, 0, 1})
	// put fluid in by re-placing with a share
	g.Place(w, cube.Pos{2, 0, 0}, Water, 1500)
	s := w[cube.Pos{1, 0, 1}].(*tank)
	g.Step(w)
	if s.amount != 1500 {
		t.Errorf("sink got %d, want 1500", s.amount)
	}
	if _, a, _, _ := g.Buffer(cube.Pos{1, 0, 0}); a != 0 {
		t.Errorf("buffer %d left", a)
	}
}

func TestBufferKeepsOneKind(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 1, 2)
	g.Place(w, cube.Pos{1, 0, 0}, Water, 500)
	lava := &tank{kind: Lava, amount: 5000, cap: 5000, source: true}
	w[cube.Pos{1, 0, 1}] = lava
	g.Changed(w, cube.Pos{1, 0, 1})
	g.Step(w)
	if k, a, _, _ := g.Buffer(cube.Pos{1, 0, 0}); k != Water || a != 500 || lava.amount != 5000 {
		t.Errorf("buffer %v %d, lava %d", k, a, lava.amount)
	}
	// a lava sink does not take from a water buffer
	sink := &tank{kind: Lava, amount: 1, cap: 5000, sink: true}
	w[cube.Pos{2, 0, 1}] = sink
	w[cube.Pos{1, 0, 1}] = block.Air{}
	g.Changed(w, cube.Pos{1, 0, 1})
	g.Changed(w, cube.Pos{2, 0, 1})
	g.Step(w)
	if sink.amount != 1 {
		t.Errorf("lava sink got water: %d", sink.amount)
	}
	if k, a, _, _ := g.Buffer(cube.Pos{1, 0, 0}); k != Water || a != 500 {
		t.Errorf("buffer %v %d", k, a)
	}
}

func TestMergeKindsLargerWins(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 0, 2) // three conduits: 3000 water
	g.Place(w, cube.Pos{0, 0, 0}, Water, 3000)
	for _, x := range []int{4} {
		w[cube.Pos{x, 0, 0}] = pipe{}
		g.Place(w, cube.Pos{x, 0, 0}, Lava, 1000)
	}
	w[cube.Pos{3, 0, 0}] = pipe{}
	g.Place(w, cube.Pos{3, 0, 0}, None, 0) // joins both
	k, a, _, _ := g.Buffer(cube.Pos{0, 0, 0})
	if a != 3000 || k != Water {
		t.Errorf("merged %v %d, want Water 3000", k, a)
	}
	if total(g) != 3000 {
		t.Errorf("shares %d", total(g))
	}
}

func TestSplitConservesFluid(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 0, 4)
	g.Place(w, cube.Pos{0, 0, 0}, Water, 1001)
	delete(w, cube.Pos{2, 0, 0})
	k, got := g.Remove(w, cube.Pos{2, 0, 0})
	rest := total(g)
	if k != Water || got+rest != 1001 {
		t.Errorf("removed %d + rest %d != 1001", got, rest)
	}
	if g.Conduits() != 4 {
		t.Errorf("%d conduits", g.Conduits())
	}
	_, x, _, _ := g.Buffer(cube.Pos{0, 0, 0})
	_, y, _, _ := g.Buffer(cube.Pos{4, 0, 0})
	if x+y+got != 1001 {
		t.Errorf("%d + %d + %d", x, y, got)
	}
}

func TestVerifyDropsVanishedLoneConduit(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 0, 0)
	delete(w, cube.Pos{0, 0, 0}) // no event
	for range verifySteps + 2 {
		g.Step(w)
	}
	if g.Conduits() != 0 {
		t.Errorf("%d conduits left", g.Conduits())
	}
}

func TestVerifyBudget(t *testing.T) {
	w, g := fakeWorld{}, NewGrid()
	line(w, g, 0, 299)
	before := g.Counters().Visited
	g.Step(w)
	if d := g.Counters().Visited - before; d != 0 {
		t.Errorf("an unchanged Step walked %d", d)
	}
}

// countWorld counts the distinct positions read.
type countWorld struct {
	fakeWorld
	seen map[cube.Pos]struct{}
}

func (c countWorld) Block(p cube.Pos) world.Block {
	c.seen[p] = struct{}{}
	return c.fakeWorld.Block(p)
}

// sameAsFresh checks that the one cached network of g is what Discover finds
// from start now: the same conduits, tier and endpoints (as sets).
func sameAsFresh(t *testing.T, w World, g *Grid, start cube.Pos) {
	t.Helper()
	fresh := Discover(w, start, 2048)
	if len(g.nets) != 1 {
		t.Fatalf("%d cached networks", len(g.nets))
	}
	for id, n := range g.nets {
		if g.dirty[id] {
			t.Fatalf("network %d is dirty, the edit should have been in place", id)
		}
		if n.Tier != fresh.Tier || len(n.Conduits) != len(fresh.Conduits) {
			t.Errorf("tier %d, %d conduits; fresh %d, %d", n.Tier, len(n.Conduits), fresh.Tier, len(fresh.Conduits))
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
				t.Errorf("endpoint %v twice", e)
			}
			ends[e] = true
		}
		want := map[Endpoint]bool{}
		for _, e := range fresh.Ends {
			want[e] = true
			if !ends[e] {
				t.Errorf("endpoint %v missing", e)
			}
		}
		for e := range ends {
			if !want[e] {
				t.Errorf("endpoint %v should not be there", e)
			}
		}
	}
}

// pipe3 is a tier 3 conduit.
type pipe3 struct{ block.Air }

func (pipe3) FluidTier() int { return 3 }

// TestPlaceBreakEditsInPlace: a conduit placed beside one network and broken
// again leaves the cached endpoints equal to a fresh Discover, keeps the turn,
// and the place, the break and the Step read at most 12 positions.
func TestPlaceBreakEditsInPlace(t *testing.T) {
	w := fakeWorld{}
	g := NewGrid()
	for x := 1; x <= 20; x++ {
		w[cube.Pos{x, 0, 0}] = pipe3{}
		g.Place(w, cube.Pos{x, 0, 0}, None, 0)
	}
	w[cube.Pos{0, 0, 0}] = &tank{cap: 1000, sink: true}
	w[cube.Pos{5, 1, 0}] = &tank{cap: 1000, source: true}
	w[cube.Pos{5, 0, 2}] = &tank{cap: 1000, sink: true}
	w[cube.Pos{21, 0, 0}] = &tank{cap: 1000, sink: true}
	for _, p := range []cube.Pos{{0, 0, 0}, {5, 1, 0}, {5, 0, 2}, {21, 0, 0}} {
		g.Changed(w, p)
	}
	start := cube.Pos{1, 0, 0}
	g.Step(w)
	var turn int
	for _, n := range g.nets {
		turn = n.turn
	}
	cw := countWorld{w, map[cube.Pos]struct{}{}}

	branch := cube.Pos{5, 0, 1} // a tier 1 branch beside a tank
	w[branch] = pipe{}
	g.Place(cw, branch, None, 0)
	g.Step(cw)
	sameAsFresh(t, w, g, start)
	for _, n := range g.nets {
		if n.Tier != 1 || n.turn != turn+1 {
			t.Errorf("tier %d turn %d, want 1 and %d", n.Tier, n.turn, turn+1)
		}
	}

	delete(w, branch)
	g.Remove(cw, branch)
	g.Step(cw)
	sameAsFresh(t, w, g, start)
	for _, n := range g.nets {
		if n.Tier != 3 {
			t.Errorf("tier %d after the branch is gone", n.Tier)
		}
	}
	if len(cw.seen) > 12 {
		t.Errorf("place, break and Step read %d positions, want at most 12", len(cw.seen))
	}

	// a tank replaces the last conduit: it is an endpoint; a conduit replaces it
	last := cube.Pos{20, 0, 0}
	w[last] = &tank{cap: 1000, sink: true}
	g.Remove(w, last)
	sameAsFresh(t, w, g, start)
	w[last] = pipe3{}
	g.Place(w, last, None, 0)
	sameAsFresh(t, w, g, start)
}
