// Package itemnet moves items over networks of touching item conduits, as
// EnderIO does: each conduit side has a Mode and channels, and each step moves
// up to the tier's rate round robin, through itemio so any container works.
package itemnet

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
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

// ModeOf is the mode that takes items from the block if extract, and gives
// items to it if insert.
func ModeOf(extract, insert bool) Mode {
	switch {
	case extract && insert:
		return Both
	case extract:
		return Input
	case insert:
		return Output
	}
	return Off
}

// Next is the mode after m when cycling through the modes.
func (m Mode) Next() Mode { return (m + 1) % 4 }

// Gives reports whether the conduit takes items out of the block (the block
// is a source); Takes whether it puts items into it.
func (m Mode) Gives() bool { return m == Both || m == Input }
func (m Mode) Takes() bool { return m == Both || m == Output }

// Rates are the items a network of a tier moves per step; a step is
// StepTicks ticks long. They move as many items a second as Ender IO 8.2's
// item conduits (32 per 20 ticks, 64 per 20 ticks, 64 per 10 ticks).
var Rates = [...]int{0, 32, 64, 128}

// StepTicks is the ticks between two steps of a network.
const StepTicks = 20

// RateOf is the rate of a tier (clamped to the known tiers).
func RateOf(tier int) int { return Rates[max(1, min(tier, len(Rates)-1))] }

// Channels is the number of channels (colours) an extract or insert side can
// be set to: items taken from a side go only to the sides that insert on the
// same channel, as in EnderIO, so one network can carry several lines.
const Channels = 16

// Conduit is an item conduit block: its tier (1 to 3), the mode of each side
// and the channels of its extract (the block is a source) and insert (the
// block is a destination) side.
type Conduit interface {
	ItemTier() int
	ItemMode(face cube.Face) Mode
	ItemChannels(face cube.Face) (extract, insert int)
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
	// Extract and Insert are the channels of taking items from the block and
	// giving items to it.
	Extract, Insert int
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
	// next is, for each source, where its round robin goes on among the
	// destinations.
	next map[endKey]int
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
			ex, ins := c.ItemChannels(f)
			n.Ends = append(n.Ends, Endpoint{Pos: next, Face: f.Opposite(), Mode: mode, Extract: ex, Insert: ins})
		}
	}
	return n
}

// Step moves up to the network's rate of items, one at a time round robin so
// destinations share evenly (EnderIO's default mode), and returns how many.
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
	if len(sources) == 0 || len(dests) == 0 {
		return 0
	}
	if n.next == nil {
		n.next = map[endKey]int{}
	}
	left := RateOf(n.Tier)
	moved := 0
	// a block that got items this step does not give them on in the same step
	received := map[cube.Pos]bool{}
	for left > 0 {
		progressed := false
		for i := range sources {
			if left <= 0 {
				break
			}
			src := sources[(n.turn+i)%len(sources)]
			if received[src.Pos] {
				continue
			}
			key := endKey{src.Pos, src.Face}
			for j := range dests {
				idx := (n.next[key] + j) % len(dests)
				dst := dests[idx]
				if dst.Pos == src.Pos || dst.Insert != src.Extract {
					continue
				}
				if itemio.Transfer(tx, src.Pos, src.Face, dst.Pos, dst.Face, 1, nil) > 0 {
					n.next[key] = idx + 1
					received[dst.Pos] = true
					left--
					moved++
					progressed = true
					break
				}
			}
		}
		if !progressed {
			break
		}
	}
	n.turn++
	return moved
}

// endKey identifies a source: the face of a block a conduit takes items from.
type endKey struct {
	pos  cube.Pos
	face cube.Face
}

// registrationTTL is how many steps a conduit stays in the grid without
// registering again.
const registrationTTL = 10

// Grid steps the item networks of one world.
type Grid struct {
	conduits map[cube.Pos]int64
	sigs     map[cube.Pos]uint64
	nets     map[cube.Pos]*Network
	step     int64
	// the nets are reused until the set of conduits changes, Invalidate is
	// called or rescanSteps Steps passed.
	dirty    bool
	scanned  int64
	counters Counters
}

// Counters say how much topology work a grid did since it was made.
type Counters struct {
	// Scans is how many times every network was found again.
	Scans int64
	// Visited is how many conduits those scans walked over in all.
	Visited int64
}

// Counters returns the topology work done so far.
func (g *Grid) Counters() Counters { return g.counters }

// rescanSteps is the most Steps the networks are reused for without a scan.
const rescanSteps = 20

// Invalidate makes the next Step find the networks again; call it when a
// block beside a conduit was placed, broken or changed.
func (g *Grid) Invalidate() { g.dirty = true }

func NewGrid() *Grid {
	return &Grid{conduits: map[cube.Pos]int64{}, sigs: map[cube.Pos]uint64{}, nets: map[cube.Pos]*Network{}}
}

// AddConduit registers the conduit at pos; call it whenever the block ticks.
func (g *Grid) AddConduit(pos cube.Pos) {
	if _, known := g.conduits[pos]; !known {
		g.dirty = true
	}
	g.conduits[pos] = g.step
}

// AddConduitSig is AddConduit for a conduit that also says a signature of its
// settings (modes, channels); a changed signature invalidates the networks.
func (g *Grid) AddConduitSig(pos cube.Pos, sig uint64) {
	if old, ok := g.sigs[pos]; ok && old != sig {
		g.dirty = true
	}
	g.sigs[pos] = sig
	g.AddConduit(pos)
}

// Conduits is how many conduits are registered.
func (g *Grid) Conduits() int { return len(g.conduits) }

// Step finds the networks of the registered conduits and lets each move its
// items; call it every StepTicks ticks. It returns how many items moved.
func (g *Grid) Step(tx *world.Tx) int {
	g.step++
	for pos, seen := range g.conduits {
		if g.step-seen > registrationTTL {
			delete(g.conduits, pos)
			delete(g.sigs, pos)
			g.dirty = true
		}
	}
	if !g.dirty && g.nets != nil && g.step-g.scanned < rescanSteps {
		moved := 0
		for _, n := range g.nets {
			moved += n.Step(tx)
		}
		return moved
	}
	g.dirty, g.scanned = false, g.step
	g.counters.Scans++
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
		g.counters.Visited += int64(len(n.Conduits))
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
			n.turn, n.next = old.turn, old.next
		}
		keep[low] = n
		moved += n.Step(tx)
	}
	g.nets = keep
	return moved
}
