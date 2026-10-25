package ui

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestChoiceMenu: the DDUI board has the figures, a heading per group and a
// card per choice; the current card is ringed and does not press, the others
// do.
func TestChoiceMenu(t *testing.T) {
	var got *oreLayout
	var layout any
	oreShown = func(l *oreLayout, lay any) { got, layout = l, lay }
	oldOre := OreUI
	OreUI = true
	defer func() { oreShown, OreUI = nil, oldOre }()

	run := func(*player.Player, *world.Tx) {}
	ChoiceMenu(nil, "board", ChoiceBoard{
		Stats: []Stat{{Label: "a", Value: "1"}, {Label: "b", Value: "2"}},
		Note:  "note",
		Groups: []ChoiceGroup{
			{Title: "g1", Choices: []Choice{{Label: "x", Badge: "now", On: true, Run: run}, {Label: "y", Info: "why", Dim: true, Run: run}}},
			{Title: "g2", Choices: []Choice{{Label: "z", Run: run}}},
		},
	})
	if got == nil {
		t.Fatal("nothing shown")
	}
	if len(got.runs) != 2 {
		t.Errorf("%d pressable, want 2", len(got.runs))
	}
	b, _ := json.Marshal(layout)
	s := string(b)
	for _, want := range []string{"mc-stats", "mc-pick-note", `"g1"`, `"g2"`, "mc-btn mc-pick on", "mc-pick-ring", "mc-btn mc-pick dim", "now", "why"} {
		if !strings.Contains(s, want) {
			t.Errorf("layout has no %s", want)
		}
	}
	if strings.Count(s, "mc-pick-ring") != 1 {
		t.Errorf("rings: %d", strings.Count(s, "mc-pick-ring"))
	}
	if out := os.Getenv("CHOICE_LAYOUT"); out != "" {
		_ = os.WriteFile(out, b, 0o644)
	}
}
