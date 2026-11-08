package netgraph

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// grid4 links a position to its four horizontal neighbours.
func grid4(p cube.Pos) []cube.Pos {
	return []cube.Pos{p.Side(cube.FaceEast), p.Side(cube.FaceWest), p.Side(cube.FaceSouth), p.Side(cube.FaceNorth)}
}

func line(n int) Links {
	return func(p cube.Pos) []cube.Pos {
		return []cube.Pos{{p[0] - 1, p[1], p[2]}, {p[0] + 1, p[1], p[2]}}
	}
}

func total(g *Graph) int {
	t := 0
	for _, id := range g.Nets() {
		t += g.Store(id).Amount
	}
	return t
}

func TestAddMergesShares(t *testing.T) {
	g := New()
	l := line(2)
	g.Add(cube.Pos{0, 0, 0}, 10, Store{0, 5}, l)
	id := g.Add(cube.Pos{1, 0, 0}, 10, Store{0, 7}, l)
	if len(g.Nets()) != 1 {
		t.Fatalf("nets = %d", len(g.Nets()))
	}
	if s := g.Store(id); s != (Store{0, 12}) {
		t.Fatalf("store = %+v", s)
	}
	if c := g.Capacity(id); c != 20 {
		t.Fatalf("cap = %d", c)
	}
	if g.Len() != 2 {
		t.Fatalf("len = %d", g.Len())
	}
}

func TestRemoveSplitExample(t *testing.T) {
	g := New()
	l := line(3)
	for i := 0; i < 3; i++ {
		g.Add(cube.Pos{i, 0, 0}, 10, Store{}, l)
	}
	id, _ := g.NetOf(cube.Pos{0, 0, 0})
	g.SetStore(id, Store{0, 30})
	share, parts := g.Remove(cube.Pos{1, 0, 0}, l)
	if share != (Store{0, 10}) {
		t.Fatalf("share = %+v", share)
	}
	if len(parts) != 2 {
		t.Fatalf("parts = %d", len(parts))
	}
	for _, p := range parts {
		if s := g.Store(p); s != (Store{0, 10}) {
			t.Fatalf("part store = %+v", s)
		}
	}
	if _, ok := g.NetOf(cube.Pos{1, 0, 0}); ok {
		t.Fatal("removed node still present")
	}
}

func TestMergeBridgeKeepsTag(t *testing.T) {
	g := New()
	l := line(3)
	g.Add(cube.Pos{0, 0, 0}, 1000, Store{1, 100}, l)
	g.Add(cube.Pos{2, 0, 0}, 1000, Store{2, 40}, l)
	id := g.Add(cube.Pos{1, 0, 0}, 1000, Store{}, l)
	if s := g.Store(id); s != (Store{1, 140}) {
		t.Fatalf("store = %+v", s)
	}
}

func TestMergeClampsAndTags(t *testing.T) {
	g := New()
	l := line(2)
	g.Add(cube.Pos{0, 0, 0}, 10, Store{0, 15}, l)
	id := g.Add(cube.Pos{1, 0, 0}, 10, Store{0, 15}, l)
	if s := g.Store(id); s != (Store{0, 20}) {
		t.Fatalf("clamp: %+v", s)
	}
	g = New()
	g.Add(cube.Pos{0, 0, 0}, 10, Store{1, 3}, l)
	id = g.Add(cube.Pos{1, 0, 0}, 10, Store{2, 9}, l)
	if s := g.Store(id); s != (Store{2, 12}) {
		t.Fatalf("tags: %+v", s)
	}
}

func TestSetStoreZeroTag(t *testing.T) {
	g := New()
	id := g.Add(cube.Pos{}, 10, Store{3, 4}, grid4)
	g.SetStore(id, Store{3, 0})
	if s := g.Store(id); s != (Store{}) {
		t.Fatalf("store = %+v", s)
	}
	g.SetStore(id, Store{3, 99})
	if s := g.Store(id); s != (Store{3, 10}) {
		t.Fatalf("store = %+v", s)
	}
	g.SetWeight(cube.Pos{}, 4)
	if s := g.Store(id); s != (Store{3, 4}) || g.Capacity(id) != 4 {
		t.Fatalf("after SetWeight %+v cap %d", s, g.Capacity(id))
	}
}

