package ui

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// shownScreen runs show with oreShown set and gives the layout and the
// screen it showed.
func shownScreen(show func()) (l *oreLayout, name string, data map[string]any) {
	oreShown = func(got *oreLayout, root any) {
		l = got
		name, data = screenRoot(root)
	}
	defer func() { oreShown = nil }()
	show()
	return
}

// TestInputScreen: the field is "text" holding def, the submit button is
// labelled create under CreateToken, and a click with the text runs f with it.
func TestInputScreen(t *testing.T) {
	var got string
	f := func(_ *player.Player, _ *world.Tx, v string) { got = v }
	l, name, data := shownScreen(func() { oreInput(nil, CreateToken+"t", "label", "def", f, nopRun) })
	if name != "input" || data["field"] != "text" || data["value"] != "def" || data["label"] != "label" {
		t.Errorf("input %s %v", name, data)
	}
	submit := data["submit"].(map[string]any)
	if submit["label"] == "" || submit["label"] != oreInputLabel(t, true) {
		t.Errorf("submit %v", submit)
	}
	if bar := data["chrome"].(map[string]any)["bar"].([]any); len(bar) != 1 {
		t.Errorf("bar %v", bar)
	}
	l.values["text"] = "abc"
	l.runs[submit["id"].(string)](nil, nil)
	if got != "abc" {
		t.Errorf("f got %q", got)
	}
	_, _, plain := shownScreen(func() { oreInput(nil, "t", "", "", f, nil) })
	if plain["submit"].(map[string]any)["label"] == submit["label"] {
		t.Errorf("plain submit %v", plain["submit"])
	}
}

// oreInputLabel is the submit label of an input screen shown for a title with
// or without CreateToken.
func oreInputLabel(t *testing.T, create bool) string {
	t.Helper()
	title := "t"
	if create {
		title = CreateToken + "t"
	}
	_, _, data := shownScreen(func() { oreInput(nil, title, "", "", nil, nil) })
	return data["submit"].(map[string]any)["label"].(string)
}

// TestListSplitScreen: a row label arrives split into name and info, and the
// actions are buttons.
func TestListSplitScreen(t *testing.T) {
	_, name, data := shownScreen(func() {
		oreListSplit(nil, "t", ListSplit{
			Rows:    []ListRow{{Label: ListRowText("§f이름", "§a정보"), Icon: "textures/x", Selected: true, Run: nopRun}, {Label: "plain"}},
			Body:    "body",
			Actions: []Option{{Label: "go", Run: nopRun}, Back("b", nopRun)},
		})
	})
	if name != "listsplit" || data["body"] != "body" {
		t.Fatalf("listsplit %s %v", name, data)
	}
	rows := data["rows"].([]any)
	r0, r1 := rows[0].(map[string]any), rows[1].(map[string]any)
	if r0["name"] != "§f이름" || r0["info"] != "§a정보" || r0["icon"] != "tex:textures/x" || r0["selected"] != true || r0["id"] == "" {
		t.Errorf("row0 %v", r0)
	}
	if r1["name"] != "plain" || r1["info"] != "" || r1["id"] != "" {
		t.Errorf("row1 %v", r1)
	}
	if len(data["actions"].([]any)) != 1 || len(data["chrome"].(map[string]any)["bar"].([]any)) != 1 {
		t.Errorf("actions %v", data)
	}
}

// TestGridScreen: sections of slots, and the other options as items.
func TestGridScreen(t *testing.T) {
	_, name, data := shownScreen(func() {
		oreItemGrid(nil, "t", "body", []GridSection{{Heading: "h", Slots: []ItemSlot{{Icon: "1", Run: nopRun}}}, {}},
			[]Option{{Label: "x", Run: nopRun}})
	})
	secs := data["sections"].([]any)
	if name != "grid" || data["body"] != "body" || len(secs) != 2 || len(data["items"].([]any)) != 1 {
		t.Fatalf("grid %s %v", name, data)
	}
	if s := secs[0].(map[string]any); s["heading"] != "h" || len(s["slots"].([]any)) != 1 {
		t.Errorf("section %v", s)
	}
	if s := secs[1].(map[string]any); s["slots"] == nil {
		t.Errorf("empty section %v", s)
	}
}

// TestSlotMenuScreen: the store slot shows its text as count, the others as
// tooltip.
func TestSlotMenuScreen(t *testing.T) {
	_, name, data := shownScreen(func() {
		oreSlotMenu(nil, "t", Slot{Icon: "1", Text: "x64"}, "info",
			[]Slot{{Icon: "2", Text: "tip", Run: nopRun}}, nil, []Option{{Label: "a", Run: nopRun}})
	})
	if name != "slotmenu" || data["info"] != "info" {
		t.Fatalf("slotmenu %s %v", name, data)
	}
	store := data["store"].(map[string]any)
	left := data["left"].([]any)
	if store["count"] != "x64" || store["id"] != "" || len(left) != 1 || left[0].(map[string]any)["tip"] != "tip" {
		t.Errorf("slots %v %v", store, left)
	}
	if data["right"] == nil || len(data["items"].([]any)) != 1 || strings.Contains(name, "null") {
		t.Errorf("lists %v", data)
	}
}
