package energy

import (
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"

	"github.com/presentkim/skygear/common/netgraph"
)

// verifyTicks is the number of ticks that make up the 60 seconds in which
// every network is looked at once.
const verifyTicks = 60 * 20

// BufferOf is the RE one conduit of a tier holds; a network holds the sum.
func BufferOf(tier int) int { return RateOf(tier) / 40 }

// role is what an endpoint block takes part as.
type role struct{ src, snk bool }

// netInfo is what the grid remembers about one network besides its buffer:
// the blocks beside its conduits and its rate. It is rebuilt only when the
// conduits of the network change, and edited in place when a block beside it
// changes.
type netInfo struct {
	ends    map[Endpoint]role
	count   int
	rate    int
	stale   bool // sources and sinks must be sorted out of ends again
	sources []Endpoint
	sinks   []Endpoint
}

// Grid shares out the energy of one world. Conduits are put in and taken out
// by events (Place, Remove, Changed); no scan runs on its own.
type Grid struct {
	graph    *netgraph.Graph
	tiers    map[cube.Pos]int
	info     map[netgraph.NetID]*netInfo
	sources  map[cube.Pos]struct{}
	cursor   netgraph.NetID
	counters Counters
}

// Counters say how much topology work a grid did since it was made.
type Counters struct {
	// Scans is how many times every network was found again from scratch.
	Scans int64
	// Visited is how many conduits the BFS walks passed over in all.
	Visited int64
}

// Counters returns the topology work done so far.
func (g *Grid) Counters() Counters {
	c := g.counters
	c.Visited = g.graph.Visited()
	return c
}

func NewGrid() *Grid {
	return &Grid{
		graph:   netgraph.New(),
		tiers:   map[cube.Pos]int{},
		info:    map[netgraph.NetID]*netInfo{},
		sources: map[cube.Pos]struct{}{},
	}
}

// Conduits returns how many conduits are in the grid.
func (g *Grid) Conduits() int { return g.graph.Len() }

func faceLinks(p cube.Pos) []cube.Pos {
	out := make([]cube.Pos, 0, 6)
	for _, f := range cube.Faces() {
		out = append(out, p.Side(f))
	}
	return out
}

// Place puts the conduit at pos in the grid with share RE of buffer. If pos is
// already a conduit its tier is read again and the share is added. Nothing
// happens when the block at pos is not a conduit.
func (g *Grid) Place(w World, pos cube.Pos, share int) {
	tier, ok := IsConduit(w.Block(pos))
	if !ok {
		return
	}
	g.tiers[pos] = tier
	id := g.graph.Add(pos, BufferOf(tier), netgraph.Store{Amount: share}, faceLinks)
	delete(g.info, id)
}

// Remove takes the conduit at pos out of the grid and returns the buffer that
// went with it.
func (g *Grid) Remove(w World, pos cube.Pos) int {
	id, ok := g.graph.NetOf(pos)
	if !ok {
		return 0
	}
	share, parts := g.graph.Remove(pos, faceLinks)
	delete(g.tiers, pos)
	delete(g.info, id)
	for _, p := range parts {
		delete(g.info, p)
	}
	return share.Amount
}

// Changed tells the grid that the block at pos changed. A conduit that
// appeared, went or changed tier is placed, removed or reweighed; for any
// other block only the endpoints beside the conduits it touches are read
// again, and no network is walked.
func (g *Grid) Changed(w World, pos cube.Pos) {
	b := w.Block(pos)
	tier, isConduit := IsConduit(b)
	old, known := g.tiers[pos]
	switch {
	case isConduit && !known:
		g.Place(w, pos, 0)
		return
	case isConduit:
		if tier != old {
			g.tiers[pos] = tier
			g.graph.SetWeight(pos, BufferOf(tier))
			if id, ok := g.graph.NetOf(pos); ok {
				delete(g.info, id)
			}
		}
		return
	case known:
		g.Remove(w, pos)
	}
	src, snk := RolesOf(b)
	r := role{src != nil, snk != nil}
	for _, f := range cube.Faces() {
		id, ok := g.graph.NetOf(pos.Side(f))
		if !ok {
			continue
		}
		ni := g.info[id]
		if ni == nil {
			continue // read when the network is first used
		}
		e := Endpoint{Pos: pos, Face: f.Opposite()}
		cur, had := ni.ends[e]
		switch {
		case !r.src && !r.snk:
			if had {
				delete(ni.ends, e)
				ni.stale = true
			}
		case !had || cur != r:
			ni.ends[e] = r
			ni.stale = true
		}
	}
}

