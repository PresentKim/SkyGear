// Package netgraph keeps conduit networks as a graph that is changed only by
// events. It maps a position to its network, merges networks when a node is
// added, splits them when a node is removed, and holds one shared buffer
// (Store) per network. It knows nothing about worlds or blocks: callers pass
// the neighbour function and the node weights.
package netgraph

import (
	"slices"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// NetID names one network. Zero is never a valid ID.
type NetID uint32

// Store is the buffer of a network. Tag tells the kind of content (for
// example a fluid kind) and is zero when Amount is zero.
type Store struct {
	Tag    byte
	Amount int
}

// Links returns the nodes that pos connects to. The graph ignores positions
// that are not nodes of the same network.
type Links func(pos cube.Pos) []cube.Pos

type node struct {
	net    NetID
	weight int
}

type net struct {
	nodes map[cube.Pos]struct{}
	store Store
	cap   int
}

// Graph is the set of all networks. It is not safe for concurrent use.
type Graph struct {
	nodes    map[cube.Pos]*node
	nets     map[NetID]*net
	next     NetID
	visited  int64
	rebuilds int64
}

// New returns an empty graph.
func New() *Graph {
	return &Graph{nodes: map[cube.Pos]*node{}, nets: map[NetID]*net{}}
}

func posLess(a, b cube.Pos) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[2] < b[2]
}

func sortPos(ps []cube.Pos) {
	slices.SortFunc(ps, func(a, b cube.Pos) int {
		if posLess(a, b) {
			return -1
		}
		if posLess(b, a) {
			return 1
		}
		return 0
	})
}

// combine merges two stores. Equal tags add; an empty side or tag 0 merges
// freely; different non-zero tags keep the larger store (tie: a).
func combine(a, b Store) Store {
	switch {
	case a.Amount <= 0:
		return norm(b)
	case b.Amount <= 0:
		return norm(a)
	case a.Tag == b.Tag:
		return Store{Tag: a.Tag, Amount: a.Amount + b.Amount}
	case a.Tag == 0:
		return Store{Tag: b.Tag, Amount: a.Amount + b.Amount}
	case b.Tag == 0:
		return Store{Tag: a.Tag, Amount: a.Amount + b.Amount}
	case b.Amount > a.Amount:
		return b
	}
	return a
}

func norm(s Store) Store {
	if s.Amount <= 0 {
		return Store{}
	}
	return s
}

func clamp(s Store, capacity int) Store {
	if s.Amount > capacity {
		s.Amount = capacity
	}
	return norm(s)
}

// Add puts a node at pos with the given weight and buffer share, joins it to
// every network it links to and returns the resulting network. If pos is
// already a node its weight is updated and the share is added.
func (g *Graph) Add(pos cube.Pos, weight int, share Store, links Links) NetID {
	if weight < 0 {
		weight = 0
	}
	cur := NetID(0)
	if nd, ok := g.nodes[pos]; ok {
		g.SetWeight(pos, weight)
		n := g.nets[nd.net]
		n.store = clamp(combine(n.store, share), n.cap)
		cur = nd.net
	} else {
		g.next++
		cur = g.next
		g.nodes[pos] = &node{net: cur, weight: weight}
		g.nets[cur] = &net{nodes: map[cube.Pos]struct{}{pos: {}}, store: clamp(share, weight), cap: weight}
	}
	group := []NetID{cur}
	seen := map[NetID]bool{cur: true}
	for _, q := range links(pos) {
		if nd, ok := g.nodes[q]; ok && !seen[nd.net] {
			seen[nd.net] = true
			group = append(group, nd.net)
		}
	}
	if len(group) == 1 {
		return cur
	}
	target := group[0]
	for _, id := range group[1:] {
		a, b := len(g.nets[id].nodes), len(g.nets[target].nodes)
		if a > b || (a == b && id < target) {
			target = id
		}
	}
	t := g.nets[target]
	for _, id := range group {
		if id == target {
			continue
		}
		o := g.nets[id]
		for p := range o.nodes {
			t.nodes[p] = struct{}{}
			g.nodes[p].net = target
		}
		t.store = combine(t.store, o.store)
		t.cap += o.cap
		delete(g.nets, id)
	}
	t.store = clamp(t.store, t.cap)
	return target
}

