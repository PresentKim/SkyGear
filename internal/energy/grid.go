package energy

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

// registrationTTL is how many steps a registration lives: a conduit or a
// source that has not registered for that long (its chunk is not loaded, or
// it is gone) drops out of the grid.
const registrationTTL = 10

// networkLimit is the most conduits one network is followed over.
const networkLimit = 2048

// Grid shares out the energy of one world.
type Grid struct {
	conduits map[cube.Pos]int64
	sources  map[cube.Pos]int64
	// step counts the Steps; registrations carry the step they were made in.
	step int64
}

// NewGrid returns an empty grid.
func NewGrid() *Grid {
	return &Grid{conduits: map[cube.Pos]int64{}, sources: map[cube.Pos]int64{}}
}

// AddConduit registers the conduit at pos; call it whenever the block ticks.
// Conduits returns how many conduits are registered.
func (g *Grid) Conduits() int { return len(g.conduits) }

func (g *Grid) AddConduit(pos cube.Pos) { g.conduits[pos] = g.step }

// AddSource registers a source that touches sinks without a conduit (a
// generator next to a machine); a source next to a conduit needs no
// registration.
func (g *Grid) AddSource(pos cube.Pos) { g.sources[pos] = g.step }

// Stats say what a Step found.
type Stats struct {
	Networks, Conduits int
	Moved              int
}

// Step moves the energy of ticks ticks: every network of the registered
// conduits carries up to its rate × ticks, and every registered source gives
// the sinks it touches up to DirectRate × ticks.
func (g *Grid) Step(w World, ticks int) Stats {
	var st Stats
	g.step++
	for pos, seen := range g.conduits {
		if g.step-seen > registrationTTL {
			delete(g.conduits, pos)
		}
	}
	for pos, seen := range g.sources {
		if g.step-seen > registrationTTL {
			delete(g.sources, pos)
		}
	}
	done := map[cube.Pos]bool{}
	for pos := range g.conduits {
		if done[pos] {
			continue
		}
		n := Discover(w, pos, networkLimit)
		if n == nil {
			delete(g.conduits, pos)
			continue
		}
		for _, c := range n.Conduits {
			done[c] = true
		}
		st.Networks++
		st.Conduits += len(n.Conduits)
		st.Moved += n.Transfer(w, n.Rate*ticks)
	}
	for pos := range g.sources {
		if src, _ := RolesOf(w.Block(pos)); src == nil {
			delete(g.sources, pos)
			continue
		}
		n := &Network{Rate: DirectRate}
		for _, f := range cube.Faces() {
			next := pos.Side(f)
			b := w.Block(next)
			if _, isConduit := IsConduit(b); isConduit {
				continue
			}
			if _, snk := RolesOf(b); snk != nil {
				n.Sinks = append(n.Sinks, Endpoint{Pos: next, Face: f.Opposite()})
			}
		}
		st.Moved += n.transferDirect(w, pos, DirectRate*ticks)
	}
	return st
}

// transferDirect moves energy from the source at pos to the sinks of n.
func (n *Network) transferDirect(w World, pos cube.Pos, budget int) int {
	if len(n.Sinks) == 0 {
		return 0
	}
	// Each sink is reached through the source's face that touches it.
	moved := 0
	left := budget
	for _, k := range n.Sinks {
		face := k.Face.Opposite()
		src := Endpoint{Pos: pos, Face: face}
		one := &Network{Sources: []Endpoint{src}, Sinks: []Endpoint{k}}
		got := one.Transfer(w, left)
		moved += got
		left -= got
	}
	return moved
}
