package session

// This file is part of the dragonfactory patch to Dragonfly (third_party/README.md):
// custom camera control and client-only block views.

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// cameraPresetFree is the index of the free camera preset sent on spawn.
const cameraPresetFree = 0

// sendCameraPresets sends the camera presets custom cameras use. Dragonfly
// sends none, and camera instructions refer to presets by index. Movement in
// the free camera is relative to the camera, so with a top-down camera the
// movement keys move across the screen.
func (s *Session) sendCameraPresets() {
	s.writePacket(&packet.CameraPresets{Presets: []protocol.CameraPreset{{
		Name:          "minecraft:free",
		ControlScheme: protocol.Option(byte(packet.ControlSchemeCameraRelativeStrafe)),
	}}})
}

// SetCamera puts the player's camera at pos looking at the rotation (pitch,
// yaw in degrees), moving there over ease seconds.
func (s *Session) SetCamera(pos mgl64.Vec3, pitch, yaw float64, ease float64) {
	if s == Nop {
		return
	}
	set := protocol.CameraInstructionSet{
		Preset:   cameraPresetFree,
		Position: protocol.Option(vec64To32(pos)),
		Rotation: protocol.Option(mgl32.Vec2{float32(pitch), float32(yaw)}),
	}
	if ease > 0 {
		set.Ease = protocol.Option(protocol.CameraEase{Type: protocol.EasingTypeLinear, Duration: float32(ease)})
	}
	s.writePacket(&packet.CameraInstruction{Set: protocol.Option(set)})
}

// ClearCamera returns the player's camera to normal.
func (s *Session) ClearCamera() {
	if s == Nop {
		return
	}
	s.writePacket(&packet.CameraInstruction{Clear: protocol.Option(true)})
}

// ViewFakeBlock shows b at pos to this player only. The world is unchanged;
// resend the real block to undo it.
func (s *Session) ViewFakeBlock(pos cube.Pos, b world.Block) {
	if s == Nop {
		return
	}
	s.ViewBlockUpdate(pos, b, 0)
}
