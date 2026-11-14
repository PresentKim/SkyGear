// Package intro is the short camera move a player sees on first entering a
// save: the camera pulls back from the player's eyes to show the island and
// comes back, then the player takes over.
package intro

import (
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	// delay lets the island's chunks reach the client first.
	delay = 1500 * time.Millisecond
	// out is how long the camera takes back to the overview, hold how long
	// it stays there, back how long it takes back to the eyes.
	out, hold, back = 1.3, 0.6, 1.3
	// distance and height are where the overview is: behind the player and
	// above its eyes.
	distance, height = 10.0, 5.0
	// locks are the inputs locked during the move.
	locks = packet.ClientInputLockCamera | packet.ClientInputLockMovement | packet.ClientInputLockLateralMovement |
		packet.ClientInputLockMoveForward | packet.ClientInputLockMoveBackward |
		packet.ClientInputLockMoveLeft | packet.ClientInputLockMoveRight | packet.ClientInputLockJump | packet.ClientInputLockSneak
)

// Play starts the move for p after delay, in the world p is in now.
func Play(p *player.Player) {
	w := p.Tx().World()
	p.H().DoAfter(delay, func(tx *world.Tx, e world.Entity) {
		if tx.World() == w {
			start(e.(*player.Player), w)
		}
	})
}

func start(p *player.Player, w *world.World) {
	keys := Keys(p.Position(), p.Rotation().Pitch(), p.Rotation().Yaw())
	sghook.HideHUD(p, true)
	sghook.LockInputs(p, locks)
	sghook.PlayCameraSpline(p, keys)
	p.H().DoAfter(seconds(out+hold+back), func(tx *world.Tx, e world.Entity) {
		if tx.World() != w {
			return // left the save: the new place sets its own camera
		}
		p := e.(*player.Player)
		sghook.ClearCamera(p)
		sghook.LockInputs(p, 0)
		sghook.HideHUD(p, false)
	})
}

// Keys are the camera's points for a player standing at pos looking at the
// rotation: its eyes, the overview (twice, for the hold) and its eyes again.
func Keys(pos mgl64.Vec3, pitch, yaw float64) []sghook.CameraKey {
	eye := pos.Add(mgl64.Vec3{0, 1.62, 0})
	a := mgl64.DegToRad(yaw)
	forward := mgl64.Vec3{-math.Sin(a), 0, math.Cos(a)}
	far := eye.Sub(forward.Mul(distance)).Add(mgl64.Vec3{0, height, 0})
	farPitch := mgl64.RadToDeg(math.Atan2(height, distance))
	ease := int32(protocol.EasingTypeInOutSine)
	return []sghook.CameraKey{
		{T: 0, Pos: eye, Pitch: pitch, Yaw: yaw},
		{T: out, Pos: far, Pitch: farPitch, Yaw: yaw, Ease: ease},
		{T: out + hold, Pos: far, Pitch: farPitch, Yaw: yaw},
		{T: out + hold + back, Pos: eye, Pitch: pitch, Yaw: yaw, Ease: ease},
	}
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }
