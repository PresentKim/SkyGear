package ui

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

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

// TestSlotLabelFields: the pack cuts a slot's button text at fixed bytes: a
// 9 byte marker, 1 byte of tooltip side, 10 bytes of count padded with '~',
// then the tooltip.
func TestSlotLabelFields(t *testing.T) {
	if len(slotAToken) != 9 {
		t.Fatalf("marker is %d bytes", len(slotAToken))
	}
	for _, tc := range []struct{ count, wantCount string }{
		{"", "~~~~~~~~~~"},
		{"64", "§f64~~~~~"},
		{"§a12", "§a12~~~~~"},
		{"§e1234567", "§e1234567"},
		{"§e123456789", "§e1234567"},
	} {
		for _, right := range []bool{false, true} {
			l := slotLabel(slotAToken, right, ItemSlot{Count: tc.count, Tip: "§f이름\n§7줄"})
			side := map[bool]string{false: "l", true: "r"}[right]
			if got := l[9:10]; got != side {
				t.Errorf("tooltip side %q, want %q", got, side)
			}
			if got := l[10:20]; got != tc.wantCount {
				t.Errorf("count %q: field %q, want %q", tc.count, got, tc.wantCount)
			}
			if got := l[20:]; got != "§f이름\n§7줄" {
				t.Errorf("count %q: tooltip %q", tc.count, got)
			}
		}
	}
}

// TestSlotCountKeepsWholeCharacters: a cut never splits a character.
func TestSlotCountKeepsWholeCharacters(t *testing.T) {
	got := padSlotCount("§f가나다라", slotCountBytes)
	if len(got) != slotCountBytes {
		t.Fatalf("%q is %d bytes", got, len(got))
	}
	if !strings.HasPrefix(got, "§f가나") {
		t.Errorf("%q", got)
	}
}

// TestActionOptionsKeepBack: a Back button keeps its marker; others get the
// action marker.
func TestActionOptionsKeepBack(t *testing.T) {
	out := actionOptions([]Option{{Label: "납품"}, Back("뒤로", nil)})
	if !strings.HasPrefix(out[0].Label, ActionToken) || !strings.HasPrefix(out[1].Label, BackToken) {
		t.Errorf("labels %q %q", out[0].Label, out[1].Label)
	}
}

// TestPager: a list on its first page has only "next", in the middle both, on
// the last only "previous"; the labels name the page they go to.
func TestPager(t *testing.T) {
	var went []int
	show := func(_ *player.Player, _ *world.Tx, page int) { went = append(went, page) }
	for _, tc := range []struct {
		page, pages int
		labels      []string
	}{
		{0, 1, nil},
		{0, 3, []string{"다음 쪽 (2/3)"}},
		{1, 3, []string{"이전 쪽 (1/3)", "다음 쪽 (3/3)"}},
		{2, 3, []string{"이전 쪽 (2/3)"}},
	} {
		opts := Pager(nil, tc.page, tc.pages, show)
		if len(opts) != len(tc.labels) {
			t.Fatalf("page %d/%d: %d buttons, want %d", tc.page, tc.pages, len(opts), len(tc.labels))
		}
		for i, o := range opts {
			if o.Label != tc.labels[i] {
				t.Errorf("page %d/%d button %d = %q, want %q", tc.page, tc.pages, i, o.Label, tc.labels[i])
			}
			o.Run(nil, nil)
		}
	}
	if len(went) != 4 || went[0] != 1 || went[1] != 0 || went[2] != 2 || went[3] != 1 {
		t.Errorf("the buttons went to pages %v", went)
	}
}
