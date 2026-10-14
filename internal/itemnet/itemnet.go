// Package itemnet moves items over item conduits (docs/design/06-power-and-
// conduits.md §3.3), the way EnderIO's item conduits do: the conduits that
// touch each other form a network; every block with items next to it is an
// endpoint whose side of the conduit has a connection mode (both ways, only
// taken from, only given to, or off). Each step a network moves up to its
// tier's rate of items from the endpoints that give to the endpoints that
// take, in turn so every destination gets some. Items move through
// internal/itemio, so chests, furnaces, drawers and machines all work.
package itemnet

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/itemio"
)

// Mode is what a conduit does with the block at one of its sides.
type Mode byte

const (
	// Both takes items from the block and gives items to it.
	Both Mode = iota
	// Input only takes items from the block.
	Input
	// Output only gives items to the block.
	Output
	// Off does not connect to the block at all.
	Off
)

// Next is the mode after m when cycling through the modes.
func (m Mode) Next() Mode { return (m + 1) % 4 }

// Gives reports whether the conduit takes items out of the block (the block
// is a source); Takes whether it puts items into it.
func (m Mode) Gives() bool { return m == Both || m == Input }
func (m Mode) Takes() bool { return m == Both || m == Output }

// Rates are the items a network of a tier moves per step; a step is
// StepTicks ticks long. Drafts.
var Rates = [...]int{0, 1, 4, 16}

// StepTicks is the ticks between two steps of a network.
const StepTicks = 8

// RateOf is the rate of a tier (clamped to the known tiers).
func RateOf(tier int) int { return Rates[max(1, min(tier, len(Rates)-1))] }

// Conduit is an item conduit block: its tier (1 to 3) and the mode of each
// side.
type Conduit interface {
	ItemTier() int
	ItemMode(face cube.Face) Mode
}

// IsConduit returns the conduit b is, if it is one (tier 0 means the block
// holds no item conduit).
func IsConduit(b world.Block) (Conduit, bool) {
	c, ok := b.(Conduit)
	if !ok || c.ItemTier() <= 0 {
		return nil, false
	}
	return c, true
}

// Endpoint is a block with items next to a conduit: Face is the face of that
// block the conduit touches.
type Endpoint struct {
	Pos  cube.Pos
	Face cube.Face
	Mode Mode
}

// Network is a set of touching conduits and the blocks with items next to them.
type Network struct {
	Conduits []cube.Pos
	Ends     []Endpoint
	// Tier is the lowest tier among the conduits.
	Tier int
	// turn is where the next step starts handing out, so every destination
	// and source gets its turn.
	turn int
}

// Discover finds the network of the conduit at start, over at most limit
// conduits. A side whose mode is Off, or whose block has no items, is not an
// endpoint.
func Discover(tx *world.Tx, start cube.Pos, limit int) *Network {
	first, ok := IsConduit(tx.Block(start))
	if !ok {
		return nil
	}
	n := &Network{Tier: first.ItemTier()}
	seen := map[cube.Pos]bool{start: true}
	queue := []cube.Pos{start}
	for len(queue) > 0 && len(n.Conduits) < limit {
		pos := queue[0]
		queue = queue[1:]
		n.Conduits = append(n.Conduits, pos)
		c, _ := IsConduit(tx.Block(pos))
		for _, f := range cube.Faces() {
			next := pos.Side(f)
			if other, ok := IsConduit(tx.Block(next)); ok {
				n.Tier = min(n.Tier, other.ItemTier())
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
				continue
			}
			mode := c.ItemMode(f)
			if mode == Off {
				continue
			}
			if in, ex := itemio.IO(tx, next); in == nil && ex == nil {
				continue
			}
			n.Ends = append(n.Ends, Endpoint{Pos: next, Face: f.Opposite(), Mode: mode})
		}
	}
	return n
}

// Step moves up to the network's rate of items and returns how many moved:
// from each endpoint that gives, in turn, to the endpoints that take, in
// turn.
func (n *Network) Step(tx *world.Tx) int {
	var sources, dests []Endpoint
	for _, e := range n.Ends {
		if e.Mode.Gives() {
			sources = append(sources, e)
		}
		if e.Mode.Takes() {
			dests = append(dests, e)
		}
	}
	left := RateOf(n.Tier)
	moved := 0
	if len(sources) == 0 || len(dests) == 0 {
		return 0
	}
	for i := range sources {
		src := sources[(n.turn+i)%len(sources)]
		for j := range dests {
			if left <= 0 {
				break
			}
			dst := dests[(n.turn+j)%len(dests)]
			if dst.Pos == src.Pos {
				continue
			}
			got := itemio.Transfer(tx, src.Pos, src.Face, dst.Pos, dst.Face, left, nil)
			moved += got
			left -= got
		}
	}
	n.turn++
	return moved
}

// registrationTTL is how many steps a conduit stays in the grid without
// registering again.
const registrationTTL = 10

// Grid steps the item networks of one world.
type Grid struct {
	conduits map[cube.Pos]int64
	nets     map[cube.Pos]*Network
	step     int64
}

// NewGrid returns an empty grid.
func NewGrid() *Grid {
	return &Grid{conduits: map[cube.Pos]int64{}, nets: map[cube.Pos]*Network{}}
}

// AddConduit registers the conduit at pos; call it whenever the block ticks.
func (g *Grid) AddConduit(pos cube.Pos) { g.conduits[pos] = g.step }

// Conduits is how many conduits are registered.
func (g *Grid) Conduits() int { return len(g.conduits) }

// Step finds the networks of the registered conduits and lets each move its
// items; call it every StepTicks ticks. It returns how many items moved.
func (g *Grid) Step(tx *world.Tx) int {
	g.step++
	for pos, seen := range g.conduits {
		if g.step-seen > registrationTTL {
			delete(g.conduits, pos)
		}
	}
	done := map[cube.Pos]bool{}
	keep := map[cube.Pos]*Network{}
	moved := 0
	for pos := range g.conduits {
		if done[pos] {
			continue
		}
		n := Discover(tx, pos, 2048)
		if n == nil {
			delete(g.conduits, pos)
			continue
		}
		low := n.Conduits[0]
		for _, c := range n.Conduits {
			done[c] = true
			if c.Y() < low.Y() || (c.Y() == low.Y() && (c.X() < low.X() || (c.X() == low.X() && c.Z() < low.Z()))) {
				low = c
			}
		}
		// Keep the turn of a network between steps (keyed by its lowest
		// conduit).
		if old, ok := g.nets[low]; ok {
			n.turn = old.turn
		}
		keep[low] = n
		moved += n.Step(tx)
	}
	g.nets = keep
	return moved
}
