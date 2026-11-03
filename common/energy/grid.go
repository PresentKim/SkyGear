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

// rescanSteps is the most Steps a discovered topology is reused for before it
// is found again, in case a change was not announced with Invalidate.
const rescanSteps = 20

// Grid shares out the energy of one world.
type Grid struct {
	conduits map[cube.Pos]int64
	sources  map[cube.Pos]int64
	// step counts the Steps; registrations carry the step they were made in.
	step int64
	// nets are the networks found by the last scan, reused until the set of
	// conduits changes, Invalidate is called or rescanSteps Steps passed.
	nets    []*Network
	dirty   bool
	scanned int64
}

func NewGrid() *Grid {
	return &Grid{conduits: map[cube.Pos]int64{}, sources: map[cube.Pos]int64{}}
}

// Conduits returns how many conduits are registered.
func (g *Grid) Conduits() int { return len(g.conduits) }

// AddConduit registers the conduit at pos; call it whenever the block ticks.
func (g *Grid) AddConduit(pos cube.Pos) {
	if _, known := g.conduits[pos]; !known {
		g.dirty = true
	}
	g.conduits[pos] = g.step
}

// Invalidate makes the next Step find the networks again; call it when a
// block beside a conduit was placed, broken or changed.
func (g *Grid) Invalidate() { g.dirty = true }

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
			g.dirty = true
		}
	}
	for pos, seen := range g.sources {
		if g.step-seen > registrationTTL {
			delete(g.sources, pos)
		}
	}
	if g.dirty || g.nets == nil || g.step-g.scanned >= rescanSteps {
		g.rescan(w)
	}
	for _, n := range g.nets {
		st.Networks++
		st.Conduits += len(n.Conduits)
		st.Moved += n.live(w).Transfer(w, n.Rate*ticks)
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

// live is n without the endpoints that stopped being a Source or a Sink since
// the scan that found them (the topology is cached between scans).
func (n *Network) live(w World) *Network {
	out := &Network{Conduits: n.Conduits, Rate: n.Rate}
	for _, e := range n.Sources {
		if _, ok := w.Block(e.Pos).(Source); ok {
			out.Sources = append(out.Sources, e)
		}
	}
	for _, e := range n.Sinks {
		if _, ok := w.Block(e.Pos).(Sink); ok {
			out.Sinks = append(out.Sinks, e)
		}
	}
	return out
}

// rescan finds the networks of the registered conduits.
func (g *Grid) rescan(w World) {
	g.dirty, g.scanned = false, g.step
	g.nets = g.nets[:0:0]
	if len(g.conduits) == 0 {
		g.nets = nil
		return
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
		g.nets = append(g.nets, n)
	}
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
