package intro

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl64"
)

func TestKeysStartHighAndEndAtTheEyes(t *testing.T) {
	// Facing south (yaw 0, +z): the start is north (-z) of the eyes and above.
	keys := Keys(mgl64.Vec3{0, 64, 0}, 5, 0)
	if len(keys) != 2 {
		t.Fatalf("%d keys", len(keys))
	}
	high, eye := keys[0], keys[1]
	if eye.Pos != (mgl64.Vec3{0, 65.62, 0}) || eye.Pitch != 5 || eye.T != glide {
		t.Fatalf("end %+v", eye)
	}
	if math.Abs(high.Pos[0]) > 1e-9 || math.Abs(high.Pos[2]+behind) > 1e-9 || math.Abs(high.Pos[1]-(eye.Pos[1]+rise)) > 1e-9 {
		t.Fatalf("start %v", high.Pos)
	}
	if high.T != 0 || high.Pitch != startPitch || high.Yaw != eye.Yaw {
		t.Fatalf("start %+v", high)
	}
}
