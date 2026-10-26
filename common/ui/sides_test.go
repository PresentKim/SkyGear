package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func abs(n int) int { return max(n, -n) }

func TestSideCardsStayApartInTheView(t *testing.T) {
	seen := map[cube.Face][2]int{}
	for _, f := range cube.Faces() {
		x, y, _ := sideCard(f)
		if x < 0 || y < 0 || x+cardW > sidesW || y+cardH > sidesH {
			t.Errorf("%v at %d,%d is outside the view", f, x, y)
		}
		for other, o := range seen {
			if abs(x-o[0]) < cardW && abs(y-o[1]) < cardH {
				t.Errorf("%v at %d,%d overlaps %v at %v", f, x, y, other, o)
			}
		}
		seen[f] = [2]int{x, y}
	}
	if sidesW > windowW-4 || sidesH > windowH-18 {
		t.Errorf("view %dx%d does not fit a %dx%d window", sidesW, sidesH, windowW, windowH)
	}
	cx, cy := sidesW/2-centreSize/2, sidesH/2-centreSize/2
	for f, o := range seen {
		if cx < o[0]+cardW && o[0] < cx+centreSize && cy < o[1]+cardH && o[1] < cy+centreSize {
			t.Errorf("%v covers the block in the middle", f)
		}
	}
}

// TestSidePortRightPress: a port's square presses to the next choice and,
// with the right mouse button, to the one before, wrapping at the ends.
func TestSidePortRightPress(t *testing.T) {
	var chosen int
	choices := make([]PortChoice, 3)
	for i := range choices {
		choices[i] = PortChoice{Colour: "#000000", Run: func(*player.Player, *world.Tx) { chosen = i }}
	}
	var b SideBoard
	for i := range b.Sides {
		b.Sides[i].Ports = []SidePort{{Label: "port", Chosen: 0, Choices: choices}}
	}
	l := newOreLayout()
	vars := sidesVars(nil, l, b, nil)
	swatch := vars["@side0"].(map[string]any)["children"].([]any)[0].(map[string]any)["children"].([]any)[0].(map[string]any)
	l.runs[swatch["id"].(string)](nil, nil)
	if chosen != 1 {
		t.Errorf("press chose %d, want 1", chosen)
	}
	l.runs[swatch["alt"].(string)](nil, nil)
	if chosen != 2 {
		t.Errorf("right press chose %d, want 2", chosen)
	}
}
