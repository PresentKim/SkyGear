package skygear

import (
	"fmt"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/network"
)

// Measurements of the conduit networks (energy, items, fluids) before the
// event-driven graph: what a tick of conduits costs, what a full scan costs
// against a cached step, how far one change reaches and how often the grids
// scan in play. docs/plans/2026-11-08-netgraph-baseline.md has the numbers.

// netY is a layer of air above the island for the measured conduits.
const netY = island.IslandY + 10

// machinePosOf is beside the plane's corner, touching one bundle.
var machinePosOf = cube.Pos{-17, netY, -16}

// allKinds is a bundle with every conduit kind at tier 1.
var allKinds = conduit.Bundle{Energy: 1, Item: 1, Fluid: 1}

// placePlane places a side × side square of bundles: one network per kind.
func placePlane(s *session, side int, b conduit.Bundle) []cube.Pos {
	var ps []cube.Pos
	s.World.Do(func(tx *world.Tx) {
		for x := range side {
			for z := range side {
				pos := cube.Pos{x - 16, netY, z - 16}
				tx.SetBlock(pos, b, nil)
				ps = append(ps, pos)
			}
		}
	})
	return ps
}

// placeLines places count separate lines of length bundles: count networks
// per kind.
func placeLines(s *session, count, length int, b conduit.Bundle) []cube.Pos {
	var ps []cube.Pos
	s.World.Do(func(tx *world.Tx) {
		for k := range count {
			for x := range length {
				pos := cube.Pos{x - 16, netY, 2*k - 16}
				tx.SetBlock(pos, b, nil)
				ps = append(ps, pos)
			}
		}
	})
	return ps
}

// settle ticks the world until every bundle registered with its grids.
func settle(tb testing.TB, s *session, want int) {
	tb.Helper()
	for range 8 {
		s.World.AdvanceTick()
	}
	if got := network.Energy(s.Session).Conduits(); got != want {
		tb.Fatalf("%d energy conduits registered, want %d", got, want)
	}
}

// replace breaks the bundle at pos and places it again: the remove and place
// events of one change.
func replace(tx *world.Tx, pos cube.Pos, b conduit.Bundle) {
	tx.SetBlock(pos, block.Air{}, nil)
	tx.SetBlock(pos, b, nil)
}

// reportScans adds the full scans and the conduits walked per step.
func reportScans(b *testing.B, before, after energyCounters) {
	scans := after.Scans - before.Scans
	b.ReportMetric(float64(scans)/float64(b.N), "scans/op")
	b.ReportMetric(float64(after.Visited-before.Visited)/float64(b.N), "visited/op")
}

// energyCounters is the counters of any grid (they have the same fields).
type energyCounters struct{ Scans, Visited int64 }

// BenchmarkConduitTick is one world tick with n bundles of every kind and
// nothing else: the per-block work (registration, phased refresh) that an
// event-driven graph would no longer need. Grid steps are not in it.
func BenchmarkConduitTick(b *testing.B) {
	for _, side := range []int{0, 10, 32} {
		n := side * side
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			s := benchSession(b)
			placePlane(s, side, allKinds)
			settle(b, s, n)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				s.World.AdvanceTick()
			}
		})
	}
}

// BenchmarkGridStep is one Step of each grid over a network of 1024 conduits,
// on the cached topology and after one bundle is broken and placed again.
func BenchmarkGridStep(b *testing.B) {
	type grid struct {
		name     string
		step     func(s *session, tx *world.Tx)
		counters func(s *session) energyCounters
	}
	grids := []grid{
		{"energy", func(s *session, tx *world.Tx) { network.Energy(s.Session).Step(tx, 4) },
			func(s *session) energyCounters { return energyCounters(network.Energy(s.Session).Counters()) }},
		{"item", func(s *session, tx *world.Tx) { network.Items(s.Session).Step(tx) },
			func(s *session) energyCounters { return energyCounters(network.Items(s.Session).Counters()) }},
		{"fluid", func(s *session, tx *world.Tx) { network.Fluids(s.Session).Step(tx) },
			func(s *session) energyCounters { return energyCounters(network.Fluids(s.Session).Counters()) }},
	}
	for _, g := range grids {
		for _, scan := range []bool{false, true} {
			name := g.name + "/cached"
			if scan {
				name = g.name + "/rescan"
			}
			b.Run(name, func(b *testing.B) {
				s := benchSession(b)
				placePlane(s, 32, allKinds)
				settle(b, s, 1024)
				s.World.Do(func(tx *world.Tx) { g.step(s, tx) })
				before := g.counters(s)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					s.World.Do(func(tx *world.Tx) {
						if scan {
							replace(tx, cube.Pos{-16, netY, -16}, allKinds)
						}
						g.step(s, tx)
					})
				}
				b.StopTimer()
				reportScans(b, before, g.counters(s))
			})
		}
	}
}

