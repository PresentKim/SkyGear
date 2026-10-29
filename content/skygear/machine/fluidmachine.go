package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

var _ fluidnet.Holder = Machine{}

// Machines with a tank: the fluid
// tank, the electric crucible (cobblestone into lava) and the lava generator.
// They have no window: a bucket fills or empties them by hand, fluid conduits
// carry the fluid (common/fluidnet), and their WAILA panel shows the tank.
const (
	// bucketMB is the mB of a bucket.
	BucketMB = 1000
	// crucibleLava is the mB of lava a cobblestone melts into.
	CrucibleLava = 250
	// lavaGenOutput is the RE per tick the lava generator makes; it burns one
	// mB every lavaGenBurn ticks.
	LavaGenOutput = 80
	LavaGenBurn   = 2
	// tankCapacity is the mB of a fluid tank.
	tankCapacity = 16000
)

// FluidOfBucket returns the fluid of a bucket's content.
func FluidOfBucket(b item.Bucket) fluidnet.Kind {
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

// BucketOf returns the full bucket of a fluid.
func BucketOf(k fluidnet.Kind) item.Stack {
	var liq world.Liquid = block.Water{Depth: 8, Still: true}
	if k == fluidnet.Lava {
		liq = block.Lava{Depth: 8, Still: true}
	}
	return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(liq)}, 1)
}

func fluidName(lang string, k fluidnet.Kind) string {
	if k == fluidnet.Lava {
		return i18n.T(lang, "fluid.lava")
	}
	return i18n.T(lang, "fluid.water")
}

// FluidRoles implements fluidnet.Holder.
func (m Machine) FluidRoles() (source, sink bool) {
	if m.State == nil {
		return false, false
	}
	return m.State.Def.fluidSource, m.State.Def.fluidSink
}

// FillFluid implements fluidnet.Holder.
func (m Machine) FillFluid(_ fluidnet.World, _ cube.Pos, k fluidnet.Kind, mB int, simulate bool) int {
	if m.State == nil || !m.State.Def.fluidSink || k == fluidnet.None {
		return 0
	}
	if only := m.State.Def.fluidOnly; only != fluidnet.None && only != k {
		return 0
	}
	if m.State.FluidAmt > 0 && m.State.Fluid != k {
		return 0
	}
	n := min(mB, m.State.Def.FluidCap-m.State.FluidAmt)
	if n <= 0 {
		return 0
	}
	if !simulate {
		m.State.Fluid, m.State.FluidAmt = k, m.State.FluidAmt+n
	}
	return n
}

// DrainFluid implements fluidnet.Holder.
func (m Machine) DrainFluid(_ fluidnet.World, _ cube.Pos, max int, simulate bool) (fluidnet.Kind, int) {
	if m.State == nil || !m.State.Def.fluidSource || m.State.FluidAmt == 0 {
		return fluidnet.None, 0
	}
	n := min(max, m.State.FluidAmt)
	k := m.State.Fluid
	if !simulate {
		if m.State.FluidAmt -= n; m.State.FluidAmt == 0 {
			m.State.Fluid = fluidnet.None
		}
	}
	return k, n
}

// UseFluidMachine handles a player using a machine with a tank: a full bucket
// pours into the tank, an empty one takes a bucket's worth out, and the
// crucible takes its cobblestone. It returns whether the click was used.
func UseFluidMachine(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, m Machine, held item.Stack, ctx *item.UseContext) bool {
	st := m.State
	if wb, ok := held.Item().(sgitem.WoodenBucket); ok {
		// A wooden bucket carries water only.
		switch {
		case wb.Water:
			if m.FillFluid(tx, pos, fluidnet.Water, BucketMB, true) < BucketMB {
				return true
			}
			m.FillFluid(tx, pos, fluidnet.Water, BucketMB, false)
			swapBucket(s, p, tx, pos, held, ctx, item.NewStack(sgitem.WoodenBucket{}, 1))
			tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
		case st.Def.fluidSource && st.Fluid == fluidnet.Water && st.FluidAmt >= BucketMB:
			m.DrainFluid(tx, pos, BucketMB, false)
			swapBucket(s, p, tx, pos, held, ctx, item.NewStack(sgitem.WoodenBucket{Water: true}, 1))
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
		}
		return true
	}
	if b, ok := held.Item().(item.Bucket); ok {
		if k := FluidOfBucket(b); k != fluidnet.None {
			if m.FillFluid(tx, pos, k, BucketMB, true) < BucketMB {
				return true
			}
			m.FillFluid(tx, pos, k, BucketMB, false)
			if held.Count() == 1 {
				_, off := p.HeldItems()
				p.SetHeldItems(item.NewStack(item.Bucket{}, 1), off)
			} else {
				ctx.SubtractFromCount(1)
				s.GiveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(item.Bucket{}, 1))
			}
			tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
			return true
		}
		if b.Empty() && st.Def.fluidSource && st.FluidAmt >= BucketMB {
			k, _ := m.DrainFluid(tx, pos, BucketMB, false)
			full := BucketOf(k)
			if held.Count() == 1 {
				_, off := p.HeldItems()
				p.SetHeldItems(full, off)
			} else {
				ctx.SubtractFromCount(1)
				s.GiveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), full)
			}
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
			return true
		}
		return true
	}
	if st.Def.Marker == "" && len(st.Def.Input) > 0 && st.Accepts(held) {
		put := 0
		for put < held.Count() && st.Insert(held) {
			put++
		}
		if put > 0 {
			ctx.SubtractFromCount(put)
		}
		return true
	}
	return false
}

