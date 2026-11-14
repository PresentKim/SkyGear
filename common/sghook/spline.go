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

// SplineOptions are the variants /sgdebug spline tries; the zero value is
// the plain spline.
type SplineOptions struct {
	// YawFirst writes each rotation Value as (yaw, pitch) instead of
	// (pitch, yaw).
	YawFirst bool
	// FlipPitch sends each pitch negated.
	FlipPitch bool
	// NoSet skips the free camera Set at the first key before the spline.
	NoSet bool
	// Origin, when set, is subtracted from every control point (points
	// relative to it instead of world positions).
	Origin *mgl64.Vec3
	// ProgressScale multiplies each progress key frame's Time (0 is 1):
	// 1/total sends times as fractions of the length, 20 as ticks.
	ProgressScale float64
	// NoProgress leaves out the progress key frames.
	NoProgress bool
	// Linear asks for a linear spline (type 1) instead of Catmull-Rom (0).
	Linear bool
}

// PlayCameraSpline moves p's camera along a smooth (Catmull-Rom) path
// through keys, played by the client on its own: keys[0] is where it starts
// (T 0) and the last key's T is the length. The client needs at least four
// distinct points and breaks on a point repeated in a row (see Subdivide).
// The camera stays at the last key; clear or set it after. Fewer than two
// keys do nothing.
func PlayCameraSpline(p *player.Player, keys []CameraKey, opt SplineOptions) {
	if len(keys) < 2 {
		return
	}
	first, last := keys[0], keys[len(keys)-1]
	if !opt.NoSet {
		SetCamera(p, first.Pos, first.Pitch, first.Yaw, 0)
	}
	pk := &splinePacket{in: cameraSpline(keys, last.T, opt)}
	if opt.Linear {
		pk.splineType = 1
	}
	write(p, pk)
}

// Subdivide returns n keys evenly spaced in time between a and b, with the
// position, pitch and yaw linearly between theirs, so a two-key move has
// the distinct points a Catmull-Rom spline needs. Every key moves linearly
// (an ease would restart at each point).
func Subdivide(a, b CameraKey, n int) []CameraKey {
	keys := make([]CameraKey, n)
	for i := range keys {
		f := float64(i) / float64(n-1)
		keys[i] = CameraKey{
			T:     a.T + (b.T-a.T)*f,
			Pos:   a.Pos.Add(b.Pos.Sub(a.Pos).Mul(f)),
			Pitch: a.Pitch + (b.Pitch-a.Pitch)*f,
			Yaw:   a.Yaw + (b.Yaw-a.Yaw)*f,
		}
	}
	return keys
}

// cameraSpline builds the instruction: the progress of each key is the
// distance along the control points up to it over the whole length (0 to
// 1), and each key's rotation is a rotation key frame at its time.
func cameraSpline(keys []CameraKey, total float64, opt SplineOptions) protocol.CameraSplineInstruction {
	in := protocol.CameraSplineInstruction{TotalTime: float32(total)}
	dist := make([]float64, len(keys))
	for i := 1; i < len(keys); i++ {
		dist[i] = dist[i-1] + keys[i].Pos.Sub(keys[i-1].Pos).Len()
	}
	length := dist[len(dist)-1]
	for i, k := range keys {
		pos := k.Pos
		if opt.Origin != nil {
			pos = pos.Sub(*opt.Origin)
		}
		in.Curve = append(in.Curve, vec32(pos))
		progress := float32(i) / float32(len(keys)-1)
		if length > 0 {
			progress = float32(dist[i] / length)
		}
		scale := opt.ProgressScale
		if scale == 0 {
			scale = 1
		}
		if !opt.NoProgress {
			in.ProgressKeyFrames = append(in.ProgressKeyFrames, protocol.CameraProgressOption{
				Value: progress, Time: float32(k.T * scale), EaseType: k.Ease,
			})
		}
		pitch := float32(k.Pitch)
		if opt.FlipPitch {
			pitch = -pitch
		}
		rot := mgl32.Vec3{pitch, float32(k.Yaw), 0}
		if opt.YawFirst {
			rot = mgl32.Vec3{float32(k.Yaw), pitch, 0}
		}
		in.RotationOptions = append(in.RotationOptions, protocol.CameraRotationOption{
			Value: rot, Time: float32(k.T), EaseType: k.Ease,
		})
	}
	return in
}

// splinePacket is a CameraInstruction that holds only a spline, encoded the
// way the client reads it. gophertunnel v1.63.0 writes the spline type,
// identifier and JSON flag as optional fields (a presence byte first); the
// client reads them as plain fields, so any spline type but the default
// shifts the rest of the packet. Upstream fixed this after v1.63.0.
type splinePacket struct {
	in         protocol.CameraSplineInstruction
	splineType uint8
}

// ID ...
func (*splinePacket) ID() uint32 { return packet.IDCameraInstruction }

// Marshal writes the instruction with every field but the spline absent.
func (pk *splinePacket) Marshal(io protocol.IO) {
	absent, present := false, true
	for range 6 { // Set, Clear, Fade, Target, RemoveTarget, FieldOfView
		io.Bool(&absent)
	}
	io.Bool(&present)
	io.Float32(&pk.in.TotalTime)
	io.Uint8(&pk.splineType)
	protocol.FuncSlice(io, &pk.in.Curve, io.Vec3)
	protocol.Slice(io, &pk.in.ProgressKeyFrames)
	protocol.Slice(io, &pk.in.RotationOptions)
	id := ""
	io.String(&id)
	io.Bool(&absent) // LoadFromJson
	io.Bool(&absent) // AttachToEntity
	io.Bool(&absent) // DetachFromEntity
}
