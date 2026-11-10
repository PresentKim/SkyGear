package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestRadialMenu: the screen has a sector per item, pressable when it has a
// run, and the middle button closes the menu unless it has a Back.
func TestRadialMenu(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	defer func() { oreShown = nil }()

	run := func(*player.Player, *world.Tx) {}
	m := RadialMenu{Title: "wand", Line: "9", Info: "pick", Items: []RadialItem{
		{Label: "lock", Value: "none", Info: "why", Icon: "textures/ui/sg/x", Run: run},
		{Label: "dim"},
		{Label: "on", On: true, Run: run},
	}}
	ShowRadial(nil, m)
	name, _ := screenRoot(layout)
	if name != "radial" || got.size != "radial" {
		t.Fatalf("screen %q size %q", name, got.size)
	}
	b, _ := json.Marshal(layout.(map[string]any)["data"])
	var d radialData
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.Title != "wand" || d.Line != "9" || d.Info != "pick" || len(d.Items) != 3 || d.Back != "close" {
		t.Fatalf("data %s", b)
	}
	if d.Items[0].ID == "" || d.Items[0].Value != "none" || d.Items[0].Info != "why" || d.Items[0].Icon == "" {
		t.Errorf("first %+v", d.Items[0])
	}
	if d.Items[1].ID != "" || !d.Items[2].On {
		t.Errorf("items %+v", d.Items)
	}
	if strings.Contains(string(b), "null") {
		t.Errorf("null in %s", b)
	}

	m.Back = run
	ShowRadial(nil, m)
	b, _ = json.Marshal(layout.(map[string]any)["data"])
	_ = json.Unmarshal(b, &d)
	if d.Back == "" || d.Back == "close" || got.runs[d.Back] == nil {
		t.Errorf("back %q", d.Back)
	}
}
