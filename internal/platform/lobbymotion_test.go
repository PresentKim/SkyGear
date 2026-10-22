package platform

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl64"
)

// TestLobbyCameraLooksAtTheIsland: the first stop is the lobby's still
// camera, and every stop looks at the island's middle from 12 blocks.
func TestLobbyCameraLooksAtTheIsland(t *testing.T) {
	pos, pitch, yaw := lobbyCameraAt(0)
	if !pos.ApproxEqualThreshold(lobbyCamera, 1e-9) || math.Abs(pitch-lobbyPitch) > 0.5 || yaw != lobbyYaw {
		t.Errorf("stop 0 is %v %.1f %.1f, want %v %.1f %.1f", pos, pitch, yaw, lobbyCamera, lobbyPitch, lobbyYaw)
	}
	pivot := LobbySpawn.Vec3Middle().Add(mgl64.Vec3{0, 0.5, 0})
	for i := range lobbyOrbit {
		pos, pitch, yaw := lobbyCameraAt(i)
		// the direction of a camera rotation, as the game reads it
		y, p := mgl64.DegToRad(yaw), mgl64.DegToRad(pitch)
		dir := mgl64.Vec3{-math.Sin(y) * math.Cos(p), -math.Sin(p), math.Cos(y) * math.Cos(p)}
		to := pivot.Sub(pos)
		if got := dir.Dot(to.Normalize()); got < 0.999 {
			t.Errorf("stop %d looks %.3f off the island", i, got)
		}
		if d := math.Hypot(to.X(), to.Z()); math.Abs(d-12) > 1e-9 {
			t.Errorf("stop %d is %.2f away", i, d)
		}
	}
}
