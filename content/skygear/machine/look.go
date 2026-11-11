package machine

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Waila fills in a machine's panel (waila.Provider): its lines and item
// line, the time boost left, the lines of a kind another package describes
// (Def.Lines), what waits in its output as slots, and the cycle figures while
// the viewer sneaks. A sneaking look also shows the machine's range for a
// moment (PeekRange).
func (m Machine) Waila(v waila.View, info waila.Info) waila.Info {
	if v.Sneaking {
		PeekRange(v.Tx, v.Pos, m)
	}
	info.Lines, info.Line = MachineLines(v.Lang, m)
	if l := BoostLine(v.Lang, m.State); l != "" {
		info.Lines = append(info.Lines, l)
	}
	if m.State != nil && m.State.Def.Lines != nil {
		info.Lines = m.State.Def.Lines(v.Lang, v.Tx, v.Pos)
	}
	if c := outputContainer(m.State); c != nil {
		info.Container = c
	}
	if v.Sneaking && m.State != nil {
		info.Lines = append(info.Lines, detailLines(v.Lang, m.State)...)
	}
	return info
}

// BoostLine is the panel line of the time boost a machine has left, or "".
func BoostLine(lang string, st *State) string {
	if st == nil || st.Boost <= 0 {
		return ""
	}
	left := time.Duration(st.Boost) * time.Second / 20
	return i18n.T(lang, "fairy.hourglass.left") + textfmt.Duration(lang, left.Round(time.Second))
}

// registerLook gives the vanilla furnaces' panels the machines' progress
// arrow.
func registerLook() {
	waila.RegisterAdapter(func(_ waila.View, b world.Block, info waila.Info) waila.Info {
		switch b.(type) {
		case block.Furnace, block.BlastFurnace, block.Smoker:
			if info.Container != nil {
				info.Container.Progress = FurnaceArrow(b)
			}
		}
		return info
	})
}

// FurnaceArrow is the progress arrow of a furnace, blast furnace or smoker:
// filled to how far the item is cooked while it burns, empty when it does
// not.
func FurnaceArrow(b world.Block) item.Stack {
	var (
		lit         bool
		cook        time.Duration
		requirement = 5 * time.Second
	)
	switch f := b.(type) {
	case block.Furnace:
		lit, requirement = f.Lit, 10*time.Second
		_, _, cook = f.Durations()
	case block.BlastFurnace:
		lit = f.Lit
		_, _, cook = f.Durations()
	case block.Smoker:
		lit = f.Lit
		_, _, cook = f.Durations()
	default:
		return item.Stack{}
	}
	step := 0
	if lit {
		step = min(int(cook*10/requirement), 10)
	}
	return item.NewStack(ProgressArrow{Step: step}, 1)
}

