package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
)

func referenceSlotDeco(tip string) (string, string) {
	rarity := map[string]int{"§a": 1, "§b": 2, "§d": 3, "§6": 4}
	if rest, ok := strings.CutPrefix(tip, "§k§r"); ok && len(rest) >= len("§a") {
		if r, ok := rarity[rest[:len("§a")]]; ok {
			return rest, fmt.Sprintf("tex:textures/ui/sg/deco_%d_0", r)
		}
	}
	return tip, ""
}

func TestSlotDeco(t *testing.T) {
	tips := []string{"", "plain", "§k§r", "§k§r§", "§k§r§c red", "§k§r§a", "§k§r§k§r§b x", "x§k§r§a"}
	for r := slotdeco.Common; r <= slotdeco.Legendary; r++ {
		for tier := 0; tier <= slotdeco.MaxFront; tier++ {
			tips = append(tips, slotdeco.Decorate(r, tier, "이름"), slotdeco.Decorate(r, tier, "")+"\n§7lore")
		}
	}
	for _, tip := range tips {
		gotTip, gotTex := slotDeco(tip)
		wantTip, wantTex := referenceSlotDeco(tip)
		if gotTip != wantTip || gotTex != wantTex {
			t.Errorf("slotDeco(%q) = %q, %q, want %q, %q", tip, gotTip, gotTex, wantTip, wantTex)
		}
	}
}