func TestSharesSumToStore(t *testing.T) {
	g := New()
	l := line(3)
	ws := []int{3, 5, 7}
	for i, w := range ws {
		g.Add(cube.Pos{i, 0, 0}, w, Store{}, l)
	}
	id, _ := g.NetOf(cube.Pos{0, 0, 0})
	g.SetStore(id, Store{4, 14})
	sum, n := 0, 0
	var first Store
	g.Shares(func(p cube.Pos, s Store) {
		if n == 0 {
			first = s
			if p != (cube.Pos{0, 0, 0}) {
				t.Errorf("first pos = %v", p)
			}
		}
		n++
		sum += s.Amount
		if s.Amount > 0 && s.Tag != 4 {
			t.Errorf("tag %d", s.Tag)
		}
	})
	// floor(14*3/15)=2, floor(14*5/15)=4, floor(14*7/15)=6; remainder 2 to first
	if n != 3 || sum != 14 || first.Amount != 4 {
		t.Fatalf("n=%d sum=%d first=%+v", n, sum, first)
	}
}

// tree builds a random tree of n nodes on the x axis positions with an adjacency map.
func tree(r *rand.Rand, n int) (map[cube.Pos][]cube.Pos, []cube.Pos) {
	adj := map[cube.Pos][]cube.Pos{}
	var order []cube.Pos
	for i := 0; i < n; i++ {
		p := cube.Pos{i, 0, 0}
		order = append(order, p)
		if i > 0 {
			q := order[r.Intn(i)]
			adj[p] = append(adj[p], q)
			adj[q] = append(adj[q], p)
		}
	}
	return adj, order
}

func TestRemoveSplitConserves(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	adj, order := tree(r, 200)
	links := func(p cube.Pos) []cube.Pos { return adj[p] }
	g := New()
	capSum := 0
	for _, p := range order {
		w := 1 + r.Intn(20)
		capSum += w
		g.Add(p, w, Store{}, links)
	}
	if len(g.Nets()) != 1 {
		t.Fatalf("nets = %d", len(g.Nets()))
	}
	id := g.Nets()[0]
	initial := capSum - 3
	g.SetStore(id, Store{7, initial})
	removed := 0
	for _, i := range r.Perm(len(order)) {
		share, _ := g.Remove(order[i], links)
		removed += share.Amount
		if got := total(g) + removed; got != initial {
			t.Fatalf("conservation broken: %d != %d", got, initial)
		}
		for _, nid := range g.Nets() {
			if s := g.Store(nid); s.Amount > g.Capacity(nid) || (s.Amount == 0 && s.Tag != 0) {
				t.Fatalf("bad store %+v cap %d", s, g.Capacity(nid))
			}
		}
	}
	if g.Len() != 0 || len(g.Nets()) != 0 {
		t.Fatal("graph not empty")
	}
}

func partition(g *Graph) map[cube.Pos]cube.Pos {
	// canonical representative: smallest pos of the net
	out := map[cube.Pos]cube.Pos{}
	for _, id := range g.Nets() {
		ns := g.Nodes(id)
		for _, p := range ns {
			out[p] = ns[0]
		}
	}
	return out
}

func bfsPartition(set map[cube.Pos]bool) map[cube.Pos]cube.Pos {
	out := map[cube.Pos]cube.Pos{}
	var keys []cube.Pos
	for p := range set {
		keys = append(keys, p)
	}
	sort.Slice(keys, func(i, j int) bool { return posLess(keys[i], keys[j]) })
	for _, k := range keys {
		if _, ok := out[k]; ok {
			continue
		}
		q := []cube.Pos{k}
		out[k] = k
		for len(q) > 0 {
			p := q[0]
			q = q[1:]
			for _, n := range grid4(p) {
				if set[n] {
					if _, ok := out[n]; !ok {
						out[n] = k
						q = append(q, n)
					}
				}
			}
		}
	}
	return out
}

