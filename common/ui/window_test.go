package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// shownLayouts runs show and returns the layouts it
// showed with their window sizes.
func shownLayouts(t *testing.T, show func()) (sizes []string, layouts []any) {
	t.Helper()
	oreShown = func(l *oreLayout, layout any) {
		sizes = append(sizes, l.size)
		layouts = append(layouts, layout)
	}
	defer func() { oreShown = nil }()
	show()
	return sizes, layouts
}

// windowScreens are a screen of every kind drawn in the window.
func windowScreens() map[string]func() {
	var p *player.Player // the layouts need no player
	nop := func(*player.Player, *world.Tx) {}
	opts := []Option{{Label: "하나", Icon: "textures/ui/confirm", Run: nop}, {Label: "둘", Run: nop}, {Label: "셋", Run: nop}, {Label: "넷", Run: nop}}
	slot := ItemSlot{Icon: "textures/ui/confirm", Count: "3", Tip: "tip", Run: nop}
	var slots []ItemSlot
	for range 30 {
		slots = append(slots, slot)
	}
	return map[string]func(){
		"menu":  func() { Menu(p, "메뉴", "본문", opts...) },
		"cards": func() { CardMenu(p, "카드", "", opts...) },
		"board": func() {
			ItemBoard(p, "판", "본문", [][]BoardCell{{SlotCell(slot), SlotCell(slot), Arrow(), SlotCell(slot)}}, opts[:2]...)
		},
		"sieve": func() { SieveBoard(p, "체", [][]BoardCell{{SlotCell(slot)}}, nil) },
		"split": func() { SplitMenu(p, "분할", Split{Slots: slots}) },
		"listsplit": func() {
			ListSplitMenu(p, "친구", ListSplit{
				Rows: []ListRow{
					{Label: ListRowText("§f친구 하나", "§a접속 중"), Icon: "textures/ui/confirm", Selected: true, Run: nop},
					{Label: ListRowText("친구 둘", "§7오프라인"), Icon: "textures/ui/confirm", Run: nop},
				},
				Body:    "본문",
				Actions: append([]Option{Tab("친구", "textures/ui/confirm", true, nop), Tab("요청", "textures/ui/confirm", false, nop)}, opts...),
			})
		},
		"input": func() { Input(p, "이름", "새 이름", "기본", func(*player.Player, *world.Tx, string) {}, nop) },
		"grid": func() {
			ItemGridSections(p, "격자", "본문", []GridSection{{Heading: "머리", Slots: slots}}, opts[:2]...)
		},
		"slotmenu": func() {
			SlotMenu(p, "서랍", Slot{Icon: "textures/ui/confirm", Text: "64"}, "§f돌\n§764 / 1024", []Slot{{Icon: "textures/ui/confirm", Text: "tip"}}, []Slot{{Icon: "textures/ui/confirm"}})
		},
		"sides": func() { SidesMenu(p, "면", SideBoard{}) },
		"quest": func() {
			QuestBookMenu(p, "발전과제", QuestBook{
				Chapters: []QuestChapter{{Name: "시작", Icon: "textures/ui/confirm", Count: "1/2", On: true, Run: nop}},
				Nodes:    []QuestNode{{ID: "a", Icon: "textures/ui/confirm", State: QuestDone}, {ID: "b", X: 1, Icon: "textures/ui/confirm", State: QuestOpen, Deps: []string{"a"}}},
				Key:      "q",
			}, Option{Label: BackToken + "뒤로", Run: nop})
		},
		"search": func() {
			SearchGrid(p, "검색", "찾기", "", func(*player.Player, string) SearchResult {
				return SearchResult{Sections: []GridSection{{Slots: slots}}}
			})
		},
	}
}

// TestWindowSize checks that every screen drawn in the window has the same
// size, so going from one to another keeps the screen open.
func TestWindowSize(t *testing.T) {
	dir := os.Getenv("OREUI_LAYOUTS") // writes the layouts there, to look at
	for name, show := range windowScreens() {
		sizes, layouts := shownLayouts(t, show)
		if len(sizes) != 1 || sizes[0] != "326x166" {
			t.Errorf("%s: window sizes %v, want one 326x166", name, sizes)
			continue
		}
		if dir != "" {
			b, _ := json.Marshal(layouts[0])
			if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// childWithClass is the first child of n whose class is class.
func childWithClass(n map[string]any, class string) map[string]any {
	kids, _ := n["children"].([]any)
	for _, k := range kids {
		if m, ok := k.(map[string]any); ok && m["class"] == class {
			return m
		}
	}
	return nil
}

// TestQuestLinesDoneOnTop: of two lines leaving one quest, the done one is
// drawn after the locked one, so the shared piece shows done.
func TestQuestLinesDoneOnTop(t *testing.T) {
	segs := questLines([]QuestNode{
		{ID: "a", State: QuestDone},
		{ID: "c", X: 1, Y: 1, State: QuestLocked, Deps: []string{"a"}},
		{ID: "b", X: 1, State: QuestDone, Deps: []string{"a"}},
	})
	if last := segs[len(segs)-1]; last.class != "oui-line-seg done" {
		t.Fatalf("last line %q", last.class)
	}
}
