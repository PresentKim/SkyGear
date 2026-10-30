package skygear

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// The projects: the long goal of the game.
// Three projects (industry, hunter, botanist) each ask for a pile of what one
// part of the game makes; the last one asks for the rarest things of all and
// opens the recipe of the dragon egg. Placing the egg on the island finishes
// the Sky Gear. They are researches whose cost is the goal, so the
// research screen, the paying and the sharing of the island come for free.

// completeDragon is the end of the game: the dragon egg is placed. The
// advancement is the record; the island celebrates.
func (s *session) completeDragon(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if progress.Advanced(s.Session, advancement.Dragon) {
		return
	}
	progress.Advance(s.Session, tx, advancement.Dragon)
	s.Broadcast(tx, "project.dragon.done", p.Name())
	c := pos.Vec3Centre()
	for range 24 {
		at := c.Add(mgl64.Vec3{rand.Float64()*4 - 2, rand.Float64() * 3, rand.Float64()*4 - 2})
		tx.AddParticle(at, particle.EndermanTeleport{})
	}
	tx.PlaySound(c, sound.Explosion{})
}
