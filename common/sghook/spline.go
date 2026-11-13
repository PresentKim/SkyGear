package sghook

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// CameraKey is a point a camera spline passes: at T seconds from the start
// the camera is at Pos looking at the rotation (pitch, yaw in degrees).
type CameraKey struct {
	T          float64
	Pos        mgl64.Vec3
	Pitch, Yaw float64
	// Ease is how the camera moves from the key before to this one (a
	// protocol.EasingType; 0 is linear).
	Ease int32
}

// PlayCameraSpline moves p's camera along a smooth (Catmull-Rom) path
// through keys, played by the client on its own: keys[0] is where it starts
// (T 0) and the last key's T is the length. The camera stays at the last
// key; clear or set it after. Fewer than two keys do nothing.
func PlayCameraSpline(p *player.Player, keys []CameraKey) {
	if len(keys) < 2 {
		return
	}
	first, last := keys[0], keys[len(keys)-1]
	SetCamera(p, first.Pos, first.Pitch, first.Yaw, 0)
	write(p, &packet.CameraInstruction{Spline: protocol.Option(cameraSpline(keys, last.T))})
}

func cameraSpline(keys []CameraKey, total float64) protocol.CameraSplineInstruction {
	in := protocol.CameraSplineInstruction{TotalTime: float32(total)}
	n := float32(len(keys) - 1)
	for i, k := range keys {
		in.Curve = append(in.Curve, vec32(k.Pos))
		in.ProgressKeyFrames = append(in.ProgressKeyFrames, protocol.CameraProgressOption{
			Value: float32(i) / n, Time: float32(k.T), EaseType: k.Ease,
		})
		in.RotationOptions = append(in.RotationOptions, protocol.CameraRotationOption{
			Value: mgl32.Vec3{float32(k.Pitch), float32(k.Yaw), 0}, Time: float32(k.T), EaseType: k.Ease,
		})
	}
	return in
}
