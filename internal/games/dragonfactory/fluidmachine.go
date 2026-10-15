package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Machines with a tank (docs/design/07-progression-expansion.md §6.1): the fluid
// tank, the electric crucible (cobblestone into lava) and the lava generator.
// They have no window: a bucket fills or empties them by hand, fluid conduits
// carry the fluid (internal/fluidnet), and their WAILA panel shows the tank.
const (
	// bucketMB is the mB of a bucket.
	bucketMB = 1000
	// crucibleLava is the mB of lava a cobblestone melts into.
	crucibleLava = 250
	// lavaGenOutput is the RE per tick the lava generator makes; it burns one
	// mB every lavaGenBurn ticks.
	lavaGenOutput = 80
	lavaGenBurn   = 2
	// tankCapacity is the mB of a fluid tank.
	tankCapacity = 16000
)

// fluidKind returns the fluid of a bucket's content.
func fluidOfBucket(b item.Bucket) fluidnet.Kind {
	liq, ok := b.Content.Liquid()
	if !ok {
		return fluidnet.None
	}
	switch liq.(type) {
	case block.Water:
		return fluidnet.Water
	case block.Lava:
		return fluidnet.Lava
	}
	return fluidnet.None
}

// bucketOf returns the full bucket of a fluid.
func bucketOf(k fluidnet.Kind) item.Stack {
	var liq world.Liquid = block.Water{Depth: 8, Still: true}
	if k == fluidnet.Lava {
		liq = block.Lava{Depth: 8, Still: true}
	}
	return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(liq)}, 1)
}

// fluidName is the name of a fluid in the catalog.
func fluidName(k fluidnet.Kind) string {
	if k == fluidnet.Lava {
		return i18n.T(i18n.Default, "fluid.lava")
	}
	return i18n.T(i18n.Default, "fluid.water")
}

// FluidRoles implements fluidnet.Holder.
func (m Machine) FluidRoles() (source, sink bool) {
	if m.st == nil {
		return false, false
	}
	return m.st.kind.fluidSource, m.st.kind.fluidSink
}

// FillFluid implements fluidnet.Holder.
func (m Machine) FillFluid(_ fluidnet.World, _ cube.Pos, k fluidnet.Kind, mB int, simulate bool) int {
	if m.st == nil || !m.st.kind.fluidSink || k == fluidnet.None {
		return 0
	}
	if only := m.st.kind.fluidOnly; only != fluidnet.None && only != k {
		return 0
	}
	if m.st.fluidAmt > 0 && m.st.fluid != k {
		return 0
	}
	n := min(mB, m.st.kind.fluidCap-m.st.fluidAmt)
	if n <= 0 {
		return 0
	}
	if !simulate {
		m.st.fluid, m.st.fluidAmt = k, m.st.fluidAmt+n
	}
	return n
}

// DrainFluid implements fluidnet.Holder.
func (m Machine) DrainFluid(_ fluidnet.World, _ cube.Pos, max int, simulate bool) (fluidnet.Kind, int) {
	if m.st == nil || !m.st.kind.fluidSource || m.st.fluidAmt == 0 {
		return fluidnet.None, 0
	}
	n := min(max, m.st.fluidAmt)
	k := m.st.fluid
	if !simulate {
		if m.st.fluidAmt -= n; m.st.fluidAmt == 0 {
			m.st.fluid = fluidnet.None
		}
	}
	return k, n
}

var _ fluidnet.Holder = Machine{}

// useFluidMachine handles a player using a machine with a tank: a full bucket
// pours into the tank, an empty one takes a bucket's worth out, and the
// crucible takes its cobblestone. It returns whether the click was used.
func (s *session) useFluidMachine(p *player.Player, tx *world.Tx, pos cube.Pos, m Machine, held item.Stack, ctx *item.UseContext) bool {
	st := m.st
	if b, ok := held.Item().(item.Bucket); ok {
		if k := fluidOfBucket(b); k != fluidnet.None {
			if m.FillFluid(tx, pos, k, bucketMB, true) < bucketMB {
				return true
			}
			m.FillFluid(tx, pos, k, bucketMB, false)
			if held.Count() == 1 {
				_, off := p.HeldItems()
				p.SetHeldItems(item.NewStack(item.Bucket{}, 1), off)
			} else {
				ctx.SubtractFromCount(1)
				s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(item.Bucket{}, 1))
			}
			tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
			return true
		}
		if b.Empty() && st.kind.fluidSource && st.fluidAmt >= bucketMB {
			k, _ := m.DrainFluid(tx, pos, bucketMB, false)
			full := bucketOf(k)
			if held.Count() == 1 {
				_, off := p.HeldItems()
				p.SetHeldItems(full, off)
			} else {
				ctx.SubtractFromCount(1)
				s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), full)
			}
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
			return true
		}
		return true
	}
	if st.kind.marker == "" && len(st.kind.input) > 0 && st.accepts(held) {
		put := 0
		for put < held.Count() && st.insert(held) {
			put++
		}
		if put > 0 {
			ctx.SubtractFromCount(put)
		}
		return true
	}
	return false
}

// fluidLines are the WAILA lines of a machine with a tank.
func fluidLines(st *machineState) []string {
	if st.fluidAmt == 0 {
		return []string{fmt.Sprintf(i18n.T(i18n.Default, "fluid.lines.empty"), st.kind.fluidCap)}
	}
	return []string{fmt.Sprintf(i18n.T(i18n.Default, "fluid.lines.tank"), fluidName(st.fluid), st.fluidAmt, st.kind.fluidCap)}
}

// runCrucible melts the cobblestone in the input of an electric crucible into
// lava, every interval one block, while the tank has room.
func (st *machineState) runCrucible() {
	slot, in, ok := st.firstInput()
	if !ok || !st.accepts(in) || st.fluidAmt+crucibleLava > st.kind.fluidCap || (st.fluidAmt > 0 && st.fluid != fluidnet.Lava) {
		st.progress = 0
		return
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	st.fluid, st.fluidAmt = fluidnet.Lava, st.fluidAmt+crucibleLava
	st.progress = 0
}

// runLavaGenerator burns the lava in its tank into energy while the buffer has room.
func (s *session) runLavaGenerator(st *machineState, pos cube.Pos) {
	gen := st.genRate()
	if st.fluidAmt > 0 && st.energy+gen <= st.energyCap() {
		if st.progress += st.burnRate(); st.progress >= lavaGenBurn {
			st.progress = 0
			if st.fluidAmt--; st.fluidAmt == 0 {
				st.fluid = fluidnet.None
			}
		}
		st.energy += gen
		st.made += gen
		st.didWork = true
	} else if st.fluidAmt == 0 {
		st.stuck = stuckFuel
	}
	s.energyGrid().AddSource(pos)
}

// washerWater is the mB of water washing one ore chunk takes; it makes three
// dusts of it instead of the crusher's two.
const washerWater = 250

// runWasher washes the ore chunk in the input: with enough water in the tank it
// works an interval, and gives three dusts of the metal.
func (st *machineState) runWasher() {
	slot, in, ok := st.firstInput()
	if !ok || !st.accepts(in) || st.fluidAmt < washerWater || st.fluid != fluidnet.Water {
		st.progress = 0
		return
	}
	if st.progress < st.kind.interval {
		st.tickWork()
		return
	}
	out := item.NewStack(MetalDust{Metal: in.Item().(OreChunk).Ore}, 3)
	if !st.roomFor([]item.Stack{out}) {
		return
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	st.emit(out)
	if st.fluidAmt -= washerWater; st.fluidAmt == 0 {
		st.fluid = fluidnet.None
	}
	st.progress = 0
}
