// Package fluidnet moves fluids over networks of touching fluid conduits,
// round robin from the tanks that give (sources) to those that take (sinks),
// up to the network tier's rate. Blocks take part via Holder and Conduit.
package fluidnet

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"

	"github.com/presentkim/skygear/common/netgraph"
)

// Kind is a kind of fluid.
type Kind byte

const (
	None Kind = iota
	Water
	Lava
)

// World is what the grid reads blocks from (*world.Tx).
type World interface {
	Block(pos cube.Pos) world.Block
}

// Holder is a block with a tank.
type Holder interface {
	// FluidRoles says whether the block gives fluid (source) and takes it (sink).
	FluidRoles() (source, sink bool)
	// FillFluid puts up to mB of a fluid in and returns how much it took; with
	// simulate set it only says how much it would take.
	FillFluid(w World, pos cube.Pos, k Kind, mB int, simulate bool) int
	// DrainFluid takes up to max mB out and returns the fluid and how much it
	// took (a tank gives one kind of fluid at a time); with simulate set it only
	// says what it would give.
	DrainFluid(w World, pos cube.Pos, max int, simulate bool) (Kind, int)
}

// Conduit is a cable block; FluidTier is its tier (1 to 3; 0 means the block
// holds no fluid conduit).
type Conduit interface {
	FluidTier() int
}

// Rates are the mB per tick a network of a tier carries (EnderIO's fluid
// conduits).
var Rates = [...]int{0, 200, 1000, 8000}

// StepTicks is the ticks between two steps of a network.
const StepTicks = 8

// RateOf is the rate of a tier (clamped to the known tiers).
func RateOf(tier int) int { return Rates[max(1, min(tier, len(Rates)-1))] }

// IsConduit returns the tier of b if it is a fluid conduit.
func IsConduit(b world.Block) (int, bool) {
	c, ok := b.(Conduit)
	if !ok || c.FluidTier() <= 0 {
		return 0, false
	}
	return c.FluidTier(), true
}

// Endpoint is a block with a tank next to a conduit.
type Endpoint struct {
	Pos cube.Pos
}

// Network is a set of touching conduits and the tanks next to them.
type Network struct {
	Conduits []cube.Pos
	Ends     []Endpoint
	// Tier is the lowest tier among the conduits.
	Tier int
	turn int
}

// Discover finds the network of the conduit at start, over at most limit
// conduits.
func Discover(w World, start cube.Pos, limit int) *Network {
	tier, ok := IsConduit(w.Block(start))
	if !ok {
		return nil
	}
	n := &Network{Tier: tier}
	seen := map[cube.Pos]bool{start: true}
	ends := map[cube.Pos]bool{}
	queue := []cube.Pos{start}
	for len(queue) > 0 && len(n.Conduits) < limit {
		pos := queue[0]
		queue = queue[1:]
		n.Conduits = append(n.Conduits, pos)
		for _, f := range cube.Faces() {
			next := pos.Side(f)
			b := w.Block(next)
			if t, ok := IsConduit(b); ok {
				n.Tier = min(n.Tier, t)
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
				continue
			}
			if _, ok := b.(Holder); ok && !ends[next] {
				ends[next] = true
				n.Ends = append(n.Ends, Endpoint{Pos: next})
			}
		}
	}
	return n
}

// Step moves up to the network's budget (rate × StepTicks) of mB and returns how
// much moved: every source in turn gives to the sinks in turn. It does not use
// a buffer; a Grid adds the shared buffer around the same transfer.
func (n *Network) Step(w World) int {
	moved := n.direct(w, RateOf(n.Tier)*StepTicks)
	n.turn++
	return moved
}

// roles splits the endpoints into sources and sinks.
func (n *Network) roles(w World) (sources, sinks []Endpoint) {
	for _, e := range n.Ends {
		h, ok := w.Block(e.Pos).(Holder)
		if !ok {
			continue
		}
		src, snk := h.FluidRoles()
		if src {
			sources = append(sources, e)
		}
		if snk {
			sinks = append(sinks, e)
		}
	}
	return sources, sinks
}

