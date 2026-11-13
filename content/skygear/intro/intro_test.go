package intro

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl64"
)

func TestKeysPullBackBehindThePlayer(t *testing.T) {
	// Facing south (yaw 0, +z): the overview is north (-z) and above.
	keys := Keys(mgl64.Vec3{0, 64, 0}, 5, 0)
	if len(keys) != 4 {
		t.Fatalf("%d keys", len(keys))
	}
	eye, far := keys[0].Pos, keys[1].Pos
	if eye != (mgl64.Vec3{0, 65.62, 0}) || keys[3].Pos != eye {
		t.Fatalf("eye %v, end %v", eye, keys[3].Pos)
	}
	if math.Abs(far[0]) > 1e-9 || math.Abs(far[2]+distance) > 1e-9 || math.Abs(far[1]-(eye[1]+height)) > 1e-9 {
		t.Fatalf("overview %v", far)
	}
	if keys[1].Pitch <= 0 || keys[2].Pos != far || keys[3].T != out+hold+back {
		t.Fatalf("keys %+v", keys)
	}
}
