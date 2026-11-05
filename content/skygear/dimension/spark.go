package dimension

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The weapons draw with the pack's pixelfx pictures (particles/fx_*.json,
// written by tools/pixelfx/export.mjs). Upright ones face the camera and
// stand their foot on the spot under the hunt camera's view (fx_flame,
// fx_lightning, fx_poof shift along its up); the blade and the bolt lie on
// the ground, turned by the yaw and stretched by the length sent with them.
const (
	fxSlash     = "skygear:fx_slash"
	fxTracer    = "skygear:fx_tracer"
	fxFlame     = "skygear:fx_flame"
	fxDisc      = "skygear:fx_disc"
	fxLightning = "skygear:fx_lightning"
	fxHit       = "skygear:fx_hit"
	fxPoof      = "skygear:fx_poof"
	fxBlast     = "skygear:fx_blast"
	fxBlink     = "skygear:fx_blink"
	fxMend      = "skygear:fx_mend"
)

// sparkSight is how far (blocks) from a particle a player sees it.
const sparkSight = 48

// sparkVar is a MoLang variable sent with a particle, as variable.<name>.
type sparkVar struct {
	name  string
	value float64
}

// spark shows a particle of kind at pos to the players near it, with vars.
func spark(tx *world.Tx, pos mgl64.Vec3, kind string, vars ...sparkVar) {
	pk := &packet.SpawnParticleEffect{
		EntityUniqueID: -1,
		Position:       mgl32.Vec3{float32(pos[0]), float32(pos[1]), float32(pos[2])},
		ParticleName:   kind,
	}
	if len(vars) > 0 {
		pk.MoLangVariables = protocol.Option(molangJSON(vars))
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		if p.Position().Sub(pos).Len() <= sparkSight {
			sghook.WritePacket(sghook.Session(p), pk)
		}
	}
}

// molangJSON is vars as the client reads them with a particle: a list of
// named values with their type.
func molangJSON(vars []sparkVar) []byte {
	parts := make([]string, len(vars))
	for i, v := range vars {
		parts[i] = fmt.Sprintf(`{"name":"variable.%s","value":{"type":"float","value":%g}}`, v.name, v.value)
	}
	return []byte("[" + strings.Join(parts, ",") + "]")
}
