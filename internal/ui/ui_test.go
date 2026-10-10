package ui

import "testing"

// TestUniqueLabels: repeated labels get "§r"s so each button is found.
func TestUniqueLabels(t *testing.T) {
	got := uniqueLabels([]Option{{Label: "a"}, {Label: "a"}, {Label: "b"}, {Label: "a"}})
	want := []string{"a", "a§r", "b", "a§r§r"}
	for i, o := range got {
		if o.Label != want[i] {
			t.Errorf("label %d = %q, want %q", i, o.Label, want[i])
		}
	}
}
