package dimension

import (
	"image/color"
	"math"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

// The spinning blades: two blades turn around the hunter and hurt every
// monster they pass. They are the hunter's weapon until relics come.

const (
	bladeCount = 2
	// bladeInner and bladeOuter are where a blade starts and ends, from the
	// hunter; bladeTurn is how far the blades turn a tick (a turn in 1.5 s).
	bladeInner, bladeOuter = 1.2, 2.6
	bladeTurn              = 2 * math.Pi / 30
	// bladeReach is how close to a blade a monster is hit.
	bladeReach = 0.8
	// bladeDamage is a hit's damage; a monster is hit again after
	// bladeCooldown ticks at the soonest.
	bladeDamage   = 4.0
	bladeCooldown = 10
)

var bladeColour = color.RGBA{R: 190, G: 225, B: 255, A: 255}

// blade is the state of a run's blades.
type blade struct {
	angle float64
	// hit counts down per monster until it may be hit again.
	hit map[*world.EntityHandle]int
}

// tick turns the blades one tick, draws them and hurts the monsters on them.
func (b *blade) tick(tx *world.Tx, p *player.Player, r *Run) {
	if b.hit == nil {
		b.hit = map[*world.EntityHandle]int{}
	}
	for h, n := range b.hit {
		if n <= 1 || !r.Alive[h] {
			delete(b.hit, h)
		} else {
			b.hit[h] = n - 1
		}
	}
	b.angle = math.Mod(b.angle+bladeTurn, 2*math.Pi)
	centre := p.Position().Add(mgl64.Vec3{0, 0.9, 0})
	var dirs [bladeCount]mgl64.Vec3
	for i := range bladeCount {
		a := b.angle + float64(i)*2*math.Pi/bladeCount
		dirs[i] = mgl64.Vec3{math.Cos(a), 0, math.Sin(a)}
		for _, d := range []float64{bladeInner, (bladeInner + bladeOuter) / 2, bladeOuter} {
			tx.AddParticle(centre.Add(dirs[i].Mul(d)), particle.Dust{Colour: bladeColour})
		}
	}
	for h := range r.Alive {
		if b.hit[h] > 0 {
			continue
		}
		e, ok := h.Entity(tx)
		if !ok {
			continue
		}
		m := e.(*Monster)
		off := m.Position().Sub(p.Position())
		if math.Abs(off[1]) > 2.5 {
			continue
		}
		flat := mgl64.Vec3{off[0], 0, off[2]}
		for _, dir := range dirs {
			along := flat.Dot(dir)
			if along < bladeInner-bladeReach || along > bladeOuter+bladeReach {
				continue
			}
			if flat.Sub(dir.Mul(along)).Len() <= bladeReach+MonsterTable[m.kind].width/2 {
				b.hit[h] = bladeCooldown
				m.Hurt(tx, bladeDamage, p)
				break
			}
		}
	}
}
