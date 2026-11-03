package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The hunt camera: while hunting, the hunter is seen from high above at a
// fixed angle, as in a survivor game, and walks relative to the screen (up
// is the camera's forward), turning to face where it walks. It is the
// client's fixed boom camera: the client keeps it at an offset from the
// hunter every frame, so it follows smoothly with nothing sent while the
// hunt goes on.

// huntPresets are the camera presets sent to a hunter: the built-in ones,
// then the hunt camera on the fixed boom.
var huntPresets = []protocol.CameraPreset{
	{Name: "minecraft:free"},
	{Name: "minecraft:first_person"},
	{Name: "minecraft:third_person"},
	{Name: "minecraft:third_person_front"},
	{Name: "minecraft:fixed_boom"},
	{
		Name: "skygear:hunt", Parent: "minecraft:fixed_boom",
		RotX:                    protocol.Option[float32](huntPitch),
		RotY:                    protocol.Option[float32](huntYaw),
		RotationSpeed:           protocol.Option[float32](0.7),
		HorizontalRotationLimit: protocol.Option(mgl32.Vec2{90, 90}),
		EntityOffset:            protocol.Option(huntOffset),
		Radius:                  protocol.Option[float32](30),
		ControlScheme:           protocol.Option[byte](packet.ControlSchemeCameraRelative),
		AudioListener:           protocol.Option[byte](protocol.AudioListenerPlayer),
		PlayerEffects:           protocol.Option(true),
	},
}

const (
	// huntPreset is the hunt camera's place in huntPresets.
	huntPreset = 5
	// huntPitch is how steeply (degrees) the camera looks down, huntYaw the
	// way it faces (south).
	huntPitch = 72
	huntYaw   = 0
)

// huntOffset is where the camera stands from the hunter: high above and a
// little behind (north), so the hunter is in the middle of the screen.
var huntOffset = mgl32.Vec3{0, 15, -4}

// huntCamera sends p the presets and puts its view on the hunt camera.
func huntCamera(p *player.Player) {
	c := sghook.Of(p)
	if c == nil {
		return
	}
	_ = c.WritePacket(&packet.CameraPresets{Presets: huntPresets})
	_ = c.WritePacket(&packet.CameraInstruction{Set: protocol.Option(protocol.CameraInstructionSet{
		Preset:       huntPreset,
		Rotation:     protocol.Option(mgl32.Vec2{huntPitch, huntYaw}),
		EntityOffset: protocol.Option(huntOffset),
	})})
	// the keys move the hunter the way the screen shows, and it turns to face
	// where it walks (the preset's scheme alone is not applied)
	_ = c.WritePacket(&packet.ClientBoundControlSchemeSet{ControlScheme: packet.ControlSchemeCameraRelative})
}

// plainCamera gives p its own view back.
func plainCamera(p *player.Player) {
	if c := sghook.Of(p); c != nil {
		_ = c.WritePacket(&packet.CameraInstruction{Clear: protocol.Option(true)})
		_ = c.WritePacket(&packet.ClientBoundControlSchemeSet{ControlScheme: packet.ControlSchemeLockedPlayerRelativeStrafe})
	}
}
