package ui

import (
	"strings"
	"testing"
)

// TestBoardOptions: every row is BoardColumns cells, each with the marker of
// its kind, and a selected slot is the chosen slot.
func TestBoardOptions(t *testing.T) {
	rows := [][]BoardCell{
		{SlotCell(ItemSlot{Tip: "a"}), Frame(), Arrow(), TextCell("head")},
		{SlotCell(ItemSlot{Tip: "b", Selected: true})},
	}
	opts := boardOptions(rows)
	if len(opts) != 2*BoardColumns {
		t.Fatalf("got %d cells, want %d", len(opts), 2*BoardColumns)
	}
	want := map[int]string{0: slotWToken, 1: slotEmptyToken, 2: arrowToken, 3: head1Token, 4: padToken, BoardColumns: slotSelToken, BoardColumns + 1: padToken}
	for i, marker := range want {
		if !strings.HasPrefix(opts[i].Label, marker) {
			t.Errorf("cell %d %q does not start with %q", i, opts[i].Label, marker)
		}
	}
	if !strings.HasSuffix(opts[3].Label, "head") {
		t.Errorf("text cell %q lost its text", opts[3].Label)
	}
}
