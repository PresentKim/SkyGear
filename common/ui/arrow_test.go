package ui

import (
	"testing"

	"github.com/presentkim/skygear/common/invui"
)

func TestArrowMovesFromTheChosenCell(t *testing.T) {
	pg := newMenuPage(invui.KeySplit)
	for i := 0; i < 3*invui.SplitColumns; i++ {
		pg.cell(i, dSlot{ID: string(rune('a' + i)), Selected: i == invui.SplitColumns+1}, invui.KindRow, invui.KindStatic)
	}
	pg.ids[invui.SlotNext] = "next"
	for _, c := range []struct {
		dx, dy int
		want   string
	}{
		{1, 0, string(rune('a' + invui.SplitColumns + 2))},
		{0, -1, "b"},
		{0, 1, string(rune('a' + 2*invui.SplitColumns + 1))},
	} {
		if got := pg.arrow(c.dx, c.dy); got != c.want {
			t.Errorf("arrow(%d, %d) = %q, want %q", c.dx, c.dy, got, c.want)
		}
	}
}

func TestArrowPagesPastTheSplitGridsEdge(t *testing.T) {
	pg := newMenuPage(invui.KeySplit)
	pg.cell(invui.SplitColumns-1, dSlot{ID: "x", Selected: true}, invui.KindRow, invui.KindStatic)
	pg.ids[invui.SlotNext] = "next"
	if got := pg.arrow(1, 0); got != "next" {
		t.Errorf("arrow right = %q", got)
	}
}
