package netshape

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func edge(b Box, f cube.Face) float64 {
	return map[cube.Face]float64{
		cube.FaceDown: b.From[1], cube.FaceUp: b.To[1], cube.FaceNorth: b.From[2],
		cube.FaceSouth: b.To[2], cube.FaceWest: b.From[0], cube.FaceEast: b.To[0],
	}[f]
}

func onEdge(f cube.Face) float64 {
	if f == cube.FaceUp || f == cube.FaceSouth || f == cube.FaceEast {
		return 16 - edgeInset
	}
	return edgeInset
}

func TestArmsReachJustInsideTheirSidesWithEveryFace(t *testing.T) {
	for _, f := range cube.Faces() {
		boxes := Of(false, 0, 0, 1<<f)
		if len(boxes[0].Faces) != 6 {
			t.Errorf("%v: the core draws %d faces", f, len(boxes[0].Faces))
		}
		arm := boxes[1]
		if got := edge(arm, f); got != onEdge(f) {
			t.Errorf("%v arm ends at %v, want %v", f, got, onEdge(f))
		}
		if len(arm.Faces) != 6 {
			t.Errorf("%v arm draws %d faces", f, len(arm.Faces))
		}
	}
}

func TestHeadPlateLiesJustInsideItsFacing(t *testing.T) {
	for _, f := range cube.Faces() {
		boxes := Of(true, Importer, f, 63)
		if len(boxes) != 1+5+3 {
			t.Fatalf("%v: %d boxes", f, len(boxes))
		}
		plate := boxes[len(boxes)-1]
		if got := edge(plate, f); got != onEdge(f) {
			t.Errorf("%v plate at %v, want %v", f, got, onEdge(f))
		}
		if len(plate.Faces) != 6 {
			t.Errorf("%v plate draws %d faces", f, len(plate.Faces))
		}
	}
}

// ofBeforeParts is Of as it was before the parts (one geometry per shape),
// kept to pin that the parts draw every shape the same.
func ofBeforeParts(head bool, h Head, f cube.Face, mask int) []Box {
	if head {
		mask &^= 1 << f
	}
	out := []Box{core()}
	for _, side := range cube.Faces() {
		if mask&(1<<side) != 0 {
			out = append(out, inset(turnBox(side, northArm())))
		}
	}
	if head {
		for _, b := range northHead(h) {
			out = append(out, inset(turnBox(f, b)))
		}
	}
	return out
}

// TestOfMatchesTheShapesBeforeParts: Of, now the visible parts, has the
// boxes of every shape it had before, for the cable and every head.
func TestOfMatchesTheShapesBeforeParts(t *testing.T) {
	for _, c := range []struct {
		head bool
		h    Head
	}{{false, 0}, {true, Importer}, {true, Exporter}, {true, External}} {
		for _, f := range cube.Faces() {
			for mask := range 64 {
				if got, want := Of(c.head, c.h, f, mask), ofBeforeParts(c.head, c.h, f, mask); !reflect.DeepEqual(got, want) {
					t.Fatalf("head %v %d facing %v mask %d: %d boxes, before %d", c.head, c.h, f, mask, len(got), len(want))
				}
			}
		}
	}
}
