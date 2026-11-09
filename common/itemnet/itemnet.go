// Package itemnet moves items over networks of touching item conduits, as
// EnderIO does: each conduit side has a Mode and channels, and each step moves
// up to the tier's rate round robin, through itemio so any container works.
package itemnet

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/common/netgraph"
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

// Counters say how much topology work a grid did since it was made.
type Counters struct {
	// Scans is how many times a network was rebuilt because the world no
	// longer matched the graph.
	Scans int64
	// Visited is how many conduits the walks passed over in all.
	Visited int64
}

// verifySteps is how many Steps make about 60 seconds.
const verifySteps = 60 * 20 / StepTicks

// Grid steps the item networks of one world. It is changed by events: Place,
// Remove and Changed. A slow Verify catches what the events missed. Conduits
// carry no buffer, so every node of the graph has weight 0.
type Grid struct {
	graph  *netgraph.Graph
	tiers  map[cube.Pos]int
	nets   map[netgraph.NetID]*Network
	dirty  map[netgraph.NetID]bool
	credit int
	cursor netgraph.NetID
	// trace, when set by a test, collects every position the grid reads.
	trace map[cube.Pos]struct{}
}

// NewGrid returns an empty grid.
func NewGrid() *Grid {
	return &Grid{
		graph: netgraph.New(),
		tiers: map[cube.Pos]int{},
		nets:  map[netgraph.NetID]*Network{},
		dirty: map[netgraph.NetID]bool{},
	}
}

// block reads the block at pos for the grid.
func (g *Grid) block(tx *world.Tx, pos cube.Pos) world.Block {
	if g.trace != nil {
		g.trace[pos] = struct{}{}
	}
	return tx.Block(pos)
}

func (g *Grid) links(tx *world.Tx) netgraph.Links {
	return func(pos cube.Pos) []cube.Pos {
		var out []cube.Pos
		for _, f := range cube.Faces() {
			q := pos.Side(f)
			if _, ok := IsConduit(g.block(tx, q)); ok {
				out = append(out, q)
			}
		}
		return out
	}
}

// Conduits returns how many conduits the grid knows.
func (g *Grid) Conduits() int { return g.graph.Len() }

// Counters returns the topology work done so far.
func (g *Grid) Counters() Counters {
	return Counters{Scans: g.graph.Rebuilds(), Visited: g.graph.Visited()}
}

// forget drops the per-network state of a network that no longer exists.
func (g *Grid) forget(id netgraph.NetID) {
	delete(g.nets, id)
	delete(g.dirty, id)
}

// Place adds the conduit block at pos (already in the world).
func (g *Grid) Place(tx *world.Tx, pos cube.Pos) {
	c, ok := IsConduit(g.block(tx, pos))
	if !ok {
		return
	}
	_, known := g.graph.NetOf(pos)
	g.tiers[pos] = c.ItemTier()
	var old []netgraph.NetID
	seen := map[netgraph.NetID]bool{}
	for _, q := range append([]cube.Pos{pos}, g.links(tx)(pos)...) {
		if id, ok := g.graph.NetOf(q); ok && !seen[id] {
			seen[id] = true
			old = append(old, id)
		}
	}
	id := g.graph.Add(pos, 0, netgraph.Store{}, g.links(tx))
	for _, o := range old {
		if o != id {
			g.forget(o)
		}
	}
	if !known && len(old) == 1 && old[0] == id && g.cached(id) {
		g.addInPlace(tx, id, pos, c)
		return
	}
	g.dirty[id] = true
}

// cached reports whether network id has a clean cached Network.
func (g *Grid) cached(id netgraph.NetID) bool { return g.nets[id] != nil && !g.dirty[id] }

// addEnds appends the endpoints beside the conduit c at pos to n: the sides
// that are not Off and whose block has items.
func (g *Grid) addEnds(tx *world.Tx, n *Network, pos cube.Pos, c Conduit) {
	for _, f := range cube.Faces() {
		next := pos.Side(f)
		if _, ok := g.graph.NetOf(next); ok {
			continue
		}
		mode := c.ItemMode(f)
		if mode == Off {
			continue
		}
		if g.trace != nil {
			g.trace[next] = struct{}{}
		}
		if in, ex := itemio.IO(tx, next); in == nil && ex == nil {
			continue
		}
		ex, ins := c.ItemChannels(f)
		n.Ends = append(n.Ends, Endpoint{Pos: next, Face: f.Opposite(), Mode: mode, Extract: ex, Insert: ins})
	}
}

// addInPlace edits the cached network id for one conduit that joined it
// without merging: the conduit, its tier and the endpoints beside it. Turn,
// next and the order of the other endpoints stay.
func (g *Grid) addInPlace(tx *world.Tx, id netgraph.NetID, pos cube.Pos, c Conduit) {
	n := g.nets[id]
	n.Conduits = append(n.Conduits, pos)
	n.Tier = min(n.Tier, c.ItemTier())
	n.Ends = slices.DeleteFunc(n.Ends, func(e Endpoint) bool { return e.Pos == pos }) // a block with items stood here
	g.addEnds(tx, n, pos, c)
}