// Remove deletes the node at pos. It returns the part of the buffer that
// belonged to the node and the networks that are left of its old network
// (more than one when the removal cut it).
func (g *Graph) Remove(pos cube.Pos, links Links) (Store, []NetID) {
	nd, ok := g.nodes[pos]
	if !ok {
		return Store{}, nil
	}
	return g.resplit(nd.net, map[cube.Pos]struct{}{pos: {}}, links(pos), links, true)
}

// comp is one connected piece of what is left of a network. pos is nil only
// for a piece that keeps the network's ID, which needs no relabelling.
type comp struct {
	pos []cube.Pos
	cap int
}

// resplit takes the removed nodes out of network id, finds the connected
// pieces that remain and divides the buffer between the pieces and the
// removed nodes by capacity. With local set (one node removed, seeds are its
// links) the pieces are found by searching from the seeds only as far as
// needed (see cut); otherwise every remaining node is walked.
func (g *Graph) resplit(id NetID, removed map[cube.Pos]struct{}, seeds []cube.Pos, links Links, local bool) (Store, []NetID) {
	n := g.nets[id]
	remW := 0
	for p := range removed {
		remW += g.nodes[p].weight
		delete(n.nodes, p)
		delete(g.nodes, p)
	}
	var comps []comp
	ok := false
	if local {
		comps, ok = g.cut(n, seeds, links, n.cap-remW)
	}
	if !ok {
		comps = g.walk(n, seeds, links)
	}

	// Pieces: the components, then the removed nodes as the last one.
	k := len(comps)
	caps := make([]int, k+1)
	for i, c := range comps {
		caps[i] = c.cap
	}
	caps[k] = remW
	amounts := make([]int, k+1)
	A, C := n.store.Amount, n.cap
	sum, big := 0, 0
	for i, c := range caps {
		if C > 0 {
			amounts[i] = A * c / C
		}
		sum += amounts[i]
		if c > caps[big] {
			big = i
		}
	}
	rem := A - sum
	for j := 0; j <= k && rem > 0; j++ {
		i := (big + j) % (k + 1)
		give := caps[i] - amounts[i]
		if give > rem {
			give = rem
		}
		if give > 0 {
			amounts[i] += give
			rem -= give
		}
	}
	tag := n.store.Tag
	mk := func(a int) Store { return norm(Store{Tag: tag, Amount: a}) }
	removedStore := mk(amounts[k])

	if k == 0 {
		delete(g.nets, id)
		return removedStore, nil
	}
	keep := 0
	for i := 1; i < k; i++ {
		if caps[i] > caps[keep] {
			keep = i
		}
	}
	parts := make([]NetID, 0, k)
	for i, c := range comps {
		if i == keep {
			n.store, n.cap = mk(amounts[i]), c.cap
			parts = append(parts, id)
			continue
		}
		g.next++
		nn := &net{nodes: make(map[cube.Pos]struct{}, len(c.pos)), store: mk(amounts[i]), cap: c.cap}
		for _, p := range c.pos {
			delete(n.nodes, p)
			nn.nodes[p] = struct{}{}
			g.nodes[p].net = g.next
		}
		g.nets[g.next] = nn
		parts = append(parts, g.next)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i] < parts[j] })
	return removedStore, parts
}

// walk finds every connected piece of n by BFS, from the seeds first and
// then from any node they did not reach (links that were not symmetric).
func (g *Graph) walk(n *net, seeds []cube.Pos, links Links) []comp {
	seen := make(map[cube.Pos]struct{}, len(n.nodes))
	var comps []comp
	bfs := func(start cube.Pos) {
		if _, ok := n.nodes[start]; !ok {
			return
		}
		if _, ok := seen[start]; ok {
			return
		}
		seen[start] = struct{}{}
		c := comp{}
		queue := []cube.Pos{start}
		for head := 0; head < len(queue); head++ {
			p := queue[head]
			g.visited++
			c.pos = append(c.pos, p)
			c.cap += g.nodes[p].weight
			for _, q := range links(p) {
				if _, ok := n.nodes[q]; !ok {
					continue
				}
				if _, ok := seen[q]; ok {
					continue
				}
				seen[q] = struct{}{}
				queue = append(queue, q)
			}
		}
		comps = append(comps, c)
	}
	for _, s := range seeds {
		bfs(s)
	}
	if len(seen) < len(n.nodes) { // links were not symmetric: sweep the rest
		rest := make([]cube.Pos, 0, len(n.nodes)-len(seen))
		for p := range n.nodes {
			if _, ok := seen[p]; !ok {
				rest = append(rest, p)
			}
		}
		sortPos(rest)
		for _, p := range rest {
			bfs(p)
		}
	}
	return comps
}

