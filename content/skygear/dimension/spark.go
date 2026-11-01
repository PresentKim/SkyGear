package dimension

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The weapons draw with the pack's sparks (particles/hunt_*.json): a dot that
// stands still where it is put and is gone after 0.15 s, so a shape drawn
// every tick around the hunter keeps its shape however the hunter walks or
// turns (vanilla dust drifts and stays a second, smearing it).

// The kinds of sparks: their colour.
const (
	sparkBlade = "skygear:hunt_blade"
	sparkFrost = "skygear:hunt_frost"
	sparkBolt  = "skygear:hunt_bolt"
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
