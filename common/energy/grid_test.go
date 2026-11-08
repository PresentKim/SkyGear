package energy

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// buffered returns a world with n tier-1 conduits in a line from x=1, a
// generator at x=0 (if store is non-nil) and a machine after the last one.
func buffered(n int, store, have *int, room int) fakeWorld {
	w := fakeWorld{}
	for i := 1; i <= n; i++ {
		w[cube.Pos{i, 0, 0}] = cable{tier: 1}
	}
	if store != nil {
		w[cube.Pos{0, 0, 0}] = gen{store: store}
	}
	if have != nil {
		w[cube.Pos{n + 1, 0, 0}] = machine{have: have, room: room}
	}
	return w
}

func TestBufferOf(t *testing.T) {
	if BufferOf(1) != 200 || BufferOf(2) != 1200 || BufferOf(3) != 4800 {
		t.Fatalf("%d %d %d", BufferOf(1), BufferOf(2), BufferOf(3))
	}
}

func TestSourceFillsBuffer(t *testing.T) {
	store := 100000
	w := buffered(3, &store, nil, 0)
	g := gridOf(w)
	for range 3 {
		g.Step(w, 4)
	}
	if a, c, ok := g.Buffer(cube.Pos{1, 0, 0}); a != 600 || c != 600 || !ok {
		t.Fatalf("buffer %d %d %v", a, c, ok)
	}
	if store != 100000-600 {
		t.Errorf("source gave %d", 100000-store)
	}
}

func TestBufferFeedsSinkWithoutSource(t *testing.T) {
	have := 0
	w := buffered(3, nil, &have, 1000)
	g := NewGrid()
	for i := 1; i <= 3; i++ {
		g.Place(w, cube.Pos{i, 0, 0}, 0)
	}
	g.Place(w, cube.Pos{1, 0, 0}, 600) // placing again adds to the buffer
	if st := g.Step(w, 4); st.Moved != 600 {
		t.Fatalf("moved %d", st.Moved)
	}
	if a, _, _ := g.Buffer(cube.Pos{1, 0, 0}); a != 0 || have != 600 {
		t.Errorf("buffer %d, sink %d", a, have)
	}
}

// TestBufferNeverExceedsRate: throughput stays at rate x ticks both ways even
// with a full buffer and unlimited supply and demand.
func TestBufferNeverExceedsRate(t *testing.T) {
	store, have := 1000000, 0
	w := buffered(3, &store, &have, 1000000)
	g := NewGrid()
	for i := 1; i <= 3; i++ {
		g.Place(w, cube.Pos{i, 0, 0}, 0)
	}
	g.Place(w, cube.Pos{1, 0, 0}, 600) // placing again adds to the buffer
	st := g.Step(w, 4)
	if have > RateOf(1)*4 || st.Moved > RateOf(1)*4 {
		t.Fatalf("delivered %d / %d", have, st.Moved)
	}
	if 1000000-store > RateOf(1)*4 {
		t.Fatalf("took %d", 1000000-store)
	}
	// buffer not full: intake adds to what the direct transfer took
	store, have = 1000000, 0
	g = gridOf(w)
	g.Step(w, 4)
	if 1000000-store > RateOf(1)*4 {
		t.Fatalf("took %d with a free buffer", 1000000-store)
	}
}

// TestFullDemandSameAsBefore: with an empty buffer, a sink that takes all gets
// what the old direct transfer gave.
func TestFullDemandSameAsBefore(t *testing.T) {
	store, have := 1000000, 0
	w := buffered(3, &store, &have, 1000000)
	g := gridOf(w)
	g.Step(w, 4)
	if have != RateOf(1)*4 {
		t.Fatalf("sink got %d, want %d", have, RateOf(1)*4)
	}
	if a, _, _ := g.Buffer(cube.Pos{1, 0, 0}); a != 0 {
		t.Errorf("buffer %d", a)
	}
}

