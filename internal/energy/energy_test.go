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

// TestRateIsTheLowestTier: a network carries what its weakest conduit does,
// times the ticks.
func TestRateIsTheLowestTier(t *testing.T) {
	store, have := 1000000, 0
	w := fakeWorld{{0, 0, 0}: gen{store: &store}, {4, 0, 0}: machine{have: &have, room: 1000000}}
	line(w, 3, 2, 3)
	g := NewGrid()
	g.AddConduit(cube.Pos{1, 0, 0})
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

// TestRegistrationsExpire: a conduit that stopped ticking drops out.
func TestRegistrationsExpire(t *testing.T) {
	w := fakeWorld{{1, 0, 0}: cable{tier: 1}}
	g := NewGrid()
	g.AddConduit(cube.Pos{1, 0, 0})
	if st := g.Step(w, 4); st.Networks != 1 {
		t.Fatalf("stats %+v", st)
	}
	for range registrationTTL {
		g.Step(w, 4)
	}
	if st := g.Step(w, 4); st.Networks != 0 || len(g.conduits) != 0 {
		t.Errorf("an expired conduit stayed: %+v", st)
	}
}

func TestSeparateNetworksDoNotMix(t *testing.T) {
	s1, s2, h1, h2 := 1000, 0, 0, 0
	w := fakeWorld{
		{0, 0, 0}: gen{store: &s1}, {1, 0, 0}: cable{tier: 3}, {2, 0, 0}: machine{have: &h1, room: 1000},
		{0, 0, 5}: gen{store: &s2}, {1, 0, 5}: cable{tier: 3}, {2, 0, 5}: machine{have: &h2, room: 1000},
	}
	g := NewGrid()
	g.AddConduit(cube.Pos{1, 0, 0})
	g.AddConduit(cube.Pos{1, 0, 5})
	g.Step(w, 1)
	if h1 == 0 || h2 != 0 {
		t.Errorf("machines got %d and %d: the second network has an empty generator", h1, h2)
	}
}