// direct moves at most budget mB from the sources to the sinks.
func (n *Network) direct(w World, budget int) int {
	sources, sinks := n.roles(w)
	if len(sources) == 0 || len(sinks) == 0 {
		return 0
	}
	left := budget
	// a pass gives every source's turn a share to every sink; passes go on until
	// the budget is spent or nothing moves
	share := max(1, left/(len(sources)*len(sinks)))
	moved := 0
	for left > 0 {
		progressed := false
		for i := range sources {
			src := sources[(n.turn+i)%len(sources)]
			sh, ok := w.Block(src.Pos).(Holder)
			if !ok {
				continue
			}
			for j := range sinks {
				if left <= 0 {
					break
				}
				dst := sinks[(n.turn+j)%len(sinks)]
				if dst.Pos == src.Pos {
					continue
				}
				k, have := sh.DrainFluid(w, src.Pos, min(share, left), true)
				if have == 0 {
					break
				}
				dh, ok := w.Block(dst.Pos).(Holder)
				if !ok {
					continue
				}
				room := dh.FillFluid(w, dst.Pos, k, have, true)
				if room == 0 {
					continue
				}
				k, took := sh.DrainFluid(w, src.Pos, min(have, room), false)
				put := dh.FillFluid(w, dst.Pos, k, took, false)
				if put > 0 {
					moved += put
					left -= put
					progressed = true
				}
			}
		}
		if !progressed {
			break
		}
	}
	return moved
}

// BufferOf is the buffer one conduit of a tier adds to its network (mB).
func BufferOf(tier int) int { return RateOf(tier) * 5 }

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

