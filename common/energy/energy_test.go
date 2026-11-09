package energy

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type fakeWorld map[cube.Pos]world.Block

func (f fakeWorld) Block(p cube.Pos) world.Block {
	if b, ok := f[p]; ok {
		return b
	}
	return block.Air{}
}

type cable struct {
	block.Stone
	tier int
}

func (c cable) EnergyTier() int { return c.tier }

// gen is a source with a store; the pointer is shared by the copies the
// world hands out.
type gen struct {
	block.Stone
	store *int
}

func (g gen) ExtractEnergy(_ World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	n := min(max, *g.store)
	if !simulate {
		*g.store -= n
	}
	return n
}

type machine struct {
	block.Stone
	have *int
	room int
}

func (m machine) ReceiveEnergy(_ World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	n := min(max, m.room-*m.have)
	if !simulate {
		*m.have += n
	}
	return max0(n)
}

func max0(n int) int { return max(n, 0) }

func line(w fakeWorld, tiers ...int) {
	for i, t := range tiers {
		w[cube.Pos{i + 1, 0, 0}] = cable{tier: t}
	}
}

// gridOf places every cable of w in a new grid.
func gridOf(w fakeWorld) *Grid {
	g := NewGrid()
	for p, b := range w {
		if _, ok := b.(cable); ok {
			g.Place(w, p, 0)
		}
	}
	return g
}

// TestRateIsTheLowestTier: a network carries what its weakest conduit does,
// times the ticks.
func TestRateIsTheLowestTier(t *testing.T) {
	store, have := 1000000, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {4, 0, 0}: machine{have: &have, room: 1000000}}
	line(w, 3, 2, 3)
	g := gridOf(w)
	st := g.Step(w, 4)
	if st.Networks != 1 || st.Conduits != 3 {
		t.Fatalf("stats %+v", st)
	}
	if have != RateOf(2)*4 || store != 1000000-RateOf(2)*4 {
		t.Errorf("moved %d, want %d (tier 2 for 4 ticks)", have, RateOf(2)*4)
	}
}

// TestSinksShareFairly: a small want is served in full, the others share the
// rest equally.
func TestSinksShareFairly(t *testing.T) {
	store := 1000
	a, b, c := 0, 0, 0
	w := fakeWorld{
		{0, 0, 0}:  gen{store: &store},
		{1, 0, 0}:  cable{tier: 3},
		{1, 1, 0}:  machine{have: &a, room: 10},
		{1, -1, 0}: machine{have: &b, room: 1000},
		{1, 0, 1}:  machine{have: &c, room: 1000},
	}
	n := Discover(w, cube.Pos{1, 0, 0}, 10)
	if len(n.Sources) != 1 || len(n.Sinks) != 3 {
		t.Fatalf("sources %d sinks %d", len(n.Sources), len(n.Sinks))
	}
	if moved := n.Transfer(w, 100); moved != 100 {
		t.Fatalf("moved %d", moved)
	}
	if a != 10 || b != 45 || c != 45 {
		t.Errorf("shares %d %d %d, want 10 45 45", a, b, c)
	}
	if store != 900 {
		t.Errorf("the source gave %d", 1000-store)
	}
}

// TestNothingFlowsWithoutBothEnds: no sink, no source, or empty ones move
// nothing and take nothing.
func TestNothingFlowsWithoutBothEnds(t *testing.T) {
	store, have := 50, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 1}}
	if moved := Discover(w, cube.Pos{1, 0, 0}, 10).Transfer(w, 100); moved != 0 || store != 50 {
		t.Errorf("moved %d without a sink", moved)
	}
	w[cube.Pos{2, 0, 0}] = machine{have: &have, room: 0}
	if moved := Discover(w, cube.Pos{1, 0, 0}, 10).Transfer(w, 100); moved != 0 {
		t.Errorf("moved %d into a full sink", moved)
	}
	w[cube.Pos{2, 0, 0}] = machine{have: &have, room: 1000}
	store = 0
	if moved := Discover(w, cube.Pos{1, 0, 0}, 10).Transfer(w, 100); moved != 0 {
		t.Errorf("moved %d from an empty source", moved)
	}
}

func TestTouchingSourceFeedsSinkWithoutConduit(t *testing.T) {
	store, have := 10000, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: machine{have: &have, room: 10000}}
	g := NewGrid()
	g.AddSource(cube.Pos{0, 0, 0})
	g.Step(w, 4)
	if have != DirectRate*4 {
		t.Errorf("moved %d, want %d", have, DirectRate*4)
	}
}

