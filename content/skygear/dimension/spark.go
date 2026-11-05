package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The weapons draw with the pack's particles. The sparks (particles/hunt_*.json)
// are dots that stand still where they are put and are gone in a moment, so a
// shape drawn every tick around the hunter keeps its shape however the hunter
// walks or turns (vanilla dust drifts and stays a second, smearing it). The
// pictures (particles/fx_*.json) are pixelfx animations, written by
// tools/pixelfx/export.mjs.
const (
	sparkBlade      = "skygear:hunt_blade"
	sparkBladeTip   = "skygear:hunt_blade_tip"
	sparkBladeTrail = "skygear:hunt_blade_trail"
	sparkBolt       = "skygear:hunt_bolt"
	fxFlame         = "skygear:fx_flame"
	fxDisc          = "skygear:fx_disc"
	fxLightning     = "skygear:fx_lightning"
	fxHit           = "skygear:fx_hit"
	fxPoof          = "skygear:fx_poof"
)

// sparkSight is how far (blocks) from a spark a player sees it.
const sparkSight = 48

// spark shows a spark of kind at pos to the players near it.
func spark(tx *world.Tx, pos mgl64.Vec3, kind string) {
	pk := &packet.SpawnParticleEffect{
		EntityUniqueID: -1,
		Position:       mgl32.Vec3{float32(pos[0]), float32(pos[1]), float32(pos[2])},
		ParticleName:   kind,
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		if p.Position().Sub(pos).Len() <= sparkSight {
			sghook.WritePacket(sghook.Session(p), pk)
		}
	}
}
