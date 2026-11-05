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

// drawBlade draws a blade from inner to outer along dir from centre: the
// steel cooling from a white tip to the hub, and an afterimage off its outer
// half that fades behind it as it turns.
func drawBlade(tx *world.Tx, centre, dir mgl64.Vec3, inner, outer float64) {
	for k := range bladeSparks {
		at := centre.Add(dir.Mul(inner + (outer-inner)*float64(k)/(bladeSparks-1)))
		kind := sparkBlade
		if k == bladeSparks-1 {
			kind = sparkBladeTip
		}
		spark(tx, at, kind)
		if k >= bladeSparks/2 {
			spark(tx, at, sparkBladeTrail)
		}
	}
}

// drawBolt draws a crossbow bolt flying from a to b and striking there.
func drawBolt(tx *world.Tx, a, b mgl64.Vec3) {
	line(tx, a, b, sparkBolt)
	spark(tx, b, fxHit)
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

// line draws a straight line of sparks of kind from a to b.
func line(tx *world.Tx, a, b mgl64.Vec3, kind string) {
	d := b.Sub(a)
	steps := max(1, int(d.Len()*2))
	for i := range steps + 1 {
		spark(tx, a.Add(d.Mul(float64(i)/float64(steps))), kind)
	}
}