func TestSeparateNetworksDoNotMix(t *testing.T) {
	s1, s2, h1, h2 := 1000, 0, 0, 0
	w := fakeWorld{
		{0, 0, 0}: gen{store: &s1}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: machine{have: &h1, room: 1000},
		{0, 0, 5}: gen{store: &s2}, {1, 0, 5}: cable{tier: 3}, {2, 0, 5}: machine{have: &h2, room: 1000},
	}
	g := gridOf(w)
	g.Step(w, 1)
	if h1 == 0 || h2 != 0 {
		t.Errorf("machines got %d and %d: the second network has an empty generator", h1, h2)
	}
}

// TestPlacedMachineJoinsNextStep: a sink placed beside a cached network gets
// energy on the next Step after Changed.
func TestPlacedMachineJoinsNextStep(t *testing.T) {
	store, h1, h2 := 10000000, 0, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: machine{have: &h1, room: 1000000}}
	g := gridOf(w)
	g.Step(w, 1)
	w[cube.Pos{1, 0, 1}] = machine{have: &h2, room: 1000000}
	g.Changed(w, cube.Pos{1, 0, 1})
	g.Step(w, 1)
	if h2 == 0 {
		t.Error("the placed machine got nothing")
	}
}

// TestBrokenMachineLeavesNextStep: a removed sink takes nothing after Changed.
func TestBrokenMachineLeavesNextStep(t *testing.T) {
	store, h1, h2 := 10000000, 0, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: machine{have: &h1, room: 1000000}, {1, 0, 1}: machine{have: &h2, room: 1000000}}
	g := gridOf(w)
	g.Step(w, 1)
	if h2 == 0 {
		t.Fatal("setup: no energy")
	}
	delete(w, cube.Pos{1, 0, 1})
	g.Changed(w, cube.Pos{1, 0, 1})
	before := h2
	g.Step(w, 1)
	if h2 != before {
		t.Error("the broken machine still got energy")
	}
}

// TestStaleCacheNoPanic: a cached endpoint that is no longer a Sink or Source
// is skipped, without Changed.
func TestStaleCacheNoPanic(t *testing.T) {
	store, h1 := 10000000, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: machine{have: &h1, room: 1000000}}
	g := gridOf(w)
	g.Step(w, 1)
	w[cube.Pos{2, 0, 0}] = block.Stone{}
	w[cube.Pos{0, 0, 0}] = block.Stone{}
	g.Step(w, 1)
}

// TestCachedEqualsFresh: the endpoints and rate the grid keeps match a fresh
// discovery, also after a machine was placed and announced with Changed.
func TestCachedEqualsFresh(t *testing.T) {
	store, h1, h2 := 100000000, 0, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: cable{tier: 2}, {3, 0, 0}: machine{have: &h1, room: 10000000}}
	g := gridOf(w)
	g.Step(w, 1)
	check := func() {
		fresh := Discover(w, cube.Pos{1, 0, 0}, 2048)
		id, _ := g.graph.NetOf(cube.Pos{1, 0, 0})
		ni := g.info[id]
		ni.sort()
		if ni.count != len(fresh.Conduits) || len(ni.sinks) != len(fresh.Sinks) || len(ni.sources) != len(fresh.Sources) || ni.rate != fresh.Rate {
			t.Fatalf("cached %+v, fresh %+v", ni, fresh)
		}
	}
	check()
	w[cube.Pos{2, 0, 1}] = machine{have: &h2, room: 10000000}
	g.Changed(w, cube.Pos{2, 0, 1})
	check()
	g.Step(w, 1)
	if h2 == 0 {
		t.Error("the placed machine got nothing")
	}
}

// portsMachine is a sink whose ports can be switched off.
type portsMachine struct {
	machine
	sink *bool
}

func (p portsMachine) EnergyPorts() (bool, bool) { return false, *p.sink }

// TestCachedEndpointHonoursPorts: a cached sink whose ports closed since the
// scan gets nothing, without Invalidate.
func TestCachedEndpointHonoursPorts(t *testing.T) {
	store, h1 := 10000000, 0
	open := true
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: portsMachine{machine{have: &h1, room: 1000000}, &open}}
	g := gridOf(w)
	g.Step(w, 1)
	if h1 == 0 {
		t.Fatal("setup: no energy")
	}
	open = false
	before := h1
	g.Step(w, 1)
	if h1 != before {
		t.Error("a sink with closed ports still got energy")
	}
}
