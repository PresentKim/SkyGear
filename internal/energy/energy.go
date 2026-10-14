// Package energy moves RE (RedstoneEnergy) from blocks that give it to
// blocks that take it, over conduits (docs/design/06-power-and-conduits.md).
// It knows nothing of the game's blocks: a block takes part by implementing
// Source (a generator, a battery), Sink (a machine) or Conduit (a cable).
//
// The Grid is owned by one world (the island instance). Conduits and sources
// register themselves each time they tick; Step, called every few ticks,
// finds the networks of the conduits that registered lately and shares out
// the energy.
package energy

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// World is what the grid reads blocks from (*world.Tx).
type World interface {
	Block(pos cube.Pos) world.Block
}

// Source is a block that gives energy.
type Source interface {
	// ExtractEnergy takes up to max RE through face (the face of the block
	// the conduit or neighbour touches) and returns how much it took; with
	// simulate set it only says how much it could give.
	ExtractEnergy(w World, pos cube.Pos, face cube.Face, max int, simulate bool) int
}

// Sink is a block that takes energy.
type Sink interface {
	// ReceiveEnergy puts up to max RE in through face and returns how much it
	// took; with simulate set it only says how much it could take.
	ReceiveEnergy(w World, pos cube.Pos, face cube.Face, max int, simulate bool) int
}

// Ports is optionally implemented by a block whose Source and Sink methods
// are only some of the time meant (a machine type that is a generator or a
// consumer by kind): it says which of the two the block takes part as. A
// block without it takes part in every role it has methods for.
type Ports interface {
	EnergyPorts() (source, sink bool)
}

// RolesOf returns the roles of a block: the Source and the Sink it takes part
// as (nil for the roles it does not).
func RolesOf(b world.Block) (Source, Sink) {
	src, _ := b.(Source)
	snk, _ := b.(Sink)
	if p, ok := b.(Ports); ok {
		isSrc, isSnk := p.EnergyPorts()
		if !isSrc {
			src = nil
		}
		if !isSnk {
			snk = nil
		}
	}
	return src, snk
}

// Conduit is a cable block. EnergyTier is its tier (1 to 3).
type Conduit interface {
	EnergyTier() int
}

// Rates are the RE per tick a network of a tier carries; the tier of a
// network is its lowest. DirectRate is what a source gives a sink it touches
// without a conduit. Drafts.
var Rates = [...]int{0, 40, 240, 960}

// DirectRate is the RE per tick between touching blocks.
const DirectRate = 160

// RateOf is the rate of a tier (clamped to the known tiers).
func RateOf(tier int) int {
	return Rates[max(1, min(tier, len(Rates)-1))]
}

// Endpoint is a block next to a conduit (or a source): Face is the face of
// that block the conduit touches.
type Endpoint struct {
	Pos  cube.Pos
	Face cube.Face
}

// Network is a set of conduits that touch each other with the blocks that
// give and take energy on them.
type Network struct {
	Conduits []cube.Pos
	Sources  []Endpoint
	Sinks    []Endpoint
	// Rate is the RE per tick the network carries.
	Rate int
}

// IsConduit reports whether b is an energy conduit and of which tier.
func IsConduit(b world.Block) (int, bool) {
	c, ok := b.(Conduit)
	if !ok {
		return 0, false
	}
	return c.EnergyTier(), true
}

// Discover finds the network of the conduit at start by walking over the
// conduits that touch it, at most limit of them. Sources and sinks next to
// any of them are collected once each.
func Discover(w World, start cube.Pos, limit int) *Network {
	tier, ok := IsConduit(w.Block(start))
	if !ok {
		return nil
	}
	n := &Network{}
	seen := map[cube.Pos]bool{start: true}
	ends := map[Endpoint]bool{}
	queue := []cube.Pos{start}
	lowest := tier
	for len(queue) > 0 && len(n.Conduits) < limit {
		pos := queue[0]
		queue = queue[1:]
		n.Conduits = append(n.Conduits, pos)
		for _, f := range cube.Faces() {
			next := pos.Side(f)
			b := w.Block(next)
			if t, ok := IsConduit(b); ok {
				lowest = min(lowest, t)
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
				continue
			}
			e := Endpoint{Pos: next, Face: f.Opposite()}
			if ends[e] {
				continue
			}
			ends[e] = true
			src, snk := RolesOf(b)
			if src != nil {
				n.Sources = append(n.Sources, e)
			}
			if snk != nil {
				n.Sinks = append(n.Sinks, e)
			}
		}
	}
	n.Rate = RateOf(lowest)
	return n
}

// Transfer moves up to budget RE from the network's sources to its sinks and
// returns how much moved. Each sink gets its share of what is available, the
// ones that take less leaving more for the others.
func (n *Network) Transfer(w World, budget int) int {
	if budget <= 0 || len(n.Sources) == 0 || len(n.Sinks) == 0 {
		return 0
	}
	supply := 0
	for _, s := range n.Sources {
		supply += w.Block(s.Pos).(Source).ExtractEnergy(w, s.Pos, s.Face, budget, true)
	}
	want := make([]int, len(n.Sinks))
	demand := 0
	for i, k := range n.Sinks {
		want[i] = w.Block(k.Pos).(Sink).ReceiveEnergy(w, k.Pos, k.Face, budget, true)
		demand += want[i]
	}
	total := min(budget, supply, demand)
	if total <= 0 {
		return 0
	}
	// Water filling: the smallest wants are served in full, the rest share
	// what is left equally.
	give := make([]int, len(n.Sinks))
	left := total
	open := make([]int, 0, len(want))
	for i := range want {
		if want[i] > 0 {
			open = append(open, i)
		}
	}
	for len(open) > 0 && left > 0 {
		share := left / len(open)
		if share == 0 {
			// Fewer RE than sinks: the first sinks get one each.
			for _, i := range open[:min(left, len(open))] {
				give[i]++
			}
			left = 0
			break
		}
		next := open[:0:0]
		for _, i := range open {
			if want[i]-give[i] <= share {
				left -= want[i] - give[i]
				give[i] = want[i]
			} else {
				next = append(next, i)
			}
		}
		if len(next) == len(open) {
			for _, i := range open {
				give[i] += share
				left -= share
			}
			// The remainder goes one RE at a time to the first of them.
			for _, i := range open {
				if left == 0 {
					break
				}
				if give[i] < want[i] {
					give[i]++
					left--
				}
			}
			break
		}
		open = next
	}
	moved := 0
	for i, k := range n.Sinks {
		moved += w.Block(k.Pos).(Sink).ReceiveEnergy(w, k.Pos, k.Face, give[i], false)
	}
	// Take from the sources what the sinks took.
	left = moved
	for _, s := range n.Sources {
		if left == 0 {
			break
		}
		left -= w.Block(s.Pos).(Source).ExtractEnergy(w, s.Pos, s.Face, left, false)
	}
	return moved
}