// Grid steps the fluid networks of one world. It is changed by events: Place,
// Remove and Changed. A slow Verify catches what the events missed.
type Grid struct {
	graph  *netgraph.Graph
	tiers  map[cube.Pos]int
	nets   map[netgraph.NetID]*Network
	dirty  map[netgraph.NetID]bool
	credit int
	cursor netgraph.NetID
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

func links(w World) netgraph.Links {
	return func(pos cube.Pos) []cube.Pos {
		var out []cube.Pos
		for _, f := range cube.Faces() {
			q := pos.Side(f)
			if _, ok := IsConduit(w.Block(q)); ok {
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

// Place adds the conduit block at pos (already in the world) with mB of fluid k
// as its share of the buffer.
func (g *Grid) Place(w World, pos cube.Pos, k Kind, mB int) {
	tier, ok := IsConduit(w.Block(pos))
	if !ok {
		return
	}
	var old []netgraph.NetID
	_, known := g.graph.NetOf(pos)
	byKind := map[byte]int{}
	if mB > 0 {
		byKind[byte(k)] += mB
	}
	seen := map[netgraph.NetID]bool{}
	for _, q := range append([]cube.Pos{pos}, links(w)(pos)...) {
		if id, ok := g.graph.NetOf(q); ok && !seen[id] {
			seen[id] = true
			old = append(old, id)
			if s := g.graph.Store(id); s.Amount > 0 {
				byKind[s.Tag] += s.Amount
			}
		}
	}
	g.tiers[pos] = tier
	id := g.graph.Add(pos, BufferOf(tier), netgraph.Store{Tag: byte(k), Amount: mB}, links(w))
	for _, o := range old {
		if o != id {
			g.forget(o)
		}
	}
	if len(byKind) > 1 { // the kind with more fluid stays, the other is lost
		best := netgraph.Store{}
		for tag, a := range byKind {
			if a > best.Amount || (a == best.Amount && tag < best.Tag) {
				best = netgraph.Store{Tag: tag, Amount: a}
			}
		}
		g.graph.SetStore(id, best)
	}
	if !known && len(old) == 1 && old[0] == id && g.cached(id) {
		g.addInPlace(w, id, pos, tier)
		return
	}
	g.dirty[id] = true
}

// cached reports whether network id has a clean cached Network.
func (g *Grid) cached(id netgraph.NetID) bool { return g.nets[id] != nil && !g.dirty[id] }

// endIndex is where the endpoint at pos is in n.Ends, or -1.
func (n *Network) endIndex(pos cube.Pos) int {
	return slices.IndexFunc(n.Ends, func(e Endpoint) bool { return e.Pos == pos })
}

// touches reports whether a conduit of network id is beside pos.
func (g *Grid) touches(pos cube.Pos, id netgraph.NetID) bool {
	for _, f := range cube.Faces() {
		if nid, ok := g.graph.NetOf(pos.Side(f)); ok && nid == id {
			return true
		}
	}
	return false
}

// addInPlace edits the cached network id for one conduit that joined it
// without merging: the conduit, its tier and the tanks beside it. Turn and the
// order of the other endpoints stay.
func (g *Grid) addInPlace(w World, id netgraph.NetID, pos cube.Pos, tier int) {
	n := g.nets[id]
	n.Conduits = append(n.Conduits, pos)
	n.Tier = min(n.Tier, tier)
	if i := n.endIndex(pos); i >= 0 { // a tank stood here
		n.Ends = slices.Delete(n.Ends, i, i+1)
	}
	for _, f := range cube.Faces() {
		q := pos.Side(f)
		if _, ok := g.graph.NetOf(q); ok {
			continue
		}
		if _, ok := w.Block(q).(Holder); ok && n.endIndex(q) < 0 {
			n.Ends = append(n.Ends, Endpoint{Pos: q})
		}
	}
}

// removeInPlace edits the cached network id for one conduit that left it
// without cutting it. tier is the tier the conduit had.
func (g *Grid) removeInPlace(w World, id netgraph.NetID, pos cube.Pos, tier int) {
	n := g.nets[id]
	if i := slices.Index(n.Conduits, pos); i >= 0 {
		last := len(n.Conduits) - 1
		n.Conduits[i] = n.Conduits[last]
		n.Conduits = n.Conduits[:last]
	}
	for _, f := range cube.Faces() {
		q := pos.Side(f)
		if i := n.endIndex(q); i >= 0 && !g.touches(q, id) {
			n.Ends = slices.Delete(n.Ends, i, i+1)
		}
	}
	if _, ok := w.Block(pos).(Holder); ok && g.touches(pos, id) && n.endIndex(pos) < 0 {
		n.Ends = append(n.Ends, Endpoint{Pos: pos})
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

// forget drops the per-network state of a network that no longer exists.
func (g *Grid) forget(id netgraph.NetID) {
	delete(g.nets, id)
	delete(g.dirty, id)
}

// Remove takes the conduit at pos (already gone from the world) out and returns
// the fluid that was its share of the buffer.
func (g *Grid) Remove(w World, pos cube.Pos) (Kind, int) {
	id, ok := g.graph.NetOf(pos)
	if !ok {
		return None, 0
	}
	tier := g.tiers[pos]
	st, parts := g.graph.Remove(pos, links(w))
	delete(g.tiers, pos)
	if len(parts) == 0 {
		g.forget(id)
	}
	if len(parts) == 1 && parts[0] == id && g.cached(id) {
		g.removeInPlace(w, id, pos, tier)
		return Kind(st.Tag), st.Amount
	}
	for _, p := range parts {
		g.dirty[p] = true
	}
	return Kind(st.Tag), st.Amount
}

// Changed tells the grid that the block at pos changed. A conduit is read
// again (its tier), a block that stopped being a conduit leaves, and for any
// other block the conduits that touch it read their tanks again.
func (g *Grid) Changed(w World, pos cube.Pos) {
	tier, isC := IsConduit(w.Block(pos))
	id, known := g.graph.NetOf(pos)
	switch {
	case known && isC:
		g.tiers[pos] = tier
		g.graph.SetWeight(pos, BufferOf(tier))
		g.dirty[id] = true
		return
	case known:
		g.Remove(w, pos)
		return
	case isC:
		g.Place(w, pos, None, 0)
		return
	}
	for _, f := range cube.Faces() {
		if nid, ok := g.graph.NetOf(pos.Side(f)); ok {
			g.dirty[nid] = true
		}
	}
}

// refresh reads the conduits and tanks of network id again.
func (g *Grid) refresh(w World, id netgraph.NetID) *Network {
	nodes := g.graph.Nodes(id)
	n := g.nets[id]
	if n == nil {
		n = &Network{}
		g.nets[id] = n
	}
	n.Conduits, n.Ends, n.Tier = nodes, nil, 0
	ends := map[cube.Pos]bool{}
	for _, pos := range nodes {
		if t := g.tiers[pos]; n.Tier == 0 || t < n.Tier {
			n.Tier = t
		}
		for _, f := range cube.Faces() {
			q := pos.Side(f)
			if _, ok := g.graph.NetOf(q); ok {
				continue
			}
			if _, ok := w.Block(q).(Holder); ok && !ends[q] {
				ends[q] = true
				n.Ends = append(n.Ends, Endpoint{Pos: q})
			}
		}
	}
	delete(g.dirty, id)
	return n
}

// verify checks a slice of the networks against the world, about all of them
// in 60 seconds worth of Steps.
func (g *Grid) verify(w World) {
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
		g.verifyNet(w, id)
	}
}

func (g *Grid) verifyNet(w World, id netgraph.NetID) {
	before := g.graph.Nodes(id)
	changed := g.graph.Verify(id, links(w), func(p cube.Pos) (int, bool) {
		t, ok := IsConduit(w.Block(p))
		if !ok {
			return 0, false
		}
		g.tiers[p] = t
		return BufferOf(t), true
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

// Step lets each network move its fluids; call it every StepTicks ticks. It
// returns how much moved between tanks, and from or to the buffers.
func (g *Grid) Step(w World) int {
	g.verify(w)
	moved := 0
	for _, id := range g.graph.Nets() {
		n := g.nets[id]
		if n == nil || g.dirty[id] {
			n = g.refresh(w, id)
		}
		moved += g.stepNet(w, id, n)
	}
	return moved
}

// stepNet moves fluid for one network: tank to tank first, then what is
// missing from the buffer to the sinks, then what is left of the sources into
// the buffer. Tanks that are both source and sink do not use the buffer. Both
// the amount the sinks get and the amount the sources give stay within
// RateOf(Tier) × StepTicks.
func (g *Grid) stepNet(w World, id netgraph.NetID, n *Network) int {
	budget := RateOf(n.Tier) * StepTicks
	moved := n.direct(w, budget)
	in, out := budget-moved, budget-moved
	total := moved
	sources, sinks := n.roles(w)
	st := g.graph.Store(id)
	capacity := g.graph.Capacity(id)

	// deficit: buffer to the sinks
	for j := 0; j < len(sinks) && out > 0 && st.Amount > 0; j++ {
		e := sinks[(n.turn+j)%len(sinks)]
		h, ok := w.Block(e.Pos).(Holder)
		if !ok {
			continue
		}
		if src, _ := h.FluidRoles(); src {
			continue
		}
		room := h.FillFluid(w, e.Pos, Kind(st.Tag), min(out, st.Amount), true)
		if room <= 0 {
			continue
		}
		put := h.FillFluid(w, e.Pos, Kind(st.Tag), room, false)
		st.Amount -= put
		out -= put
		total += put
	}
	if st.Amount <= 0 {
		st = netgraph.Store{}
	}

	// surplus: sources into the buffer, one kind only
	for i := 0; i < len(sources) && in > 0 && st.Amount < capacity; i++ {
		e := sources[(n.turn+i)%len(sources)]
		h, ok := w.Block(e.Pos).(Holder)
		if !ok {
			continue
		}
		if _, snk := h.FluidRoles(); snk {
			continue
		}
		k, have := h.DrainFluid(w, e.Pos, min(in, capacity-st.Amount), true)
		if have <= 0 || (st.Amount > 0 && byte(k) != st.Tag) {
			continue
		}
		k, took := h.DrainFluid(w, e.Pos, have, false)
		if took <= 0 {
			continue
		}
		st = netgraph.Store{Tag: byte(k), Amount: st.Amount + took}
		in -= took
		total += took
	}
	g.graph.SetStore(id, st)
	n.turn++
	return total
}

// Shares calls yield for every conduit with its part of the buffer.
func (g *Grid) Shares(yield func(pos cube.Pos, k Kind, mB int)) {
	g.graph.Shares(func(pos cube.Pos, s netgraph.Store) { yield(pos, Kind(s.Tag), s.Amount) })
}

// Buffer returns the buffer of the network of the conduit at pos and its
// capacity.
func (g *Grid) Buffer(pos cube.Pos) (k Kind, amount, capacity int, ok bool) {
	id, ok := g.graph.NetOf(pos)
	if !ok {
		return None, 0, 0, false
	}
	s := g.graph.Store(id)
	return Kind(s.Tag), s.Amount, g.graph.Capacity(id), true
}
