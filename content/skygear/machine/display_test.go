package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

// displaySlots are the window slots the display writes.
func displaySlots(st *State) map[int]bool {
	return map[int]bool{st.Def.Indicator: true, EnergyGaugeSlot: true, infoSlot: true}
}

// TestNoDisplayWithoutViewers: ticks nobody watches do not write the display.
func TestNoDisplayWithoutViewers(t *testing.T) {
	st := New(KindFurnace).State
	slots := displaySlots(st)
	writes := 0
	st.storage.OnChange(func(slot int, _, _ item.Stack) {
		if slots[slot] {
			writes++
		}
	})
	for tick := int64(0); tick < 100; tick++ {
		st.Progress, st.Energy = int(tick%10)*st.CycleLen()/10, int(tick)
		st.tickDisplay(tick, false)
	}
	if writes != 0 {
		t.Fatalf("%d display writes without a viewer", writes)
	}
}

// TestDisplayFreshOnOpen: a machine that worked unwatched shows its current
// progress and energy the moment a window opens.
func TestDisplayFreshOnOpen(t *testing.T) {
	st := New(KindFurnace).State
	st.Progress, st.Energy = st.CycleLen()/2, st.EnergyCap()/2
	shown := st.ShownPct
	st.tickDisplay(10, false)
	if st.ShownPct != shown {
		t.Fatalf("shown pct moved while unwatched: %d", st.ShownPct)
	}
	beforeInd, _ := st.Inv().Item(st.Def.Indicator)
	beforeGauge, _ := st.Inv().Item(EnergyGaugeSlot)
	st.prepareDisplay()
	ind, _ := st.Inv().Item(st.Def.Indicator)
	gauge, _ := st.Inv().Item(EnergyGaugeSlot)
	if ind.Comparable(beforeInd) && ind.Item() == beforeInd.Item() {
		t.Fatalf("indicator not refreshed: %v", ind)
	}
	if st.Def.EnergyCap > 0 && st.Def.Marker != "" && gauge.Item() == beforeGauge.Item() {
		t.Fatalf("gauge not refreshed: %v", gauge)
	}
	if want := (ProgressArrow{Step: 5}); ind.Item() != want {
		t.Fatalf("indicator %v, want %v", ind.Item(), want)
	}
}