// swapBucket replaces one bucket of the held stack with other: in the hand if
// it was the only one, else the rest stays and other goes to the inventory.
func swapBucket(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack, ctx *item.UseContext, other item.Stack) {
	if held.Count() == 1 {
		_, off := p.HeldItems()
		p.SetHeldItems(other, off)
		return
	}
	ctx.SubtractFromCount(1)
	s.GiveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), other)
}

// FluidLines are the WAILA lines of a machine with a tank.
func FluidLines(lang string, st *MachineState) []string {
	if st.FluidAmt == 0 {
		return []string{fmt.Sprintf(i18n.T(lang, "fluid.lines.empty"), st.Def.FluidCap)}
	}
	return []string{fmt.Sprintf(i18n.T(lang, "fluid.lines.tank"), fluidName(lang, st.Fluid), st.FluidAmt, st.Def.FluidCap)}
}

// runCrucible melts the cobblestone in the input of an electric crucible into
// lava, every interval one block, while the tank has room.
func (st *MachineState) runCrucible() {
	slot, in, ok := st.FirstInput()
	if !ok || !st.Accepts(in) || st.FluidAmt+CrucibleLava > st.Def.FluidCap || (st.FluidAmt > 0 && st.Fluid != fluidnet.Lava) {
		st.Progress = 0
		return
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	_ = st.Inv().SetItem(slot, in.Grow(-1))
	st.Fluid, st.FluidAmt = fluidnet.Lava, st.FluidAmt+CrucibleLava
	st.Progress = 0
}

// runLavaGenerator burns the lava in its tank into energy while the buffer has room.
func runLavaGenerator(s *core.Session, st *MachineState, pos cube.Pos) {
	gen := st.GenRate()
	if st.FluidAmt > 0 && st.Energy+gen <= st.EnergyCap() {
		if st.Progress += st.BurnRate(); st.Progress >= LavaGenBurn {
			st.Progress = 0
			if st.FluidAmt--; st.FluidAmt == 0 {
				st.Fluid = fluidnet.None
			}
		}
		st.Energy += gen
		st.made += gen
		st.markWorked()
	} else if st.FluidAmt == 0 {
		st.Stuck = stuckFuel
	}
	network.Energy(s).AddSource(pos)
}

// washerWater is the mB of water washing one raw ore takes.
const washerWater = 250

// runWasher washes the raw ore in the input: with enough water in the tank it
// works an interval, and gives washerDust dusts of the metal (ore.go).
func (st *MachineState) runWasher() {
	slot, in, ok := st.FirstInput()
	if !ok || !st.Accepts(in) || st.FluidAmt < washerWater || st.Fluid != fluidnet.Water {
		st.Progress = 0
		return
	}
	if st.Progress < st.Def.Interval {
		st.TickWork()
		return
	}
	out, _ := sgitem.WasherOutput(in)
	if !st.RoomFor([]item.Stack{out}) {
		return
	}
	_ = st.Inv().SetItem(slot, in.Grow(-1))
	st.Emit(out)
	if st.FluidAmt -= washerWater; st.FluidAmt == 0 {
		st.Fluid = fluidnet.None
	}
	st.Progress = 0
}