// removeInPlace edits the cached network id for one conduit that left it
// without cutting it. tier is the tier the conduit had.
func (g *Grid) removeInPlace(tx *world.Tx, id netgraph.NetID, pos cube.Pos, tier int) {
	n := g.nets[id]
	if i := slices.Index(n.Conduits, pos); i >= 0 {
		last := len(n.Conduits) - 1
		n.Conduits[i] = n.Conduits[last]
		n.Conduits = n.Conduits[:last]
	}
	n.Ends = slices.DeleteFunc(n.Ends, func(e Endpoint) bool {
		return e.Pos.Side(e.Face) == pos // the end the conduit at pos had
	})
	// the block now at pos may be an end of the conduits beside it
	if _, isC := IsConduit(g.block(tx, pos)); !isC {
		g.block(tx, pos)
		if in, ex := itemio.IO(tx, pos); in != nil || ex != nil {
			for _, f := range cube.Faces() {
				q := pos.Side(f)
				if nid, ok := g.graph.NetOf(q); !ok || nid != id {
					continue
				}
				c, ok := IsConduit(g.block(tx, q))
				if !ok {
					continue
				}
				side := f.Opposite() // the side of the conduit at q that touches pos
				mode := c.ItemMode(side)
				if mode == Off {
					continue
				}
				ex, ins := c.ItemChannels(side)
				n.Ends = append(n.Ends, Endpoint{Pos: pos, Face: f, Mode: mode, Extract: ex, Insert: ins})
			}
		}
	}
	if tier <= n.Tier {
		n.Tier = 0
		g.graph.Each(id, func(p cube.Pos) {
			if t := g.tiers[p]; n.Tier == 0 || t < n.Tier {
				n.Tier = t
			}
		})
	}
}

// Remove takes the conduit at pos (already gone from the world) out. Other
// networks are not touched; the pieces of a cut network read their blocks again.
func (g *Grid) Remove(tx *world.Tx, pos cube.Pos) {
	id, ok := g.graph.NetOf(pos)
	if !ok {
		return
	}
	tier := g.tiers[pos]
	_, parts := g.graph.Remove(pos, g.links(tx))
	delete(g.tiers, pos)
	if len(parts) == 0 {
		g.forget(id)
	}
	if len(parts) == 1 && parts[0] == id && g.cached(id) {
		g.removeInPlace(tx, id, pos, tier)
		return
	}
	for _, p := range parts {
		g.dirty[p] = true
	}
}

// Changed tells the grid that the block at pos changed. A conduit reads its
// sides (modes, channels) and tier again, a block that stopped being a conduit
// leaves, and for any other block the conduits that touch it read it again.
func (g *Grid) Changed(tx *world.Tx, pos cube.Pos) {
	_, isC := IsConduit(g.block(tx, pos))
	id, known := g.graph.NetOf(pos)
	switch {
	case known && isC:
		g.dirty[id] = true
		return
	case known:
		g.Remove(tx, pos)
		return
	case isC:
		g.Place(tx, pos)
		return
	}
	for _, f := range cube.Faces() {
		if nid, ok := g.graph.NetOf(pos.Side(f)); ok {
			g.dirty[nid] = true
		}
	}
}

// refresh reads the conduits and endpoints of network id again, keeping turn
// and next.
func (g *Grid) refresh(tx *world.Tx, id netgraph.NetID) *Network {
	n := g.nets[id]
	if n == nil {
		n = &Network{}
		g.nets[id] = n
	}
	n.Conduits, n.Ends, n.Tier = g.graph.Nodes(id), nil, 0
	for _, pos := range n.Conduits {
		c, ok := IsConduit(g.block(tx, pos))
		if !ok {
			continue
		}
		g.tiers[pos] = c.ItemTier()
		if n.Tier == 0 || c.ItemTier() < n.Tier {
			n.Tier = c.ItemTier()
		}
		g.addEnds(tx, n, pos, c)
	}
	delete(g.dirty, id)
	return n
}

// verify checks a slice of the networks against the world, about all of them
// in 60 seconds worth of Steps.
func (g *Grid) verify(tx *world.Tx) {
	total := g.graph.Len()
	if total == 0 {
		g.credit = 0
		return
	}
	g.credit += max(1, total/verifySteps)
	ids := g.graph.Nets()
	for tries := 0; tries < len(ids); tries++ {
		i := 0
		for i < len(ids) && ids[i] <= g.cursor {
			i++
		}
		if i == len(ids) {
			i = 0
		}
		id := ids[i]
		size := g.graph.Size(id)
		if g.credit < size {
			return
		}
		g.credit -= size
		g.cursor = id
		g.verifyNet(tx, id)
	}
}

func (g *Grid) verifyNet(tx *world.Tx, id netgraph.NetID) {
	before := g.graph.Nodes(id)
	changed := g.graph.Verify(id, g.links(tx), func(p cube.Pos) (int, bool) {
		c, ok := IsConduit(g.block(tx, p))
		if ok {
			g.tiers[p] = c.ItemTier()
		}
		return 0, ok
	})
	if !changed {
		return
	}
	g.forget(id)
	for _, p := range before {
		if nid, ok := g.graph.NetOf(p); ok {
			g.dirty[nid] = true
		} else {
			delete(g.tiers, p)
		}
	}
}

// Step lets each network move its items; call it every StepTicks ticks. It
// returns how many items moved.
func (g *Grid) Step(tx *world.Tx) int {
	g.verify(tx)
	moved := 0
	for _, id := range g.graph.Nets() {
		n := g.nets[id]
		if n == nil || g.dirty[id] {
			n = g.refresh(tx, id)
		}
		moved += n.Step(tx)
	}
	return moved
}
