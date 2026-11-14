package sghook

import (
	"bytes"
	"testing"

	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCameraSplineKeys(t *testing.T) {
	in := cameraSpline([]CameraKey{
		{T: 0, Pos: mgl64.Vec3{0, 70, 0}, Pitch: 10, Yaw: 180},
		{T: 2, Pos: mgl64.Vec3{0, 75, 10}, Pitch: 20, Yaw: 180, Ease: protocol.EasingTypeInOutSine},
		{T: 3, Pos: mgl64.Vec3{0, 70, 0}, Pitch: 10, Yaw: 180},
	}, 3, SplineOptions{})
	if in.TotalTime != 3 || len(in.Curve) != 3 || in.ProgressKeyFrames[1].Value != 0.5 || in.ProgressKeyFrames[2].Value != 1 {
		t.Fatalf("spline %+v", in)
	}
	if r := in.RotationOptions[1]; r.Value[0] != 20 || r.Value[1] != 180 || r.Time != 2 || r.EaseType != protocol.EasingTypeInOutSine {
		t.Fatalf("rotation %+v", r)
	}
	// The packet encodes and decodes the same.
	pk := &packet.CameraInstruction{Spline: protocol.Option(in)}
	buf := bytes.NewBuffer(nil)
	pk.Marshal(protocol.NewWriter(buf, 0))
	var got packet.CameraInstruction
	got.Marshal(protocol.NewReader(buf, 0, false))
	s, ok := got.Spline.Value()
	if !ok || len(s.Curve) != 3 || s.RotationOptions[1].EaseType != protocol.EasingTypeInOutSine {
		t.Fatalf("decoded %+v", got)
	}
}

func TestCameraSplineOptions(t *testing.T) {
	keys := []CameraKey{
		{T: 0, Pos: mgl64.Vec3{0, 70, 0}, Pitch: 10, Yaw: 180},
		{T: 3, Pos: mgl64.Vec3{0, 75, 10}, Pitch: 20, Yaw: 90},
	}
	in := cameraSpline(keys, 3, SplineOptions{YawFirst: true, FlipPitch: true})
	if r := in.RotationOptions[1].Value; r[0] != 90 || r[1] != -20 {
		t.Fatalf("rotation %v, want (90, -20)", r)
	}
}

func TestSubdivideGivesDistinctPointsAndDistanceProgress(t *testing.T) {
	a := CameraKey{T: 0, Pos: mgl64.Vec3{0, 80, 0}, Pitch: 60, Yaw: 0}
	b := CameraKey{T: 1.5, Pos: mgl64.Vec3{0, 64, 4}, Pitch: 0, Yaw: 0}
	keys := Subdivide(a, b, 5)
	if len(keys) != 5 || keys[0].Pos != a.Pos || keys[4].Pos != b.Pos || keys[4].T != 1.5 || keys[2].Pitch != 30 {
		t.Fatalf("keys %+v", keys)
	}
	for i := 1; i < len(keys); i++ {
		if keys[i].Pos == keys[i-1].Pos {
			t.Fatalf("points %d and %d are the same", i-1, i)
		}
	}
	in := cameraSpline(keys, 1.5, SplineOptions{})
	for i, want := range []float32{0, 0.25, 0.5, 0.75, 1} {
		if got := in.ProgressKeyFrames[i].Value; got < want-1e-5 || got > want+1e-5 {
			t.Fatalf("progress %d = %v, want %v", i, got, want)
		}
	}
}
