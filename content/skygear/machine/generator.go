package machine

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
)

var (
	_ energy.Ports  = Machine{}
	_ energy.Source = Machine{}
	_ energy.Sink   = Machine{}
)

// The combustion generator: it
// burns a fuel item at a time (the furnace's fuels, with their burning time)
// and makes generatorOutput RE per tick while it burns, into a buffer. The
// buffer gives its energy to the conduits and blocks that touch it. When the
// buffer is full the fuel is not burned.
const (
	GeneratorOutput = 20
	generatorBuffer = 10000
)

// ConsumerBuffer is the RE buffer of a machine that uses energy.
const ConsumerBuffer = 2000

// CellCapacity is the RE an energy cell holds.
const CellCapacity = 100000

// stallShow is how many ticks in a row a machine has no energy before it shows
// as stopped (2 seconds).
const stallShow = 40

// consume takes the energy of one tick of work from the buffer and reports
// whether there was enough; a machine without need (or an unlimited one)
// always works.
func (st *MachineState) consume() bool {
	u := st.useCost()
	if u == 0 || st.Unlimited {
		return true
	}
	if st.free {
		// a run the snow adds: free, but only while there is energy
		return st.Energy >= u
	}
	if st.Energy < u {
		// a stop shows after stallShow ticks in a row: a machine that gets
		// energy as it uses it does not flicker "stopped"
		if st.stallTicks++; st.stallTicks >= stallShow {
			st.Stalled, st.Stuck = true, stuckPower
		}
		return false
	}
	st.Energy -= u
	st.Stalled, st.stallTicks = false, 0
	if st.Sess != nil {
		FactoryOf(st.Sess).Stats.Buckets[FactoryOf(st.Sess).Stats.Head].Use += u
	}
	return true
}

// FuelDuration is the ticks an item burns for as fuel (0: not a fuel).
func FuelDuration(st item.Stack) int {
	f, ok := st.Item().(item.Fuel)
	if !ok {
		return 0
	}
	return int(f.FuelInfo().Duration / (50 * time.Millisecond))
}

// runGenerator burns fuel and fills the buffer; it runs every tick.
func runGenerator(s *core.Session, st *MachineState, pos cube.Pos, tx *world.Tx) {
	gen, energyCap := st.GenRate(), st.EnergyCap()
	if st.Burn == 0 && st.Energy+gen <= energyCap {
		slot, fuel, ok := st.FirstInput()
		if !ok {
			st.Progress = 0
			st.Stuck = stuckFuel
			return
		}
		if d := FuelDuration(fuel); d > 0 {
			residue := fuel.Item().(item.Fuel).FuelInfo().Residue
			_ = st.Inv().SetItem(slot, fuel.Grow(-1))
			if !residue.Empty() {
				if left, _ := st.Inv().AddItem(residue); left < residue.Count() {
					s.DropAt(tx, pos.Vec3Centre(), residue)
				}
			}
			st.Burn, st.burnMax = d, d
		}
	}
	if st.Burn > 0 && st.Energy+gen <= energyCap {
		st.Energy += gen
		st.made += gen
		st.Burn = max(0, st.Burn-st.BurnRate())
		st.markWorked()
	}
	st.Progress = 0
	if st.burnMax > 0 {
		st.Progress = st.Burn * st.Def.Interval / st.burnMax
	}
	network.Energy(s).AddSource(pos)
}

// EnergyPorts implements energy.Ports: a generator gives energy, a machine
// that uses it takes energy.
func (m Machine) EnergyPorts() (source, sink bool) {
	if m.State == nil {
		return false, false
	}
	return m.State.Def.Generates > 0 || m.State.Def.Cell, m.State.Def.Use > 0 || m.State.Def.Cell
}

// ExtractEnergy implements energy.Source.
func (m Machine) ExtractEnergy(_ energy.World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	if m.State == nil || (m.State.Def.Generates == 0 && !m.State.Def.Cell) {
		return 0
	}
	n := min(max, m.State.Energy)
	if !simulate {
		m.State.Energy -= n
	}
	return max0(n)
}

// ReceiveEnergy implements energy.Sink.
func (m Machine) ReceiveEnergy(_ energy.World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	if m.State == nil || (m.State.Def.Use == 0 && !m.State.Def.Cell) {
		return 0
	}
	n := min(max, m.State.EnergyCap()-m.State.Energy)
	if !simulate && n > 0 {
		m.State.Energy += n
	}
	return max0(n)
}

func max0(n int) int { return max(n, 0) }

// GeneratorLines are the lines of a generator's WAILA panel.
func GeneratorLines(lang string, st *MachineState) []string {
	lines := []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap())}
	switch {
	case st.Burn > 0:
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "generator.lines.burning"), st.GenRate()))
	case st.Energy+st.GenRate() > st.EnergyCap():
		lines = append(lines, i18n.T(lang, "generator.lines.full"))
	default:
		lines = append(lines, i18n.T(lang, "generator.lines.noFuel"))
	}
	return lines
}

// ConsumerLines are the energy lines of a machine that uses energy.
func ConsumerLines(lang string, st *MachineState) []string {
	if st.Def.Use == 0 || st.Unlimited {
		return nil
	}
	lines := []string{fmt.Sprintf(i18n.T(lang, "generator.lines.consumer"), st.Energy, st.EnergyCap(), st.UseTotal())}
	if st.Stalled {
		lines = append(lines, i18n.T(lang, "generator.lines.stalled"))
	}
	return lines
}