// cut finds the pieces left of connected network n after one node went,
// given that node's links as seeds and restCap, the capacity left. It gives
// the same pieces in the same order as walk, but walks only what it must: no
// seed in n's nodes but one means no cut and no walk; otherwise one search
// per seed takes a node in turn per group of searches that met, until all met
// (no cut) or all groups but one ran out (those are the cut-off pieces, the
// last one is the rest). The rest is walked out too only when it is not the
// piece that keeps the network. It takes links to be symmetric, as Add
// builds networks; it reports false when no seed is in n or a search finds
// links that are not, and walk must be used.
func (g *Graph) cut(n *net, seeds []cube.Pos, links Links, restCap int) ([]comp, bool) {
	var starts []cube.Pos
	for _, s := range seeds {
		if _, ok := n.nodes[s]; !ok {
			continue
		}
		dup := false
		for _, t := range starts {
			dup = dup || t == s
		}
		if !dup {
			starts = append(starts, s)
		}
	}
	switch len(starts) {
	case 0:
		return nil, false
	case 1:
		return []comp{{cap: restCap}}, true
	}
	m := len(starts)
	parent := make([]int, m)        // union-find over searches; a root is its group's smallest index
	members := make([][]int, m)     // searches of each root's group
	queues := make([][]cube.Pos, m) // nodes each search took; heads[i] of them were expanded
	heads := make([]int, m)
	caps := make([]int, m)
	done := make([]bool, m) // the root's group ran out
	owner := map[cube.Pos]int{}
	for i, s := range starts {
		parent[i], members[i] = i, []int{i}
		queues[i], caps[i] = []cube.Pos{s}, g.nodes[s].weight
		owner[s] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	groups, open, odd := m, m, false
	// step expands one node of root r's group and reports false if it had none.
	step := func(r int) bool {
		for _, i := range members[r] {
			if heads[i] == len(queues[i]) {
				continue
			}
			p := queues[i][heads[i]]
			heads[i]++
			g.visited++
			for _, q := range links(p) {
				if _, ok := n.nodes[q]; !ok {
					continue
				}
				o, taken := owner[q]
				if !taken {
					owner[q] = i
					queues[i] = append(queues[i], q)
					caps[i] += g.nodes[q].weight
					continue
				}
				a, b := find(o), find(i)
				if a == b {
					continue
				}
				if done[a] || done[b] { // a piece that ran out is linked to: links are not symmetric
					odd = true
					continue
				}
				if a > b {
					a, b = b, a
				}
				parent[b] = a
				members[a] = append(members[a], members[b]...)
				members[b] = nil
				groups--
				open--
			}
			return true
		}
		return false
	}
	for groups > 1 && open > 1 {
		for r := 0; r < m && groups > 1 && open > 1; r++ {
			if parent[r] != r || done[r] {
				continue
			}
			if !step(r) {
				done[r] = true
				open--
			}
		}
	}
	if odd {
		return nil, false
	}
	if groups == 1 {
		return []comp{{cap: restCap}}, true
	}
	var comps []comp
	rest, restRoot, cut := -1, -1, 0
	for r := 0; r < m; r++ {
		if parent[r] != r {
			continue
		}
		if !done[r] {
			rest, restRoot = len(comps), r
			comps = append(comps, comp{})
			continue
		}
		c := comp{}
		for _, i := range members[r] {
			c.pos = append(c.pos, queues[i]...)
			c.cap += caps[i]
		}
		cut += c.cap
		comps = append(comps, c)
	}
	comps[rest].cap = restCap - cut
	keep := 0
	for i := 1; i < len(comps); i++ {
		if comps[i].cap > comps[keep].cap {
			keep = i
		}
	}
	if keep != rest {
		for step(restRoot) {
		}
		if odd {
			return nil, false
		}
		for _, i := range members[restRoot] {
			comps[rest].pos = append(comps[rest].pos, queues[i]...)
		}
	}
	return comps, true
}

// SetWeight changes the weight of the node at pos. The buffer is cut to the
// new capacity.
func (g *Graph) SetWeight(pos cube.Pos, weight int) {
	nd, ok := g.nodes[pos]
	if !ok {
		return
	}
	if weight < 0 {
		weight = 0
	}
	n := g.nets[nd.net]
	n.cap += weight - nd.weight
	nd.weight = weight
	n.store = clamp(n.store, n.cap)
}

// NetOf returns the network of the node at pos.
func (g *Graph) NetOf(pos cube.Pos) (NetID, bool) {
	nd, ok := g.nodes[pos]
	if !ok {
		return 0, false
	}
	return nd.net, true
}

// Nodes returns the positions of network id in a fixed order (x, then y, then z).
func (g *Graph) Nodes(id NetID) []cube.Pos {
	n, ok := g.nets[id]
	if !ok {
		return nil
	}
	out := make([]cube.Pos, 0, len(n.nodes))
	for p := range n.nodes {
		out = append(out, p)
	}
	sortPos(out)
	return out
}

// Each calls yield for every node of network id in no fixed order; use it
// when the order does not matter, since Nodes has to sort.
func (g *Graph) Each(id NetID, yield func(pos cube.Pos)) {
	if n, ok := g.nets[id]; ok {
		for p := range n.nodes {
			yield(p)
		}
	}
}

// Nets returns all network IDs in ascending order.
func (g *Graph) Nets() []NetID {
	out := make([]NetID, 0, len(g.nets))
	for id := range g.nets {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Store returns the buffer of network id.
func (g *Graph) Store(id NetID) Store {
	if n, ok := g.nets[id]; ok {
		return n.store
	}
	return Store{}
}

// SetStore replaces the buffer of network id. It is cut to the capacity, and
// the tag is zero when the amount is zero.
func (g *Graph) SetStore(id NetID, s Store) {
	if n, ok := g.nets[id]; ok {
		n.store = clamp(s, n.cap)
	}
}

// Size returns how many nodes network id has, without listing them.
func (g *Graph) Size(id NetID) int {
	if n, ok := g.nets[id]; ok {
		return len(n.nodes)
	}
	return 0
}

// Capacity returns the sum of the node weights of network id.
func (g *Graph) Capacity(id NetID) int {
	if n, ok := g.nets[id]; ok {
		return n.cap
	}
	return 0
}

// Len returns the number of nodes in all networks.
func (g *Graph) Len() int { return len(g.nodes) }

// Verify checks network id against the world. Nodes for which isNode is false
// are removed with their share of the buffer, weights that changed are taken
// over, and a network that is no longer connected is split. It reports whether
// anything changed.
func (g *Graph) Verify(id NetID, links Links, isNode func(cube.Pos) (int, bool)) bool {
	n, ok := g.nets[id]
	if !ok {
		return false
	}
	changed := false
	removed := map[cube.Pos]struct{}{}
	for p := range n.nodes { // order does not matter: removed is a set
		w, ok := isNode(p)
		if !ok {
			removed[p] = struct{}{}
			continue
		}
		if w < 0 {
			w = 0
		}
		if nd := g.nodes[p]; nd.weight != w {
			n.cap += w - nd.weight
			nd.weight = w
			changed = true
		}
	}
	n.store = clamp(n.store, n.cap)
	if len(removed) > 0 {
		changed = true
	}
	_, parts := g.resplit(id, removed, nil, links, false)
	if len(parts) > 1 {
		changed = true
	}
	if changed {
		g.rebuilds++
	}
	return changed
}

// Shares calls yield for every node with its part of the buffer of its
// network. Parts follow the node weights; the remainder goes to the first node
// of the network, so the parts add up to the Store exactly.
func (g *Graph) Shares(yield func(pos cube.Pos, s Store)) {
	for _, id := range g.Nets() {
		n := g.nets[id]
		ns := g.Nodes(id)
		amounts := make([]int, len(ns))
		sum := 0
		if n.cap > 0 {
			for i, p := range ns {
				amounts[i] = n.store.Amount * g.nodes[p].weight / n.cap
				sum += amounts[i]
			}
		}
		if len(ns) > 0 {
			amounts[0] += n.store.Amount - sum
		}
		for i, p := range ns {
			yield(p, norm(Store{Tag: n.store.Tag, Amount: amounts[i]}))
		}
	}
}

// Visited returns how many nodes BFS walks have passed over.
func (g *Graph) Visited() int64 { return g.visited }

// Rebuilds returns how many Verify calls changed something.
func (g *Graph) Rebuilds() int64 { return g.rebuilds }
