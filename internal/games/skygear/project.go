package skygear

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/games/skygear/achievement"
	"github.com/presentkim/skygear/internal/games/skygear/advancement"
)

// The projects: the long goal of the game.
// Three projects (industry, hunter, botanist) each ask for a pile of what one
// part of the game makes; the last one asks for the rarest things of all and
// opens the recipe of the dragon egg. Placing the egg on the island finishes
// the Sky Gear. They are researches whose cost is the goal, so the
// research screen, the paying and the sharing of the island come for free.

// advAchievements gives an account achievement to everyone on the island when
// an advancement is done.
var advAchievements = map[string]string{
	advancement.Wave1:    achievement.Wave1,
	advancement.Wave8:    achievement.Wave8,
	advancement.ToolMade: achievement.Tinker,
	advancement.Bloom:    achievement.Bloom,
	advancement.Dragon:   achievement.Dragon,
}

func init() {
	ach := func(id, group string) account.Achievement { return account.Keyed(id, achievement.Group(group)) }
	account.Register(ach(achievement.Wave1, "combat"), ach(achievement.Wave8, "combat"), ach(achievement.Tinker, "gear"), ach(achievement.Bloom, "farm"), ach(achievement.Dragon, "progress"))
}

// grantAdvancementAchievements gives the achievements of advancements just done.
func (s *session) grantAdvancementAchievements(tx *world.Tx, ids []string) {
	for _, id := range ids {
		ach, ok := advAchievements[id]
		if !ok {
			continue
		}
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), ach)
		}
	}
}

// completeDragon is the end of the game: the dragon egg is placed. The
// advancement is the record; the island celebrates.
func (s *session) completeDragon(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if s.advanced(advancement.Dragon) {
		return
	}
	s.advance(tx, advancement.Dragon)
	s.broadcast(tx, "project.dragon.done", p.Name())
	c := pos.Vec3Centre()
	for range 24 {
		at := c.Add(mgl64.Vec3{rand.Float64()*4 - 2, rand.Float64() * 3, rand.Float64()*4 - 2})
		tx.AddParticle(at, particle.EndermanTeleport{})
	}
	tx.PlaySound(c, sound.Explosion{})
}
