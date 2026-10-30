package ui

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// screenRoot is the screen name and the data (as JSON objects) of a root
// that showScreen gave oreShown.
func screenRoot(root any) (name string, data map[string]any) {
	b, err := json.Marshal(root)
	if err != nil {
		panic(err)
	}
	var m struct {
		Screen string         `json:"screen"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		panic(err)
	}
	return m.Screen, m.Data
}

func nopRun(*player.Player, *world.Tx) {}

// TestArraysNotNull: empty lists are sent as [].
func TestArraysNotNull(t *testing.T) {
	b, _ := json.Marshal(dChrome{})
	if !strings.Contains(string(b), `"tabs":[]`) || !strings.Contains(string(b), `"bar":[]`) {
		t.Errorf("chrome %s", b)
	}
	l := newOreLayout()
	for _, v := range []any{l.buttonsOf(nil), l.itemsOf(nil), l.rowsOf(nil), l.rowsOf([]SlotRow{{Heading: "h"}}), l.cellsOf(nil)} {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "null") {
			t.Errorf("null in %s", b)
		}
	}
	c, rest := l.chromeOf("t", nil)
	if b, _ := json.Marshal(c); strings.Contains(string(b), "null") || rest != nil {
		t.Errorf("chrome %s", b)
	}
}

// TestChromeOf: tabs, bar order and the rest.
func TestChromeOf(t *testing.T) {
	l := newOreLayout()
	c, rest := l.chromeOf("§0§2§r", []Option{Tab("A", "textures/x", true, nil), Back("뒤로", nopRun), {Label: "x"}})
	want := dChrome{
		Tabs: []dTab{{Tip: "A", Icon: "tex:textures/x", On: true}},
		Bar:  []dBarButton{{ID: "o0", Tip: "뒤로", Icon: "tex:textures/ui/sg/chevron_left", Quick: true}},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("chrome %+v", c)
	}
	if len(rest) != 1 || rest[0].Label != "x" {
		t.Errorf("rest %v", rest)
	}
	c, _ = l.chromeOf("", []Option{Back("b", nopRun), Help("h", nopRun), BarButton("p", "textures/y", nopRun), Tab("T", "", false, nopRun)})
	if len(c.Tabs) != 1 || c.Tabs[0].ID == "" {
		t.Errorf("tab %+v", c.Tabs)
	}
	if len(c.Bar) != 3 || c.Bar[0].Tip != "p" || c.Bar[1].Glyph != "?" || c.Bar[1].Quick || c.Bar[2].Tip != "b" {
		t.Errorf("bar %+v", c.Bar)
	}
}

// TestSlotOf: tooltip, rarity and id.
func TestSlotOf(t *testing.T) {
	l := newOreLayout()
	got := l.slotOf(ItemSlot{Icon: "1", Tip: "§k§r§a돌", Run: nopRun})
	want := dSlot{ID: "o0", Icon: "aux:1", Tip: "§a돌", Deco: "tex:textures/ui/sg/deco_1_0"}
	if got != want {
		t.Errorf("slot %+v", got)
	}
	if s := l.slotOf(ItemSlot{Run: nopRun, Selected: true}); s.ID != "" || !s.Selected {
		t.Errorf("selected %+v", s)
	}
	if s := l.slotOf(ItemSlot{}); s.ID != "" {
		t.Errorf("no run %+v", s)
	}
}

// TestCellsOf: a text cell spans the blanks after it, the row has 11 columns.
func TestCellsOf(t *testing.T) {
	l := newOreLayout()
	s := ItemSlot{Run: nopRun}
	cells := l.cellsOf([]BoardCell{TextCell("t"), Blank(), Blank(), SlotCell(s)})
	if len(cells) != 1+1+7 {
		t.Fatalf("%d cells", len(cells))
	}
	if cells[0].K != "text" || cells[0].Text != "t" || cells[0].Span != 3 {
		t.Errorf("text %+v", cells[0])
	}
	if cells[1].K != "slot" || cells[1].Slot == nil || cells[1].Slot.ID != "o0" {
		t.Errorf("slot %+v", cells[1])
	}
	for _, c := range cells[2:] {
		if c.K != "blank" {
			t.Errorf("cell %+v", c)
		}
	}
	m := l.cellsOf([]BoardCell{MiniCell(s), Frame(), Arrow()})
	if m[0].K != "mini" || m[1].K != "frame" || m[2].K != "arrow" {
		t.Errorf("cells %+v", m[:3])
	}
}

// TestItemsOf: a toggle row carries its state.
func TestItemsOf(t *testing.T) {
	l := newOreLayout()
	it := l.itemsOf([]Option{Toggle("t", true, nopRun), {Label: "a", Run: nopRun}})
	if !it[0].Toggle || !it[0].On || it[1].Toggle {
		t.Errorf("items %+v", it)
	}
	b, _ := json.Marshal(it[1])
	if string(b) != `{"id":"o1","label":"a","icon":""}` {
		t.Errorf("json %s", b)
	}
}

// TestIDsUnique: ids from one layout never repeat.
func TestIDsUnique(t *testing.T) {
	l := newOreLayout()
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		id := l.id(nopRun)
		if id == "" || seen[id] {
			t.Fatalf("id %q", id)
		}
		seen[id] = true
	}
	if l.id(nil) != "" {
		t.Error("nil run got an id")
	}
}

// shownRoot runs show and returns the root it gave oreShown.
func shownRoot(show func()) (root map[string]any) {
	oreShown = func(_ *oreLayout, layout any) { root = layout.(map[string]any) }
	defer func() { oreShown = nil }()
	show()
	return root
}

// TestShowScreenRoot: keys, lobby and the size go into the root.
func TestShowScreenRoot(t *testing.T) {
	l := newOreLayout()
	l.key(KeyOK, nopRun)
	root := shownRoot(func() { showScreen(nil, l, "menu", map[string]any{"body": "b"}) })
	if root["screen"] != "menu" || root["keys"].(map[string]string)["ok"] != "o0" {
		t.Errorf("root %v", root)
	}
	if _, ok := root["lobby"]; ok {
		t.Error("lobby outside the lobby")
	}
	if _, ok := root["debug"]; ok {
		t.Error("debug set")
	}
	if l.size != "326x166" {
		t.Errorf("size %q", l.size)
	}
	name, data := screenRoot(root)
	if name != "menu" || data["body"] != "b" {
		t.Errorf("screenRoot %q %v", name, data)
	}

	inLobby, header := InLobby, LobbyHeader
	InLobby = func(*player.Player) bool { return true }
	LobbyHeader = func(*player.Player) (string, string) { return "SkyGear", "textures/ui/confirm" }
	defer func() { InLobby, LobbyHeader = inLobby, header }()
	root = shownRoot(func() { showScreen(nil, newOreLayout(), "menu", struct{}{}) })
	if got, want := root["lobby"], (&dLobby{Title: "SkyGear", Logo: "tex:textures/ui/confirm"}); !reflect.DeepEqual(got, want) {
		t.Errorf("lobby %+v", got)
	}
	if _, ok := root["keys"]; ok {
		t.Error("keys without keys")
	}
	b, _ := json.Marshal(root)
	if strings.Contains(string(b), "mc-lobby") {
		t.Errorf("legacy lobby node in %s", b)
	}
}

// TestRootDebug: the debug flag goes into the root.
func TestRootDebug(t *testing.T) {
	if r := rootOf("menu", nil, nil, nil, true); r["debug"] != true {
		t.Errorf("root %v", r)
	}
}
