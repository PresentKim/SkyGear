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

// MoveInput returns the movement keys the client last held, as sent in
// PlayerAuthInput before input permissions apply (so also while immobile):
// Y is forward (+1) and back, X is left (+1) and right.
func (s *Session) MoveInput() mgl64.Vec2 {
	if v := s.moveInput.Load(); v != nil {
		return *v
	}
	return mgl64.Vec2{}
}

// ViewFakeEntity shows an entity of the type identifier (e.g.
// "minecraft:allay") at pos to this player only, standing still and scaled,
// and returns its runtime ID for MoveFakeEntity and RemoveFakeEntity.
func (s *Session) ViewFakeEntity(identifier string, pos mgl64.Vec3, yaw float64, scale float32) uint64 {
	if s == Nop {
		return 0
	}
	s.entityMutex.Lock()
	s.currentEntityRuntimeID++
	id := s.currentEntityRuntimeID
	s.entityMutex.Unlock()

	m := protocol.NewEntityMetadata()
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagNoAI)
	m[protocol.EntityDataKeyScale] = scale
	s.writePacket(&packet.AddActor{
		EntityUniqueID:  int64(id),
		EntityRuntimeID: id,
		EntityType:      identifier,
		EntityMetadata:  m,
		Position:        vec64To32(pos),
		Yaw:             float32(yaw),
		HeadYaw:         float32(yaw),
		BodyYaw:         float32(yaw),
	})
	return id
}

// MoveFakeEntity moves an entity shown with ViewFakeEntity.
func (s *Session) MoveFakeEntity(id uint64, pos mgl64.Vec3, yaw float64) {
	if s == Nop || id == 0 {
		return
	}
	s.writePacket(&packet.MoveActorAbsolute{EntityRuntimeID: id, Position: vec64To32(pos), Rotation: mgl32.Vec3{0, float32(yaw), float32(yaw)}})
}

// RemoveFakeEntity removes an entity shown with ViewFakeEntity.
func (s *Session) RemoveFakeEntity(id uint64) {
	if s == Nop || id == 0 {
		return
	}
	s.writePacket(&packet.RemoveActor{EntityUniqueID: int64(id)})
}
