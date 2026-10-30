package ui

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
)

func TestForget(t *testing.T) {
	gone, kept := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{gone, kept} {
		toasts.m[id] = &toastState{}
		oreScreens.m[id] = &oreScreen{}
	}
	t.Cleanup(func() { forget(kept) })
	forget(gone)
	if _, ok := toasts.m[gone]; ok {
		t.Error("toast state kept")
	}
	if _, ok := oreScreens.m[gone]; ok {
		t.Error("ore screen kept")
	}
	if toasts.m[kept] == nil || oreScreens.m[kept] == nil {
		t.Error("another player's state dropped")
	}
}

// TestListRowText: the name is padded to a fixed width with invisible codes
// and the information follows, where the screen cuts the two apart again.
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
	if name, info := listRowParts(label); name != "세이브 1" || info != "§7오늘" {
		t.Errorf("parts %q %q", name, info)
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

// TestConfirmItems: Confirm sends a menu with two items, the first one
// running f.
func TestConfirmItems(t *testing.T) {
	ran := false
	var ls []*oreLayout
	var roots []any
	oreShown = func(l *oreLayout, root any) { ls, roots = append(ls, l), append(roots, root) }
	defer func() { oreShown = nil }()
	Confirm(nil, "t", "b", "예", func(*player.Player, *world.Tx) { ran = true })
	name, d := screenRoot(roots[0])
	items, _ := d["items"].([]any)
	if name != "menu" || len(items) != 2 {
		t.Fatalf("confirm %s %v", name, d)
	}
	ls[0].runs[items[0].(map[string]any)["id"].(string)](nil, nil)
	if !ran || items[0].(map[string]any)["label"] != "예" {
		t.Errorf("ran %v, items %v", ran, items)
	}
}
