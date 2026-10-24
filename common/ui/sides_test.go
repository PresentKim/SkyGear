package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
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
	if sidesW > sidesWindowW-4 || sidesH > sidesWindowH-18 {
		t.Errorf("view %dx%d does not fit a %dx%d window", sidesW, sidesH, sidesWindowW, sidesWindowH)
	}
	cx, cy := sidesW/2-centreSize/2, sidesH/2-centreSize/2
	for f, o := range seen {
		if cx < o[0]+cardW && o[0] < cx+centreSize && cy < o[1]+cardH && o[1] < cy+centreSize {
			t.Errorf("%v covers the block in the middle", f)
		}
	}
}
