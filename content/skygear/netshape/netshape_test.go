package netshape

import (
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
