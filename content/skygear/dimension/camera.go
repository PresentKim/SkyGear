package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The hunt camera: while hunting, the hunter looks down on itself from
// above and behind, as in a survivor game, and walks the way the camera
// faces. It is an orbit camera that follows the player on the client, so
// nothing is sent while it moves; the player can still turn it around.

// huntPresets are the camera presets sent to a hunter: the built-in ones
// the hunt camera builds on, then the hunt camera itself.
var huntPresets = []protocol.CameraPreset{
	{Name: "minecraft:free"},
	{Name: "minecraft:first_person"},
	{Name: "minecraft:third_person"},
	{Name: "minecraft:third_person_front"},
	{Name: "minecraft:follow_orbit"},
	{
		Name: "skygear:hunt", Parent: "minecraft:follow_orbit",
		RotX:                  protocol.Option[float32](huntPitch),
		Radius:                protocol.Option[float32](huntDistance),
		EntityOffset:          protocol.Option(mgl32.Vec3{0, 1, 0}),
		VerticalRotationLimit: protocol.Option(mgl32.Vec2{huntPitch - 10, huntPitch + 10}),
		ControlScheme:         protocol.Option[byte](packet.ControlSchemeCameraRelative),
		AudioListener:         protocol.Option[byte](protocol.AudioListenerPlayer),
	},
}

const (
	// huntPreset is the hunt camera's place in huntPresets.
	huntPreset = 5
	// huntPitch is how steeply (degrees) the camera looks down, and
	// huntDistance how far (blocks) it is from the hunter.
	huntPitch    = 60
	huntDistance = 12
)

// huntCamera puts p's view on the hunt camera.
func huntCamera(p *player.Player) {
	c := sghook.Of(p)
	if c == nil {
		return
	}
	_ = c.WritePacket(&packet.CameraPresets{Presets: huntPresets})
	_ = c.WritePacket(&packet.CameraInstruction{Set: protocol.Option(protocol.CameraInstructionSet{Preset: huntPreset})})
}

// plainCamera gives p its own view back.
func plainCamera(p *player.Player) {
	if c := sghook.Of(p); c != nil {
		_ = c.WritePacket(&packet.CameraInstruction{Clear: protocol.Option(true)})
	}
}
