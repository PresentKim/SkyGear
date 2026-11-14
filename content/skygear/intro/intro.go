// Package intro is the short camera move a player sees on first entering a
// save: the camera starts high above the island looking down and glides to
// the player's eyes, then the player takes over.
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
	// glide is how long the camera takes from above the island to the eyes.
	glide = 1.5
	// rise and behind are where the camera starts: above the player's eyes
	// and a few blocks back; startPitch is how far it looks down there.
	rise, behind, startPitch = 14.0, 4.0, 60.0
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
	sghook.SetCamera(p, keys[0].Pos, keys[0].Pitch, keys[0].Yaw, 0)
	sghook.GlideCamera(p, keys[1].Pos, keys[1].Pitch, keys[1].Yaw, glide)
	p.H().DoAfter(seconds(glide), func(tx *world.Tx, e world.Entity) {
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
// rotation: above the island and a few blocks behind the player looking
// down with the player's yaw, then the player's eyes.
func Keys(pos mgl64.Vec3, pitch, yaw float64) []sghook.CameraKey {
	eye := pos.Add(mgl64.Vec3{0, 1.62, 0})
	a := mgl64.DegToRad(yaw)
	forward := mgl64.Vec3{-math.Sin(a), 0, math.Cos(a)}
	high := eye.Sub(forward.Mul(behind)).Add(mgl64.Vec3{0, rise, 0})
	return []sghook.CameraKey{
		{T: 0, Pos: high, Pitch: startPitch, Yaw: yaw},
		{T: glide, Pos: eye, Pitch: pitch, Yaw: yaw, Ease: int32(protocol.EasingTypeInOutSine)},
	}
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }
