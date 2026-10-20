package ui

import (
	"strings"
	"testing"
)

// TestSectionOptions: every section starts on a new row of the grid with its
// heading and ends on a whole row.
func TestSectionOptions(t *testing.T) {
	slots := func(n int) []ItemSlot { return make([]ItemSlot, n) }
	opts := sectionOptions([]GridSection{{Heading: "A", Slots: slots(13)}, {Heading: "B", Slots: slots(0)}, {Heading: "C", Slots: slots(11)}})
	if len(opts)%gridColumns != 0 {
		t.Fatalf("%d cells is not whole rows", len(opts))
	}
	var heads []int
	for i, o := range opts {
		if strings.HasPrefix(o.Label, head1Token) {
			heads = append(heads, i)
		}
	}
	// A: heading row + 2 rows of slots, B: heading row, C: heading row + 1 row
	if len(heads) != 3 || heads[0] != 0 || heads[1] != 3*gridColumns || heads[2] != 4*gridColumns {
		t.Fatalf("headings at %v", heads)
	}
}
