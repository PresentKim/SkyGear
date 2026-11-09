package machine

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/content/skygear/netshape"
)

func TestNetBoxesSentMirroredLikeTheGeometry(t *testing.T) {
	shape := netShapeValue(KindNetImporter, cube.FaceEast, 0)
	world := netModel{netBoxes(KindNetImporter, shape)}.BBox(cube.Pos{}, nil)
	sent := netProperties(KindNetImporter, shape).SelectionBox
	var east float64
	for _, b := range world {
		east = max(east, b.Max().X())
	}
	if east < 0.99 {
		t.Fatalf("an east head ends at x %v in the world, want the east edge", east)
	}
	if sent.Min().X() > 0.01 {
		t.Fatalf("the client's selection box starts at x %v, want the west edge (mirrored)", sent.Min().X())
	}
}

// TestNetPartsDrawEveryShape: the bones a shape shows hold exactly the boxes
// of the shape (netshape.Of, the server's model and collision), so one
// geometry per kind draws what the per-shape geometries drew.
func TestNetPartsDrawEveryShape(t *testing.T) {
	for kind, shapes := range NetShapes() {
		parts := map[string][]netshape.Box{}
		for _, p := range NetParts(kind) {
			parts[p.Name] = p.Boxes
		}
		for _, s := range shapes {
			var drawn []netshape.Box
			for _, name := range netVisible(kind, s) {
				drawn = append(drawn, parts[name]...)
			}
			if want := netBoxes(kind, s); !reflect.DeepEqual(drawn, want) {
				t.Fatalf("%s shape %d: bones draw %d boxes, the shape has %d", kind, s, len(drawn), len(want))
			}
			props := netProperties(kind, s)
			if len(props.BoneVisibility) != len(parts) {
				t.Fatalf("%s shape %d: bone_visibility names %d bones, the geometry has %d", kind, s, len(props.BoneVisibility), len(parts))
			}
		}
	}
}
