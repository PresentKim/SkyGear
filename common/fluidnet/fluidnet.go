// Package fluidnet moves fluids over networks of touching fluid conduits,
// round robin from the tanks that give (sources) to those that take (sinks),
// up to the network tier's rate. Blocks take part via Holder and Conduit.
package fluidnet

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
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
// much moved: every source in turn gives to the sinks in turn.
func (n *Network) Step(w World) int {
	var sources, sinks []Endpoint
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
	if len(sources) == 0 || len(sinks) == 0 {
		return 0
	}
	left := RateOf(n.Tier) * StepTicks
	// a pass gives every source's turn a share to every sink; passes go on until
	// the budget is spent or nothing moves
	share := max(1, left/(len(sources)*len(sinks)))
	moved := 0
	for left > 0 {
		progressed := false
		for i := range sources {
			src := sources[(n.turn+i)%len(sources)]
			sh := w.Block(src.Pos).(Holder)
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
				dh := w.Block(dst.Pos).(Holder)
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
	n.turn++
	return moved
}

// registrationTTL is how many steps a conduit stays in the grid without
// registering again.
const registrationTTL = 10

// Grid steps the fluid networks of one world.
type Grid struct {
	conduits map[cube.Pos]int64
	nets     map[cube.Pos]*Network
	step     int64
}

func NewGrid() *Grid {
	return &Grid{conduits: map[cube.Pos]int64{}, nets: map[cube.Pos]*Network{}}
}

// AddConduit registers the conduit at pos; call it whenever the block ticks.
func (g *Grid) AddConduit(pos cube.Pos) { g.conduits[pos] = g.step }

// Step finds the networks of the registered conduits and lets each move its
// fluids; call it every StepTicks ticks. It returns how much moved.
func (g *Grid) Step(w World) int {
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
		n := Discover(w, pos, 2048)
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
		if old, ok := g.nets[low]; ok {
			n.turn = old.turn
		}
		keep[low] = n
		moved += n.Step(w)
	}
	g.nets = keep
	return moved
}
