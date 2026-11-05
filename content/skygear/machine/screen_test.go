package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/ui"
)

func TestMachineScreenColumns(t *testing.T) {
	for n, want := range map[int]int{1: 1, 2: 2, 4: 2, 9: 3} {
		if got := cols(n); got != want {
			t.Errorf("%d slots: %d columns, want %d", n, got, want)
		}
	}
}

func TestMachineScreenHeight(t *testing.T) {
	slots := func(n, cols int) ui.InvPart { return ui.InvPart{Cols: cols, Slots: make([]ui.InvSlot, n)} }
	if got := windowHeight([]ui.InvPart{slots(1, 1), slots(1, 1)}); got != windowRest+slotH {
		t.Errorf("one slot row: %d", got)
	}
	if got := windowHeight([]ui.InvPart{slots(9, 3), slots(4, 2)}); got != windowRest+3*slotH {
		t.Errorf("3x3 input: %d", got)
	}
	if got := windowHeight([]ui.InvPart{{Gauge: &ui.InvGauge{}}, slots(1, 1)}); got != windowRest+gaugeH {
		t.Errorf("gauge: %d", got)
	}
}

func TestMachineScreenRedrawsOnlyOnVisibleChange(t *testing.T) {
	st := New(KindCrusher, "", "").State
	shown := st.screenLook()
	st.Progress = st.CycleLen() / 20
	if st.screenLook() != shown {
		t.Fatal("a step under 10% redraws the window")
	}
	st.Progress = st.CycleLen() / 2
	if st.screenLook() == shown {
		t.Fatal("half done does not redraw the window")
	}
	shown = st.screenLook()
	_ = st.Inv().SetItem(st.Def.Input[0], item.NewStack(block.Cobblestone{}, 3))
	if st.screenLook() == shown {
		t.Fatal("an input change does not redraw the window")
	}
}

func TestMachineScreenHolds(t *testing.T) {
	st := New(KindCrusher, "", "").State
	cobble := item.NewStack(block.Cobblestone{}, 1)
	if st.holds(cobble) {
		t.Fatal("an empty crusher holds cobblestone")
	}
	_ = st.Inv().SetItem(st.Def.Input[0], cobble.Grow(4))
	if !st.holds(cobble) || st.holds(item.NewStack(block.Gravel{}, 1)) {
		t.Fatal("holds does not follow the input slot")
	}
}
