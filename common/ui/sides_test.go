package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// sidesBoard is a board whose sides all have one port of three choices and
// are named by their cube.Face; chosen records the choice a Run set.
func sidesBoard(chosen *int, chosenAt int) SideBoard {
	choices := make([]PortChoice, 3)
	for i := range choices {
		choices[i] = PortChoice{Colour: "#000000", Run: func(*player.Player, *world.Tx) { *chosen = i }}
	}
	var b SideBoard
	for i := range b.Sides {
		b.Sides[i].Name = cube.Face(i).String()
		b.Sides[i].Ports = []SidePort{{Label: "port", Chosen: chosenAt, Choices: choices}}
	}
	return b
}

func TestSidesCardOrder(t *testing.T) {
	var ls []*oreLayout
	var lays []any
	oreShown = func(l *oreLayout, lay any) { ls = append(ls, l); lays = append(lays, lay) }
	defer func() { oreShown = nil }()
	var chosen int
	b := sidesBoard(&chosen, 0)
	b.Guide = "guide"
	SidesMenu(nil, "sides", b)
	d := lastSides(t, ls[0], lays[0])
	var got []string
	for _, c := range d.Cards {
		got = append(got, c.Name)
	}
	want := []string{"west", "up", "east", "north", "down", "south"}
	if strings.Join(got, " ") != strings.Join(want, " ") || d.Guide != "guide" {
		t.Errorf("cards %v guide %q", got, d.Guide)
	}
}

func lastSides(t *testing.T, l *oreLayout, lay any) sidesData {
	t.Helper()
	bs, _ := json.Marshal(lay.(map[string]any)["data"])
	var d sidesData
	if err := json.Unmarshal(bs, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// TestSidesCycle (sides_cycle): a swatch press runs the next choice, Refresh,
// and shows the sides again; its alt runs the previous one.
func TestSidesCycle(t *testing.T) {
	var chosen, refreshed int
	var ls []*oreLayout
	var lays []any
	oreShown = func(l *oreLayout, lay any) { ls = append(ls, l); lays = append(lays, lay) }
	defer func() { oreShown = nil }()
	b := sidesBoard(&chosen, 0)
	b.Refresh = func(*player.Player, *world.Tx) SideBoard { refreshed++; return b }
	SidesMenu(nil, "sides", b)
	if name, _ := screenRoot(lays[0]); name != "sides" {
		t.Fatalf("screen %q", name)
	}
	d := lastSides(t, ls[0], lays[0])
	sw := d.Cards[0].Ports[0].Swatch
	ls[0].runs[sw.ID](nil, nil)
	if chosen != 1 || refreshed != 1 || len(ls) != 2 {
		t.Errorf("press: chosen %d refreshed %d shown %d", chosen, refreshed, len(ls))
	}
	ls[0].runs[sw.Alt](nil, nil)
	if chosen != 2 || len(ls) != 3 {
		t.Errorf("alt: chosen %d shown %d", chosen, len(ls))
	}
	if b := mustJSON(lays[2]); strings.Contains(string(b), "null") && !strings.Contains(string(b), `"picker":null`) {
		t.Errorf("null in %s", b)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

// TestSidesPicker (picker): a plus opens the picker of its port and a choice
// in it closes it.
func TestSidesPicker(t *testing.T) {
	var chosen int
	var ls []*oreLayout
	var lays []any
	oreShown = func(l *oreLayout, lay any) { ls = append(ls, l); lays = append(lays, lay) }
	defer func() { oreShown = nil }()
	SidesMenu(nil, "sides", sidesBoard(&chosen, 1))
	d := lastSides(t, ls[0], lays[0])
	if d.Picker != nil {
		t.Fatal("picker open at first")
	}
	down := 4
	ls[0].runs[d.Cards[down].Ports[0].Plus.ID](nil, nil)
	d = lastSides(t, ls[1], lays[1])
	if d.Picker == nil || !d.Cards[down].Ports[0].Plus.On {
		t.Fatalf("picker %+v", d.Picker)
	}
	if len(d.Picker.Choices) != 3 || !d.Picker.Choices[1].On {
		t.Errorf("picker %+v", d.Picker)
	}
	ls[1].runs[d.Picker.Choices[2].ID](nil, nil)
	if chosen != 2 || len(ls) != 3 || lastSides(t, ls[2], lays[2]).Picker != nil {
		t.Errorf("choice: chosen %d shown %d", chosen, len(ls))
	}
}
