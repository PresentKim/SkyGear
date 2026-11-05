package dimension

import (
	"math"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// How each weapon looks, apart from what it hits: the weapons (gear.go) and
// their showcase (fxdemo.go) draw with these.

// flameGap is about how far apart (blocks) the flames of the ring stand.
const flameGap = 1.2

// drawBlade lays a slash from inner to outer along dir from centre. One is
// laid every tick and each cools from white to a dotted trace, so the
// turning blades leave their sweep behind them.
func drawBlade(tx *world.Tx, centre, dir mgl64.Vec3, inner, outer float64) {
	at := centre.Add(dir.Mul((inner + outer) / 2))
	spark(tx, at, fxSlash, yawOf(dir), sparkVar{"len", (outer - inner) * 1.1})
}

// drawBolt lays a crossbow bolt's tracer from a to b and strikes there.
func drawBolt(tx *world.Tx, a, b mgl64.Vec3) {
	d := b.Sub(a)
	flat := mgl64.Vec3{d[0], 0, d[2]}
	spark(tx, a.Add(d.Mul(0.5)), fxTracer, yawOf(flat), sparkVar{"len", flat.Len() / 2})
	spark(tx, b, fxHit)
}

// yawOf is the turn (degrees) that lays a picture's right along dir on the
// ground.
func yawOf(dir mgl64.Vec3) sparkVar {
	return sparkVar{"yaw", mgl64.RadToDeg(math.Atan2(-dir[2], dir[0]))}
}

// drawFlameRing stands flames round feet at radius.
func drawFlameRing(tx *world.Tx, feet mgl64.Vec3, radius float64) {
	n := max(8, int(2*math.Pi*radius/flameGap))
	for i := range n {
		a := float64(i) * 2 * math.Pi / float64(n)
		spark(tx, feet.Add(mgl64.Vec3{math.Cos(a) * radius, 0, math.Sin(a) * radius}), fxFlame)
	}
}

// drawLightning drops a bolt from the sky on feet.
func drawLightning(tx *world.Tx, feet mgl64.Vec3) {
	spark(tx, feet, fxLightning)
	spark(tx, feet.Add(mgl64.Vec3{0, 0.4, 0}), fxHit)
}

// drawDisc draws the boomerang at at, spinning.
func drawDisc(tx *world.Tx, at mgl64.Vec3) { spark(tx, at, fxDisc) }

// drawPoof is a monster going up in smoke at feet.
func drawPoof(tx *world.Tx, feet mgl64.Vec3) { spark(tx, feet, fxPoof) }

// drawBlast is a monster blowing up at feet.
func drawBlast(tx *world.Tx, feet mgl64.Vec3) { spark(tx, feet, fxBlast) }

// drawBlink is a monster vanishing from or appearing at feet.
func drawBlink(tx *world.Tx, feet mgl64.Vec3) { spark(tx, feet, fxBlink) }

// drawMend is a monster at feet being mended.
func drawMend(tx *world.Tx, feet mgl64.Vec3) { spark(tx, feet, fxMend) }
