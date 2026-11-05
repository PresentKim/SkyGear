package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestNetBoxesSentMirroredLikeTheGeometry(t *testing.T) {
	shape := netShapeValue(KindNetImporter, cube.FaceEast, 0)
	world := netModel{netBoxes(KindNetImporter, shape)}.BBox(cube.Pos{}, nil)
	sent := netProperties(KindNetImporter, shape).SelectionBox
	var east float64
	for _, b := range world {
		east = max(east, b.Max().X())
	}
	if east != 1 {
		t.Fatalf("an east head ends at x %v in the world, want 1", east)
	}
	if sent.Min().X() != 0 {
		t.Fatalf("the client's selection box starts at x %v, want 0 (mirrored)", sent.Min().X())
	}
}
