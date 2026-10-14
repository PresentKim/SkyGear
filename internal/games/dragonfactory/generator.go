package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// The combustion generator (docs/design/06-power-and-conduits.md §2): it
// burns a fuel item at a time (the furnace's fuels, with their burning time)
// and makes generatorOutput RE per tick while it burns, into a buffer. The
// buffer gives its energy to the conduits and blocks that touch it. When the
// buffer is full the fuel is not burned.
const (
	generatorOutput = 20
	generatorBuffer = 10000
)

// consumerBuffer is the RE buffer of a machine that uses energy.
const consumerBuffer = 2000

// consume takes the energy of one tick of work from the buffer and reports
// whether there was enough; a machine without need (or an unlimited one)
// always works.
func (st *machineState) consume() bool {
	u := st.kind.use
	if u == 0 || st.unlimited {
		return true
	}
	if st.energy < u {
		st.stalled = true
		return false
	}
	st.energy -= u
	st.stalled = false
	return true
}

// fuelDuration is the ticks an item burns for as fuel (0: not a fuel).
func fuelDuration(st item.Stack) int {
	f, ok := st.Item().(item.Fuel)
	if !ok {
		return 0
	}
	return int(f.FuelInfo().Duration / (50 * time.Millisecond))
}

// runGenerator burns fuel and fills the buffer; it runs every tick.
func (s *session) runGenerator(st *machineState, pos cube.Pos, tx *world.Tx) {
	k := st.kind
	if st.burn == 0 && st.energy+k.generates <= k.energyCap {
		slot, fuel, ok := st.firstInput()
		if !ok {
			st.progress = 0
			return
		}
		if d := fuelDuration(fuel); d > 0 {
			residue := fuel.Item().(item.Fuel).FuelInfo().Residue
			_ = st.inv().SetItem(slot, fuel.Grow(-1))
			if !residue.Empty() {
				if left, _ := st.inv().AddItem(residue); left < residue.Count() {
					s.dropAt(tx, pos.Vec3Centre(), residue)
				}
			}
			st.burn, st.burnMax = d, d
		}
	}
	if st.burn > 0 && st.energy+k.generates <= k.energyCap {
		st.energy += k.generates
		st.burn--
	}
	st.progress = 0
	if st.burnMax > 0 {
		st.progress = st.burn * k.interval / st.burnMax
	}
	s.energyGrid().AddSource(pos)
}

// EnergyPorts implements energy.Ports: a generator gives energy, a machine
// that uses it takes energy.
func (m Machine) EnergyPorts() (source, sink bool) {
	if m.st == nil {
		return false, false
	}
	return m.st.kind.generates > 0, m.st.kind.use > 0
}

// ExtractEnergy implements energy.Source.
func (m Machine) ExtractEnergy(_ energy.World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	if m.st == nil || m.st.kind.generates == 0 {
		return 0
	}
	n := min(max, m.st.energy)
	if !simulate {
		m.st.energy -= n
	}
	return max0(n)
}

// ReceiveEnergy implements energy.Sink.
func (m Machine) ReceiveEnergy(_ energy.World, _ cube.Pos, _ cube.Face, max int, simulate bool) int {
	if m.st == nil || m.st.kind.use == 0 {
		return 0
	}
	n := min(max, m.st.kind.energyCap-m.st.energy)
	if !simulate && n > 0 {
		m.st.energy += n
	}
	return max0(n)
}

func max0(n int) int { return max(n, 0) }

var (
	_ energy.Ports  = Machine{}
	_ energy.Source = Machine{}
	_ energy.Sink   = Machine{}
)

// generatorLines are the lines of a generator's WAILA panel.
func generatorLines(st *machineState) []string {
	lines := []string{fmt.Sprintf(i18n.T(i18n.Default, "generator.lines.energy"), st.energy, st.kind.energyCap)}
	switch {
	case st.burn > 0:
		lines = append(lines, fmt.Sprintf(i18n.T(i18n.Default, "generator.lines.burning"), st.kind.generates))
	case st.energy+st.kind.generates > st.kind.energyCap:
		lines = append(lines, i18n.T(i18n.Default, "generator.lines.full"))
	default:
		lines = append(lines, i18n.T(i18n.Default, "generator.lines.noFuel"))
	}
	return lines
}

// consumerLines are the energy lines of a machine that uses energy.
func consumerLines(st *machineState) []string {
	if st.kind.use == 0 || st.unlimited {
		return nil
	}
	lines := []string{fmt.Sprintf(i18n.T(i18n.Default, "generator.lines.consumer"), st.energy, st.kind.energyCap, st.kind.use)}
	if st.stalled {
		lines = append(lines, i18n.T(i18n.Default, "generator.lines.stalled"))
	}
	return lines
}
