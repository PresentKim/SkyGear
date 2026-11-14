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
	in := cameraSpline(keys, 3, SplineOptions{YawFirst: true, Pad: true})
	if len(in.Curve) != 4 || in.Curve[0] != in.Curve[1] || in.Curve[2] != in.Curve[3] || len(in.ProgressKeyFrames) != 2 {
		t.Fatalf("padded curve %+v", in.Curve)
	}
	if r := in.RotationOptions[1].Value; r[0] != 90 || r[1] != 20 {
		t.Fatalf("yaw-first rotation %v", r)
	}
}

func TestCameraSplineIndexProgress(t *testing.T) {
	keys := []CameraKey{{T: 0}, {T: 1.5}}
	in := cameraSpline(keys, 1.5, SplineOptions{Pad: true, Index: true})
	if len(in.Curve) != 4 {
		t.Fatalf("curve has %d points, want 4", len(in.Curve))
	}
	if a, b := in.ProgressKeyFrames[0].Value, in.ProgressKeyFrames[1].Value; a != 1 || b != 2 {
		t.Fatalf("progress values %v, %v, want 1, 2 (the keys' control points)", a, b)
	}
}
