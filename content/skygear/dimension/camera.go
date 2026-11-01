package dimension

import (
	"math"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The hunt camera: while hunting, the hunter is seen from high above at a
// fixed angle, as in a survivor game, and walks relative to the screen (up
// is the camera's forward). The view cannot be turned: it is a free camera
// the server moves after the hunter, easing so it glides.

// huntPresets are the camera presets sent to a hunter: the built-in ones,
// then the hunt camera on the free one, with movement relative to it.
var huntPresets = []protocol.CameraPreset{
	{Name: "minecraft:free"},
	{Name: "minecraft:first_person"},
	{Name: "minecraft:third_person"},
	{Name: "minecraft:third_person_front"},
	{
		Name: "skygear:hunt", Parent: "minecraft:free",
		ControlScheme: protocol.Option[byte](packet.ControlSchemeCameraRelative),
		AudioListener: protocol.Option[byte](protocol.AudioListenerPlayer),
	},
}

const (
	// huntPreset is the hunt camera's place in huntPresets.
	huntPreset = 4
	// huntPitch is how steeply (degrees) the camera looks down, huntYaw the
	// way it faces (south), huntDistance how far (blocks) it is from the
	// hunter's chest.
	huntPitch    = 72
	huntYaw      = 0
	huntDistance = 16
	// huntEase is how long (seconds) the camera glides to each new spot.
	huntEase = 0.1
)

// huntCamera sends p the presets and puts its view on the hunt camera.
func huntCamera(p *player.Player) {
	c := sghook.Of(p)
	if c == nil {
		return
	}
	_ = c.WritePacket(&packet.CameraPresets{Presets: huntPresets})
	followCamera(p, false)
	// the keys move the hunter the way the screen shows, and it turns to face
	// where it walks (the preset's scheme alone is not applied)
	_ = c.WritePacket(&packet.ClientBoundControlSchemeSet{ControlScheme: packet.ControlSchemeCameraRelative})
}

// followCamera moves p's hunt camera over p, gliding unless at once.
func followCamera(p *player.Player, glide bool) {
	c := sghook.Of(p)
	if c == nil {
		return
	}
	pitch, yaw := huntPitch*math.Pi/180, huntYaw*math.Pi/180
	// the camera looks along (−sin yaw·cos pitch, −sin pitch, cos yaw·cos pitch):
	// it stands that far back from the hunter's chest
	look := mgl64.Vec3{-math.Sin(yaw) * math.Cos(pitch), -math.Sin(pitch), math.Cos(yaw) * math.Cos(pitch)}
	at := p.Position().Add(mgl64.Vec3{0, 1, 0}).Sub(look.Mul(huntDistance))
	set := protocol.CameraInstructionSet{
		Preset:   huntPreset,
		Position: protocol.Option(mgl32.Vec3{float32(at[0]), float32(at[1]), float32(at[2])}),
		Rotation: protocol.Option(mgl32.Vec2{huntPitch, huntYaw}),
	}
	if glide {
		set.Ease = protocol.Option(protocol.CameraEase{Type: protocol.EasingTypeLinear, Duration: huntEase})
	}
	_ = c.WritePacket(&packet.CameraInstruction{Set: protocol.Option(set)})
}

// plainCamera gives p its own view back.
func plainCamera(p *player.Player) {
	if c := sghook.Of(p); c != nil {
		_ = c.WritePacket(&packet.CameraInstruction{Clear: protocol.Option(true)})
		_ = c.WritePacket(&packet.ClientBoundControlSchemeSet{ControlScheme: packet.ControlSchemeLockedPlayerRelativeStrafe})
	}
}
