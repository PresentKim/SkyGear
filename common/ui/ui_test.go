package ui

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/internal/slottext"
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

func TestForget(t *testing.T) {
	gone, kept := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{gone, kept} {
		pending.m[id] = sentForm{title: "t"}
		toasts.m[id] = &toastState{}
		oreScreens.m[id] = &oreScreen{}
	}
	t.Cleanup(func() { forget(kept) })
	forget(gone)
	if _, ok := pending.m[gone]; ok {
		t.Error("pending kept")
	}
	if _, ok := toasts.m[gone]; ok {
		t.Error("toast state kept")
	}
	if _, ok := oreScreens.m[gone]; ok {
		t.Error("ore screen kept")
	}
	if pending.m[kept].title != "t" || toasts.m[kept] == nil || oreScreens.m[kept] == nil {
		t.Error("another player's state dropped")
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
			l := slotLabel(slotAToken, right, ItemSlot{Count: tc.count, Tip: "§f이름\n§7줄", Run: func(*player.Player, *world.Tx) {}})
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
	got := slottext.Pad("§f가나다라", slotCountBytes)
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

// TestListSplitOptions: the rows come first (the chosen one last), then the
// actions, the first of them marked as the full width one and the rest
// alternating left and right. Every row runs, the chosen one included: it
// shows the screen again so pressing it does not close the window.
func TestListSplitOptions(t *testing.T) {
	runs := 0
	run := func(*player.Player, *world.Tx) { runs++ }
	opts := listSplitOptions(ListSplit{
		Rows: []ListRow{
			{Label: "첫째", Run: run},
			{Label: "둘째", Selected: true, Run: run},
		},
		Actions: []Option{{Label: "입장", Run: run}, {Label: "초대 코드", Run: run}, {Label: "참여자 관리", Run: run}},
	})
	var labels []string
	for _, o := range opts {
		labels = append(labels, o.Label)
	}
	if len(labels) != 5 {
		t.Fatalf("labels %q", labels)
	}
	want := []string{listRowToken, listRowOnToken, listPrimaryToken, listColAToken, listColBToken}
	for i, m := range want {
		if !strings.HasPrefix(labels[i], m) {
			t.Errorf("button %d = %q, want the mark %q", i, labels[i], m)
		}
	}
	for _, o := range opts {
		if o.Run != nil {
			o.Run(nil, nil)
		}
	}
	if runs != 5 {
		t.Errorf("%d runs, want both rows and the three actions", runs)
	}
	if !strings.HasPrefix(ListSplitToken+"x", "§d§f§l") {
		t.Errorf("title marker %q", ListSplitToken)
	}
}

// TestListRowText: the name is padded to a fixed width with invisible codes
// and the information follows where the pack starts drawing the small line.
func TestListRowText(t *testing.T) {
	label := ListRowText("세이브 1", "§7오늘")
	if len(label) <= listNameBytes || !strings.HasSuffix(label, "§7오늘") {
		t.Fatalf("label %q", label)
	}
	if name := label[:listNameBytes]; !strings.HasPrefix(name, "세이브 1§r") || strings.Trim(strings.TrimSuffix(name[len("세이브 1"):], " "), "§r") != "" {
		t.Errorf("name field %q", name)
	}
	if got := ListRowText(strings.Repeat("가", 40), "x"); len(got) != listNameBytes+1 {
		t.Errorf("a long name is %d bytes, want the field and the info", len(got))
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
		{0, 3, []string{"다음 페이지 (2/3)"}},
		{1, 3, []string{"이전 페이지 (1/3)", "다음 페이지 (3/3)"}},
		{2, 3, []string{"이전 페이지 (2/3)"}},
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

// TestRowSlotsWrap: the ninth slot of a headed row and later go on the
// row's second line.
func TestRowSlotsWrap(t *testing.T) {
	if l := rowSlotOption(slotAToken, RowSlots-1, ItemSlot{}).Label; l[9:10] == "2" {
		t.Errorf("slot %d is on the second line", RowSlots-1)
	}
	if l := rowSlotOption(slotAToken, RowSlots, ItemSlot{Tip: "x"}).Label; l[9:10] != "2" || !strings.HasSuffix(l, "x") {
		t.Errorf("slot %d label %q", RowSlots, l)
	}
}

// TestCountPrefix: the count is 12 bytes, three resets and three digits.
func TestCountPrefix(t *testing.T) {
	for n, want := range map[int]string{0: "§r§r§r000", 9: "§r§r§r009", 93: "§r§r§r093", 1200: "§r§r§r999"} {
		if got := countPrefix(n); got != want || len(got) != 12 {
			t.Errorf("countPrefix(%d) = %q, want %q", n, got, want)
		}
	}
}