func TestRandomOpsMatchPartition(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	g := New()
	set := map[cube.Pos]bool{}
	for i := 0; i < 500; i++ {
		p := cube.Pos{r.Intn(8), 0, r.Intn(8)}
		if set[p] && r.Intn(2) == 0 {
			delete(set, p)
			g.Remove(p, grid4)
		} else if !set[p] {
			set[p] = true
			g.Add(p, 5, Store{0, 1}, grid4)
		}
		if g.Len() != len(set) {
			t.Fatalf("len %d != %d", g.Len(), len(set))
		}
		got, want := partition(g), bfsPartition(set)
		if len(got) != len(want) {
			t.Fatalf("step %d size", i)
		}
		for p, rep := range want {
			if got[p] != rep {
				t.Fatalf("step %d: partition differs at %v", i, p)
			}
		}
		for _, id := range g.Nets() {
			if g.Capacity(id) != 5*len(g.Nodes(id)) {
				t.Fatalf("cap mismatch")
			}
		}
	}
}

func TestRemoveVisitsOnlyItsNet(t *testing.T) {
	g := New()
	l := func(p cube.Pos) []cube.Pos {
		return []cube.Pos{{p[0] - 1, p[1], p[2]}, {p[0] + 1, p[1], p[2]}}
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 64; x++ {
			g.Add(cube.Pos{x, y, 0}, 1, Store{}, l)
		}
	}
	if len(g.Nets()) != 16 {
		t.Fatalf("nets = %d", len(g.Nets()))
	}
	if g.Visited() != 0 {
		t.Fatalf("add visited %d", g.Visited())
	}
	g.Remove(cube.Pos{30, 5, 0}, l)
	if v := g.Visited(); v <= 0 || v > 64 {
		t.Fatalf("visited = %d", v)
	}
}

func TestVerify(t *testing.T) {
	g := New()
	l := line(4)
	for i := 0; i < 4; i++ {
		g.Add(cube.Pos{i, 0, 0}, 10, Store{}, l)
	}
	id, _ := g.NetOf(cube.Pos{0, 0, 0})
	g.SetStore(id, Store{0, 40})
	all := func(cube.Pos) (int, bool) { return 10, true }
	v0 := g.Visited()
	if g.Verify(id, l, all) {
		t.Fatal("unchanged net reported changed")
	}
	if g.Rebuilds() != 0 || g.Visited()-v0 != 4 {
		t.Fatalf("rebuilds %d visited %d", g.Rebuilds(), g.Visited()-v0)
	}
	// node 1 vanished: splits into {0} and {2,3}; its share (10) is dropped.
	gone := func(p cube.Pos) (int, bool) { return 10, p != (cube.Pos{1, 0, 0}) }
	if !g.Verify(id, l, gone) {
		t.Fatal("expected change")
	}
	if g.Rebuilds() != 1 || g.Len() != 3 || len(g.Nets()) != 2 || total(g) != 30 {
		t.Fatalf("rebuilds %d len %d nets %d total %d", g.Rebuilds(), g.Len(), len(g.Nets()), total(g))
	}
	// weight change
	n2, _ := g.NetOf(cube.Pos{2, 0, 0})
	heavy := func(cube.Pos) (int, bool) { return 20, true }
	if !g.Verify(n2, l, heavy) || g.Capacity(n2) != 40 {
		t.Fatalf("weight verify cap %d", g.Capacity(n2))
	}
}

func TestIncrementalEqualsFull(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	g := New()
	set := map[cube.Pos]bool{}
	for i := 0; i < 300; i++ {
		p := cube.Pos{r.Intn(7), 0, r.Intn(7)}
		if set[p] {
			delete(set, p)
			g.Remove(p, grid4)
		} else {
			set[p] = true
			g.Add(p, 1, Store{}, grid4)
		}
	}
	before := partition(g)
	for _, id := range g.Nets() {
		if g.Verify(id, grid4, func(p cube.Pos) (int, bool) { return 1, set[p] }) {
			t.Fatal("verify changed a consistent graph")
		}
	}
	after := partition(g)
	for p, rep := range before {
		if after[p] != rep {
			t.Fatal("verify altered partition")
		}
	}
}

func TestDeepLine(t *testing.T) {
	g := New()
	l := line(0)
	const n = 200000
	for i := 0; i < n; i++ {
		g.Add(cube.Pos{i, 0, 0}, 1, Store{}, l)
	}
	if len(g.Nets()) != 1 {
		t.Fatal("not one net")
	}
	g.Remove(cube.Pos{n / 2, 0, 0}, l)
	if len(g.Nets()) != 2 {
		t.Fatalf("nets = %d", len(g.Nets()))
	}
}
