package skygear

import (
	"math"
	"testing"
)

func TestHolderPoseRoundTrip(t *testing.T) {
	p := newHolderPose(-90, 45, 180, 65, -16, 3, 15)
	v, m := float64(float32(p.variant)), float64(float32(p.mark))
	got := []float64{
		math.Mod(v, 73)*5 - 180,
		math.Mod(math.Floor(v/73), 73)*5 - 180,
		math.Floor(v/5329)*5 - 180,
		math.Mod(m, 256),
		math.Mod(math.Floor(m/256), 32) - 16,
		math.Mod(math.Floor(m/8192), 32) - 16,
		math.Floor(m/262144) - 16,
	}
	want := []float64{-90, 45, 180, 65, -16, 3, 15}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("decoded %v, want %v", got, want)
		}
	}
}