// Shares calls yield for every conduit with its part of the buffer of its
// network, for saving.
func (g *Grid) Shares(yield func(pos cube.Pos, share int)) {
	g.graph.Shares(func(pos cube.Pos, s netgraph.Store) { yield(pos, s.Amount) })
}

// Buffer returns the RE held and the RE that fit in the network of the
// conduit at pos.
func (g *Grid) Buffer(pos cube.Pos) (amount, capacity int, ok bool) {
	id, ok := g.graph.NetOf(pos)
	if !ok {
		return 0, 0, false
	}
	return g.graph.Store(id).Amount, g.graph.Capacity(id), true
}

// AddSource registers a source that touches sinks without a conduit (a
// generator next to a machine); a source next to a conduit needs no
// registration.
func (g *Grid) AddSource(pos cube.Pos) { g.sources[pos] = struct{}{} }

// Stats say what a Step found.
type Stats struct {
	Networks, Conduits int
	Moved              int
}

// netInfoOf returns the info of network id, reading the blocks beside its
// conduits if it is not known yet.
func (g *Grid) netInfoOf(w World, id netgraph.NetID) *netInfo {
	if ni := g.info[id]; ni != nil {
		return ni
	}
	ni := &netInfo{ends: map[Endpoint]role{}, stale: true}
	lowest := 0
	for _, pos := range g.graph.Nodes(id) {
		ni.count++
		if t := g.tiers[pos]; lowest == 0 || t < lowest {
			lowest = t
		}
		for _, f := range cube.Faces() {
			next := pos.Side(f)
			b := w.Block(next)
			if _, isConduit := IsConduit(b); isConduit {
				continue
			}
			src, snk := RolesOf(b)
			if src != nil || snk != nil {
				ni.ends[Endpoint{Pos: next, Face: f.Opposite()}] = role{src != nil, snk != nil}
			}
		}
	}
	ni.rate = RateOf(lowest)
	g.info[id] = ni
	return ni
}

func (ni *netInfo) sort() {
	if !ni.stale {
		return
	}
	ni.stale = false
	ni.sources, ni.sinks = ni.sources[:0], ni.sinks[:0]
	for e, r := range ni.ends {
		if r.src {
			ni.sources = append(ni.sources, e)
		}
		if r.snk {
			ni.sinks = append(ni.sinks, e)
		}
	}
	less := func(s []Endpoint) func(i, j int) bool {
		return func(i, j int) bool {
			if s[i].Pos != s[j].Pos {
				return s[i].Pos[0] < s[j].Pos[0] || (s[i].Pos[0] == s[j].Pos[0] && (s[i].Pos[1] < s[j].Pos[1] || (s[i].Pos[1] == s[j].Pos[1] && s[i].Pos[2] < s[j].Pos[2])))
			}
			return s[i].Face < s[j].Face
		}
	}
	sort.Slice(ni.sources, less(ni.sources))
	sort.Slice(ni.sinks, less(ni.sinks))
}

