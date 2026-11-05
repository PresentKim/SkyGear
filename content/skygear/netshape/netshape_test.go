package netshape

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestArmsReachTheirSides(t *testing.T) {
	for _, f := range cube.Faces() {
		b := Of(false, 0, 0, 1<<f)[1]
		edge := map[cube.Face]float64{
			cube.FaceDown: b.From[1], cube.FaceUp: b.To[1], cube.FaceNorth: b.From[2],
			cube.FaceSouth: b.To[2], cube.FaceWest: b.From[0], cube.FaceEast: b.To[0],
		}[f]
		want := 0.0
		if f == cube.FaceUp || f == cube.FaceSouth || f == cube.FaceEast {
			want = 16
		}
		if edge != want {
			t.Errorf("%v arm ends at %v, want %v", f, edge, want)
		}
		if _, ok := b.Faces[f]; ok {
			t.Errorf("%v arm draws its end", f)
		}
		if _, ok := b.Faces[f.Opposite()]; ok {
			t.Errorf("%v arm draws its inner face", f)
		}
		if len(b.Faces) != 4 {
			t.Errorf("%v arm has %d faces", f, len(b.Faces))
		}
	}
}

func TestHeadPlateLiesOnItsFacing(t *testing.T) {
	for _, f := range cube.Faces() {
		boxes := Of(true, Importer, f, 63)
		if len(boxes) != 1+5+3 {
			t.Fatalf("%v: %d boxes", f, len(boxes))
		}
		plate := boxes[len(boxes)-1]
		bb := Bounds([]Box{plate})
		at := map[cube.Face]float64{
			cube.FaceDown: bb.Min()[1], cube.FaceUp: bb.Max()[1], cube.FaceNorth: bb.Min()[2],
			cube.FaceSouth: bb.Max()[2], cube.FaceWest: bb.Min()[0], cube.FaceEast: bb.Max()[0],
		}[f]
		if at != 0 && at != 1 {
			t.Errorf("%v plate at %v", f, at)
		}
		if _, ok := plate.Faces[f]; ok {
			t.Errorf("%v plate draws the face against the block", f)
		}
	}
}