// BenchmarkOneChange is the energy step after one change in a world of 16
// separate networks of 64 conduits: one remove and one place on one line,
// against a cached step. The graph walks only the changed network.
func BenchmarkOneChange(b *testing.B) {
	for _, scan := range []bool{false, true} {
		name := "cached"
		if scan {
			name = "changed"
		}
		b.Run(name, func(b *testing.B) {
			s := benchSession(b)
			energyOnly := conduit.Bundle{Energy: 1}
			placeLines(s, 16, 64, energyOnly)
			settle(b, s, 1024)
			g := network.Energy(s.Session)
			s.World.Do(func(tx *world.Tx) { g.Step(tx, 4) })
			before := energyCounters(g.Counters())
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				s.World.Do(func(tx *world.Tx) {
					if scan {
						replace(tx, cube.Pos{-16, netY, -16}, energyOnly)
					}
					g.Step(tx, 4)
				})
			}
			b.StopTimer()
			reportScans(b, before, energyCounters(g.Counters()))
		})
	}
}

// TestMeasureNetworkScans plays two minutes (2400 ticks, Fast every 4) of a
// 1024-conduit network of every kind and logs how often each grid scanned
// everything again and how many conduits it walked: idle, and with a
// crusher beside the network that starts and stops working (its working
// look is a block change next to the conduits). Run with -v to see the
// numbers; it checks that idle and machine runs scan nothing again.
func TestMeasureNetworkScans(t *testing.T) {
	if testing.Short() {
		t.Skip("measurement")
	}
	const ticks = 2400
	run := func(t *testing.T, withMachine bool) {
		s := testSession(t)
		loadAround(s)
		placePlane(s, 32, allKinds)
		var st *machine.State
		if withMachine {
			// Beside the plane's corner, touching one energy conduit.
			st = placeTestMachine(s, machinePosOf, machine.KindCrusher)
		}
		settle(t, s, 1024)
		e0, i0, f0 := network.Energy(s.Session).Counters(), network.Items(s.Session).Counters(), network.Fluids(s.Session).Counters()
		changes, active := 0, false
		start := time.Now()
		for tick := range ticks {
			if st != nil && tick%300 == 0 {
				// One cobblestone works for a while, then the crusher idles
				// past ActiveHold: one start and one stop every 300 ticks.
				for _, slot := range st.Def.Output {
					_ = st.Inv().SetItem(slot, item.Stack{})
				}
				_ = st.Insert(item.NewStack(block.Cobblestone{}, 1))
			}
			s.World.AdvanceTick()
			if st != nil {
				s.World.Do(func(tx *world.Tx) {
					if m, ok := tx.Block(machinePosOf).(machine.Machine); ok && m.Active != active {
						active = m.Active
						changes++
					}
				})
			}
			if tick%4 == 3 {
				s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
			}
		}
		took := time.Since(start)
		e, i, f := network.Energy(s.Session).Counters(), network.Items(s.Session).Counters(), network.Fluids(s.Session).Counters()
		t.Logf("%d ticks in %v (%.3f ms/tick), working look changed %d times", ticks, took, float64(took.Microseconds())/1000/ticks, changes)
		t.Logf("energy: %d scans, %d conduits walked", e.Scans-e0.Scans, e.Visited-e0.Visited)
		t.Logf("item:   %d scans, %d conduits walked", i.Scans-i0.Scans, i.Visited-i0.Visited)
		t.Logf("fluid:  %d scans, %d conduits walked", f.Scans-f0.Scans, f.Visited-f0.Visited)
	}
	t.Run("idle", func(t *testing.T) { run(t, false) })
	t.Run("machine", func(t *testing.T) { run(t, true) })
}