func TestChangedIgnoresSameRole(t *testing.T) {
	have := 0
	w := buffered(3, nil, &have, 1000)
	g := gridOf(w)
	g.Step(w, 1)
	v := g.Counters().Visited
	m := cube.Pos{4, 0, 0}
	w[m] = machine{have: &have, room: 500} // same role, other state
	g.Changed(w, m)
	if got := g.Counters().Visited - v; got != 0 {
		t.Fatalf("visited %d", got)
	}
	if g.Counters().Scans != 0 {
		t.Fatal("scan happened")
	}
}

func TestChangedAddsAndDropsEndpoint(t *testing.T) {
	store, have := 1000000, 0
	w := buffered(3, &store, nil, 0)
	g := gridOf(w)
	g.Step(w, 1)
	w[cube.Pos{4, 0, 0}] = machine{have: &have, room: 1000}
	g.Changed(w, cube.Pos{4, 0, 0})
	g.Step(w, 1)
	if have == 0 {
		t.Fatal("new sink got nothing")
	}
	delete(w, cube.Pos{4, 0, 0})
	g.Changed(w, cube.Pos{4, 0, 0})
	before := have
	g.Step(w, 1)
	if have != before {
		t.Fatal("removed sink still fed")
	}
}

func TestSplitShares(t *testing.T) {
	w := buffered(3, nil, nil, 0)
	g := NewGrid()
	for i := 1; i <= 3; i++ {
		g.Place(w, cube.Pos{i, 0, 0}, 0)
	}
	g.Place(w, cube.Pos{1, 0, 0}, 600) // placing again adds to the buffer
	delete(w, cube.Pos{2, 0, 0})
	if got := g.Remove(w, cube.Pos{2, 0, 0}); got != 200 {
		t.Fatalf("returned %d", got)
	}
	a, _, _ := g.Buffer(cube.Pos{1, 0, 0})
	b, _, _ := g.Buffer(cube.Pos{3, 0, 0})
	if a != 200 || b != 200 {
		t.Fatalf("pieces %d %d", a, b)
	}
}

func TestMergeConserves(t *testing.T) {
	w := fakeWorld{{1, 0, 0}: cable{tier: 1}, {3, 0, 0}: cable{tier: 1}, {2, 0, 0}: cable{tier: 1}}
	g := NewGrid()
	g.Place(w, cube.Pos{1, 0, 0}, 100)
	g.Place(w, cube.Pos{3, 0, 0}, 70)
	g.Place(w, cube.Pos{2, 0, 0}, 0)
	if a, c, _ := g.Buffer(cube.Pos{1, 0, 0}); a != 170 || c != 600 {
		t.Fatalf("merged %d/%d", a, c)
	}
	sum := 0
	g.Shares(func(_ cube.Pos, s int) { sum += s })
	if sum != 170 {
		t.Fatalf("shares %d", sum)
	}
}

func TestVerifyDropsVanishedConduit(t *testing.T) {
	w := fakeWorld{{1, 0, 0}: cable{tier: 1}, {9, 0, 9}: cable{tier: 1}}
	g := gridOf(w)
	delete(w, cube.Pos{9, 0, 9}) // gone without an event
	for range 300 {
		g.Step(w, 4)
	}
	if g.Conduits() != 1 {
		t.Fatalf("conduits %d", g.Conduits())
	}
	if _, _, ok := g.Buffer(cube.Pos{9, 0, 9}); ok {
		t.Fatal("vanished conduit still in grid")
	}
}

func TestVerifySplitsCutNet(t *testing.T) {
	w := buffered(3, nil, nil, 0)
	g := gridOf(w)
	delete(w, cube.Pos{2, 0, 0})
	for range 300 {
		g.Step(w, 4)
	}
	if g.Conduits() != 2 || len(g.graph.Nets()) != 2 {
		t.Fatalf("conduits %d nets %d", g.Conduits(), len(g.graph.Nets()))
	}
}
