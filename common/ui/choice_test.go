package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestChoiceMenu: the screen has the figures, a group per heading and a tile
// per choice; the current tile has no id, the others do.
func TestChoiceMenu(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	defer func() { oreShown = nil }()

	run := func(*player.Player, *world.Tx) {}
	ChoiceMenu(nil, "board", ChoiceBoard{
		Stats: []Stat{{Label: "a", Value: "1"}, {Label: "b", Value: "2"}},
		Note:  "note",
		Groups: []ChoiceGroup{
			{Title: "g1", Choices: []Choice{{Label: "x", Badge: "now", On: true, Run: run}, {Label: "y", Info: "why", Dim: true, Run: run}}},
			{Title: "g2", Choices: []Choice{{Label: "z", Run: run}}},
		},
		Actions: []Option{{Label: "more", Run: run}},
	})
	name, _ := screenRoot(layout)
	if name != "choice" {
		t.Fatalf("screen %q", name)
	}
	if len(got.runs) != 3 {
		t.Errorf("%d pressable, want 3", len(got.runs))
	}
	b, _ := json.Marshal(layout.(map[string]any)["data"])
	var d choiceData
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Stats) != 2 || d.Note != "note" || len(d.Groups) != 2 || len(d.Items) != 1 {
		t.Fatalf("data %s", b)
	}
	x, y := d.Groups[0].Choices[0], d.Groups[0].Choices[1]
	if !x.On || x.ID != "" || !strings.Contains(x.Tip, "now") {
		t.Errorf("current %+v", x)
	}
	if y.On || y.ID == "" || !y.Dim || !strings.Contains(y.Tip, "why") {
		t.Errorf("other %+v", y)
	}
	if strings.Contains(string(b), "null") {
		t.Errorf("null in %s", b)
	}
}
