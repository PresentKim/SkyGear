package fluidnet

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
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