// Step moves the energy of ticks ticks. A network carries up to its rate ×
// ticks into its sinks and up to the same out of its sources. Sources feed
// sinks first; what the sinks did not take fills the buffer, and what the
// sinks still want comes from the buffer. Every registered direct source
// gives the sinks it touches up to DirectRate × ticks. At the end part of the
// networks are checked against the world.
func (g *Grid) Step(w World, ticks int) Stats {
	var st Stats
	ids := g.graph.Nets()
	alive := make(map[netgraph.NetID]struct{}, len(ids))
	for _, id := range ids {
		alive[id] = struct{}{}
	}
	for id := range g.info {
		if _, ok := alive[id]; !ok {
			delete(g.info, id)
		}
	}
	for _, id := range ids {
		ni := g.netInfoOf(w, id)
		st.Networks++
		st.Conduits += ni.count
		st.Moved += g.flow(w, id, ni, ni.rate*ticks)
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
	g.verify(w, ticks)
	return st
}

// flow runs one network for a Step with limit RE of throughput each way and
// returns what reached the sinks.
func (g *Grid) flow(w World, id netgraph.NetID, ni *netInfo, limit int) int {
	ni.sort()
	n := (&Network{Sources: ni.sources, Sinks: ni.sinks, Rate: ni.rate}).live(w)
	store := g.graph.Store(id).Amount
	capacity := g.graph.Capacity(id)
	direct := n.Transfer(w, limit)
	moved := direct
	// What the sinks left of the supply fills the buffer.
	want := min(limit-direct, capacity-store)
	for _, s := range n.Sources {
		if want <= 0 {
			break
		}
		got := w.Block(s.Pos).(Source).ExtractEnergy(w, s.Pos, s.Face, want, false)
		store += got
		want -= got
	}
	// What the sinks still want comes from the buffer.
	if budget := min(limit-direct, store); budget > 0 && len(n.Sinks) > 0 {
		out := drain(w, n.Sinks, budget)
		store -= out
		moved += out
	}
	g.graph.SetStore(id, netgraph.Store{Amount: store})
	return moved
}

// drain gives up to budget RE to sinks by the same water filling Transfer
// does, and returns what they took.
func drain(w World, sinks []Endpoint, budget int) int {
	want := make([]int, len(sinks))
	demand := 0
	for i, k := range sinks {
		want[i] = w.Block(k.Pos).(Sink).ReceiveEnergy(w, k.Pos, k.Face, budget, true)
		demand += want[i]
	}
	total := min(budget, demand)
	if total <= 0 {
		return 0
	}
	give := fill(want, total)
	moved := 0
	for i, k := range sinks {
		if give[i] > 0 {
			moved += w.Block(k.Pos).(Sink).ReceiveEnergy(w, k.Pos, k.Face, give[i], false)
		}
	}
	return moved
}

// fill divides total over the wants: small wants in full, the rest equally.
func fill(want []int, total int) []int {
	give := make([]int, len(want))
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
			for _, i := range open[:min(left, len(open))] {
				give[i]++
			}
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
	return give
}

// verify checks networks against the world, going round so that all of them
// are seen about once in a minute: a Step checks up to conduits / (Steps in a
// minute) conduits, and at least one network.
func (g *Grid) verify(w World, ticks int) {
	conduits := g.graph.Len()
	if conduits == 0 {
		return
	}
	perMinute := max(1, verifyTicks/max(1, ticks))
	budget := max(1, conduits/perMinute)
	ids := g.graph.Nets()
	start := sort.Search(len(ids), func(i int) bool { return ids[i] > g.cursor })
	isNode := func(p cube.Pos) (int, bool) {
		tier, ok := IsConduit(w.Block(p))
		if !ok {
			return 0, false
		}
		g.tiers[p] = tier
		return BufferOf(tier), true
	}
	used := 0
	for k := 0; k < len(ids) && used < budget; k++ {
		id := ids[(start+k)%len(ids)]
		g.cursor = id
		size := 1
		if ni := g.info[id]; ni != nil {
			size = ni.count
		}
		used += size
		before := g.graph.Nodes(id)
		if g.graph.Verify(id, faceLinks, isNode) {
			delete(g.info, id)
			for _, p := range before {
				if _, ok := g.graph.NetOf(p); !ok {
					delete(g.tiers, p)
				}
			}
		}
	}
}

// live is n without the endpoints that stopped being a Source or a Sink since
// they were read.
func (n *Network) live(w World) *Network {
	out := &Network{Conduits: n.Conduits, Rate: n.Rate}
	for _, e := range n.Sources {
		if src, _ := RolesOf(w.Block(e.Pos)); src != nil {
			out.Sources = append(out.Sources, e)
		}
	}
	for _, e := range n.Sinks {
		if _, snk := RolesOf(w.Block(e.Pos)); snk != nil {
			out.Sinks = append(out.Sinks, e)
		}
	}
	return out
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