func TestOreIcon(t *testing.T) {
	for in, want := range map[string]string{
		"":                          "",
		"20840448":                  "aux:20840448",
		"textures/ui/sg/arrow_left": "tex:textures/ui/sg/arrow_left",
	} {
		if got := oreIcon(in); got != want {
			t.Errorf("oreIcon(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUntoken(t *testing.T) {
	if got := untoken(TabOnToken + "§e도감"); got != "§e도감" {
		t.Errorf("untoken = %q", got)
	}
	if got := untoken(CreateToken + BackToken + "메뉴"); got != "메뉴" {
		t.Errorf("untoken = %q", got)
	}
}

// shown runs show with oreShown set and returns the layout and root of each
// screen it showed.
func shown(show func()) (ls []*oreLayout, roots []any) {
	oreShown = func(l *oreLayout, root any) { ls, roots = append(ls, l), append(roots, root) }
	defer func() { oreShown = nil }()
	show()
	return ls, roots
}

// listOf is the array data[key] as objects.
func listOf(t *testing.T, data map[string]any, key string) []any {
	t.Helper()
	v, ok := data[key].([]any)
	if !ok {
		t.Fatalf("%s = %v, want an array", key, data[key])
	}
	return v
}

// TestMenuScreens: menu, cards and board send the chrome, the body and their
// items.
func TestMenuScreens(t *testing.T) {
	opts := []Option{Back("뒤로", nopRun), {Label: "a", Icon: "textures/x", Run: nopRun}, Toggle("t", true, nopRun)}
	_, roots := shown(func() { oreMenu(nil, "제목", "본문", opts) })
	name, d := screenRoot(roots[0])
	items := listOf(t, d, "items")
	if name != "menu" || d["body"] != "본문" || len(items) != 2 || items[1].(map[string]any)["toggle"] != true {
		t.Errorf("menu %s %v", name, d)
	}
	if bar := listOf(t, d["chrome"].(map[string]any), "bar"); len(bar) != 1 {
		t.Errorf("bar %v", bar)
	}
	_, roots = shown(func() { oreCardMenu(nil, "제목", "", opts[1:2]) })
	name, d = screenRoot(roots[0])
	if items := listOf(t, d, "items"); name != "cards" || len(items) != 1 || items[0].(map[string]any)["icon"] != "tex:textures/x" {
		t.Errorf("cards %s %v", name, d)
	}
	_, roots = shown(func() { oreBoard(nil, "제목", "b", [][]BoardCell{{TextCell("t"), Blank()}}, opts[1:2]) })
	name, d = screenRoot(roots[0])
	rows := listOf(t, d, "rows")
	if cells := rows[0].([]any); name != "board" || len(rows) != 1 || len(cells) != 1 || cells[0].(map[string]any)["span"] != float64(BoardColumns) {
		t.Errorf("board %s %v", name, d)
	}
	_, roots = shown(func() { oreBoard(nil, "제목", "", nil, nil) })
	_, d = screenRoot(roots[0])
	listOf(t, d, "rows")
	listOf(t, d, "items")
}

// TestSieveData: the drops lose their counts, the lines count the longest
// recipe and the group name is bold.
func TestSieveData(t *testing.T) {
	drops := make([]ItemSlot, 10)
	for i := range drops {
		drops[i] = ItemSlot{Icon: "1", Count: "5"}
	}
	g := SieveGroup{Head: ItemSlot{Icon: "1"}, Name: "체", Recipes: []SieveRecipe{{Input: ItemSlot{Icon: "2"}, Mesh: ItemSlot{Icon: "3"}, Drops: drops}}}
	_, roots := shown(func() { SieveBoard(nil, "t", nil, []SieveGroup{g}) })
	name, d := screenRoot(roots[0])
	groups := listOf(t, d, "groups")
	rec := groups[0].(map[string]any)["recipes"].([]any)[0].(map[string]any)
	if name != "sieve" || d["lines"] != float64(2) || groups[0].(map[string]any)["name"] != "§l체" || len(rec["drops"].([]any)) != 10 || rec["drops"].([]any)[0].(map[string]any)["count"] != "" {
		t.Errorf("sieve %s %v", name, d)
	}
	listOf(t, d, "top")
	listOf(t, d, "items")
}

// TestSplitPrevOff: page 0 has prev "", the last page next "", and the
// arrows press Turn.
func TestSplitPrevOff(t *testing.T) {
	var went []int
	turn := func(_ *player.Player, _ *world.Tx, page int) { went = append(went, page) }
	slots := make([]ItemSlot, 60)
	for page, want := range []struct{ prev, next bool }{{false, true}, {true, true}, {true, false}} {
		ls, roots := shown(func() {
			oreSplit(nil, "t", Split{Slots: slots, Page: page, Pages: 3, Turn: turn, Detail: Detail{Icon: "1", Name: "n"}})
		})
		name, d := screenRoot(roots[0])
		if name != "split" || (d["prev"] != "") != want.prev || (d["next"] != "") != want.next || len(listOf(t, d, "slots")) != 48 {
			t.Fatalf("page %d: %s prev %q next %q", page, name, d["prev"], d["next"])
		}
		listOf(t, d, "actions")
		listOf(t, d, "rows")
		if want.next {
			ls[0].runs[d["next"].(string)](nil, nil)
		}
	}
	if len(went) != 2 || went[0] != 1 || went[1] != 2 {
		t.Errorf("turned to %v", went)
	}
}

// TestSearchRedraw: typing shows the whole search screen again with the new
// results, whose slots press, and the form's data again can be typed into.
func TestSearchRedraw(t *testing.T) {
	var picked []string
	find := func(_ *player.Player, q string) SearchResult {
		return SearchResult{Info: "결과 " + q, Sections: []GridSection{{Heading: "h", Slots: []ItemSlot{{Icon: "1", Run: func(*player.Player, *world.Tx) { picked = append(picked, q) }}}}}}
	}
	var ls []*oreLayout
	var roots []any
	oreShown = func(l *oreLayout, root any) { ls, roots = append(ls, l), append(roots, root) }
	defer func() { oreShown = nil }()
	SearchGrid(nil, "검색", "찾기", "a", find, Back("뒤로", nopRun))
	name, d := screenRoot(roots[0])
	if name != "search" || d["field"] != "search" || d["query"] != "a" || d["placeholder"] != "찾기" || d["info"] != "결과 a" {
		t.Fatalf("search %s %v", name, d)
	}
	ls[0].input(nil, nil, "b")
	ls[len(ls)-1].input(nil, nil, "c")
	if len(roots) != 3 {
		t.Fatalf("%d screens shown", len(roots))
	}
	for i, q := range []string{"b", "c"} {
		name, d = screenRoot(roots[i+1])
		sec := listOf(t, d, "sections")[0].(map[string]any)
		id := sec["slots"].([]any)[0].(map[string]any)["id"].(string)
		ls[i+1].runs[id](nil, nil)
		if name != "search" || d["query"] != q || d["info"] != "결과 "+q || picked[len(picked)-1] != q {
			t.Errorf("redraw %d: %s %v picked %v", i, name, d, picked)
		}
		if len(listOf(t, d["chrome"].(map[string]any), "bar")) != 1 {
			t.Errorf("redraw %d lost the bar", i)
		}
	}
	if ls[1].input == nil {
		t.Error("the redrawn screen cannot be typed into")
	}
}

func TestRootDebugToggle(t *testing.T) {
	root := rootOf("menu", menuData{}, nil, nil, false)
	on := rootDebug(root, true).(map[string]any)
	if on["debug"] != true || on["screen"] != "menu" {
		t.Errorf("on = %v", on)
	}
	if _, ok := root["debug"]; ok {
		t.Error("rootDebug changed its argument")
	}
	if _, ok := rootDebug(on, false).(map[string]any)["debug"]; ok {
		t.Error("debug kept after off")
	}
}