// MachineLines are a machine's lines and item line: what it holds, with
// its icon and progress as a ring gauge ("흙 [icon] ×3 ◔"), or grey
// "비어 있음" when empty. What waits in its output shows as slots
// (outputContainer).
func MachineLines(lang string, m Machine) ([]string, *waila.ItemLine) {
	st := m.State
	if st == nil {
		return nil, nil
	}
	step := min(10, st.Progress*10/st.CycleLen())
	gauge := textfmt.GaugeGlyph(step)
	var (
		lines []string
		line  *waila.ItemLine
	)
	switch {
	case st.Def.Crafter:
		if st.Recipe == nil {
			return crafterLines(lang, st), nil
		}
		return crafterLines(lang, st)[1:], itemLine(lang, st.Recipe.Out)
	case st.Def == Kinds[KindLab]:
		// The research it makes shows with its icon on the item line.
		lines := LabLines(lang, st)
		if st.Lab == nil {
			return lines, nil
		}
		return lines[1:], itemLine(lang, item.NewStack(st.Lab.Out, st.Lab.OutCount))
	case st.Def == Kinds[KindHarvester]:
		lines = append(lines, i18n.T(lang, "look.machine.harvestCycle")+gauge)
	case st.Def == Kinds[KindGenerator]:
		return GeneratorLines(lang, st), nil
	case IsNatureGenerator(m.Kind):
		return NatureLines(lang, st), nil
	case m.Kind == KindDrawerTrim, m.Kind == KindTinker, m.Kind == KindRelicTable:
		return nil, nil
	case st.Def.Controller:
		return nil, nil
	case st.Def.FluidCap > 0:
		lines := FluidLines(lang, st)
		if _, in, ok := st.FirstInput(); ok {
			line = itemLine(lang, in)
		}
		if st.Def.Generates > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap()))
		} else {
			lines = append(lines, ConsumerLines(lang, st)...)
		}
		return lines, line
	case st.Def.Cell:
		return []string{fmt.Sprintf(i18n.T(lang, "generator.lines.energy"), st.Energy, st.EnergyCap())}, nil
	case st.Def == Kinds[KindMiner]:
		// Its output shows as a slot.
		return append(append([]string{i18n.T(lang, "look.machine.mining") + gauge}, ConsumerLines(lang, st)...), BlockedLines(lang, st)...), nil
	default:
		if _, it, ok := st.FirstInput(); ok {
			line = &waila.ItemLine{Item: it, Name: naming.ItemName(lang, it), Gauge: step, ShowGauge: true}
			if it.Count() > 1 {
				line.Text = fmt.Sprintf("§f×%d", it.Count())
			}
		} else {
			lines = append(lines, i18n.T(lang, "look.empty"))
		}
	}
	if len(st.Def.Charge) > 0 && st.Charges > 0 {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "magicsieve.lines.charged"), i18n.T(lang, "mob."+st.Mob), st.Charges))
	}
	lines = append(lines, ConsumerLines(lang, st)...)
	lines = append(lines, BlockedLines(lang, st)...)
	return lines, line
}

// detailLines are the figures the panel adds while its viewer sneaks: the
// length of a work cycle with the ticks left (the energy is in the lines).
func detailLines(lang string, st *State) []string {
	var lines []string
	if cycle := st.CycleLen(); cycle > 1 && len(st.Def.Input)+len(st.Def.Output) > 0 && !st.Def.Crafter {
		if st.Progress > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycle"), cycle, max(0, cycle-st.Progress)))
		} else {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "look.detail.cycleOnly"), cycle))
		}
	}
	return lines
}

// crafterLines say what the auto crafter does: its recipe (the first line),
// what it lacks.
func crafterLines(lang string, st *State) []string {
	if st.Recipe == nil {
		return []string{i18n.T(lang, "crafter.status.none")}
	}
	lines := []string{"§f" + naming.ItemName(lang, st.Recipe.Out)}
	if _, ok := st.CrafterPlan(st.Recipe); !ok {
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "crafter.status.missing"), CrafterMissing(lang, st)))
	} else if !Fits(st.Inv(), st.Def.Output, []item.Stack{st.Recipe.Out}) {
		lines = append(lines, i18n.T(lang, "look.blocked"))
	} else {
		lines = append(lines, i18n.T(lang, "crafter.status.working"))
	}
	return append(lines, ConsumerLines(lang, st)...)
}

// outputContainer is the output slots of the machine st as a chest's slots,
// each item with its count under it, or nil for a machine with no output or
// a drawer controller (whose lines count the drawers instead), and nil while
// the output is empty.
func outputContainer(st *State) *waila.Container {
	if st == nil || st.Def.Controller || len(st.Def.Output) == 0 {
		return nil
	}
	var items []item.Stack
	for _, slot := range st.Def.Output {
		if it, _ := st.Inv().Item(slot); !it.Empty() {
			items = append(items, it)
		}
	}
	if len(items) == 0 {
		return nil
	}
	return &waila.Container{Layout: waila.LayoutChest, Items: items}
}

// itemLine is the line of item it with its icon, its count after it.
func itemLine(lang string, it item.Stack) *waila.ItemLine {
	line := &waila.ItemLine{Item: it, Name: naming.ItemName(lang, it)}
	if it.Count() > 1 {
		line.Text = fmt.Sprintf("§f×%d", it.Count())
	}
	return line
}
