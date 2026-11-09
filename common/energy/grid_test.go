package energy

import (
	"maps"
	"math/rand"
	"slices"
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

// F-T3-1: a Step on an unchanged network walks no conduits and keeps its
// endpoint info; the check against the world walks the network once a
// minute of Steps, not once per Step.
func TestStepOnUnchangedNetWalksOncePerMinute(t *testing.T) {
	store, have := 1<<30, 0
	w := buffered(64, &store, &have, 1<<30)
	g := gridOf(w)
	v0 := g.Counters().Visited
	g.Step(w, 4)
	ni := g.info[g.graph.Nets()[0]]
	for i := 2; i < verifyTicks/4; i++ {
		g.Step(w, 4)
		if v := g.Counters().Visited - v0; v != 0 {
			t.Fatalf("step %d walked %d conduits", i, v)
		}
	}
	g.Step(w, 4) // the minute is full: one check of the whole net
	if v := g.Counters().Visited - v0; v != 64 {
		t.Fatalf("minute walked %d conduits, want 64", v)
	}
	if g.info[g.graph.Nets()[0]] != ni {
		t.Fatal("endpoint info rebuilt for an unchanged net")
	}
}

// F-T3-1: the endpoint info edited in place by Place and Remove is what
// reading the network afresh gives. Other block changes drop the info of the
// networks beside them here, so only the in-place edits are compared.
func TestEditedInfoMatchesFresh(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	w := fakeWorld{}
	g := NewGrid()
	have := 0
	for op := 0; op < 4000; op++ {
		p := cube.Pos{r.Intn(6), r.Intn(2), r.Intn(6)}
		switch k := r.Intn(10); {
		case k < 6:
			w[p] = cable{tier: 1 + r.Intn(3)}
		case k == 6:
			w[p] = gen{store: &have}
		case k == 7:
			w[p] = machine{have: &have, room: 10}
		default:
			delete(w, p)
		}
		_, known := g.tiers[p]
		_, isC := IsConduit(w.Block(p))
		switch {
		case isC && !known:
			g.Place(w, p, 0)
		case isC:
			g.Changed(w, p) // tier change: rebuilt
		case known:
			g.Remove(w, p)
		default:
			for _, f := range cube.Faces() {
				if id, ok := g.graph.NetOf(p.Side(f)); ok {
					delete(g.info, id)
				}
			}
		}
		for _, id := range g.graph.Nets() {
			old := g.info[id]
			if old == nil {
				g.netInfoOf(w, id)
				continue
			}
			delete(g.info, id)
			fresh := g.netInfoOf(w, id)
			if old.count != fresh.count || old.lowest != fresh.lowest || old.rate != fresh.rate || len(old.ends) != len(fresh.ends) {
				t.Fatalf("op %d net %d: count %d/%d lowest %d/%d ends %d/%d", op, id, old.count, fresh.count, old.lowest, fresh.lowest, len(old.ends), len(fresh.ends))
			}
			for e, ro := range fresh.ends {
				if old.ends[e] != ro {
					t.Fatalf("op %d net %d: end %v %v vs %v", op, id, e, old.ends[e], ro)
				}
			}
			old.sort()
			fresh.sort()
			if !slices.Equal(old.sources, fresh.sources) || !slices.Equal(old.sinks, fresh.sinks) {
				t.Fatalf("op %d net %d: sorted ends differ", op, id)
			}
		}
	}
}

func endSet(s []Endpoint) map[Endpoint]bool {
	m := map[Endpoint]bool{}
	for _, e := range s {
		m[e] = true
	}
	return m
}

func TestChangedKeepsOneEndpoint(t *testing.T) {
	store, have := 1000, 0
	w := buffered(2, &store, nil, 0)
	g := gridOf(w)
	g.Step(w, 1) // reads the network info
	m := cube.Pos{3, 0, 0}
	w[m] = machine{have: &have, room: 1000}
	for range 3 {
		g.Changed(w, m)
	}
	id, _ := g.graph.NetOf(cube.Pos{1, 0, 0})
	ni := g.info[id]
	if ni == nil {
		t.Fatal("no info")
	}
	ni.sort()
	if len(ni.sinks) != 1 {
		t.Fatalf("sinks %v", ni.sinks)
	}
	fresh := Discover(w, cube.Pos{1, 0, 0}, 100)
	if a, b := endSet(ni.sinks), endSet(fresh.Sinks); !maps.Equal(a, b) {
		t.Errorf("sinks %v, fresh %v", a, b)
	}
	if a, b := endSet(ni.sources), endSet(fresh.Sources); !maps.Equal(a, b) {
		t.Errorf("sources %v, fresh %v", a, b)
	}
}
