package dragonfactory

import (
	"fmt"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/account"
)

// The projects (docs/design/10-user-flow.md §5): the long goal of the game.
// Three projects (industry, hunter, botanist) each ask for a pile of what one
// part of the game makes; the last one asks for the rarest things of all and
// opens the recipe of the dragon egg. Placing the egg on the island finishes
// the Dragon Factory. They are researches whose cost is the goal, so the
// research screen, the paying and the sharing of the island come for free.

// Achievements of the account that the game's milestones give; they open
// titles and, by their number, name colours (docs/design/01-platform.md §5).
const (
	achWave1  = "df_wave_1"
	achWave8  = "df_wave_8"
	achTinker = "df_tinker"
	achBloom  = "df_bloom"
	achDragon = "df_dragon"
)

// advAchievements gives an account achievement to everyone on the island when
// an advancement is done.
var advAchievements = map[string]string{
	advWave1:    achWave1,
	advWave8:    achWave8,
	advToolMade: achTinker,
	advBloom:    achBloom,
	advDragon:   achDragon,
}

func init() {
	ach := func(id, group string) account.Achievement {
		return account.Achievement{
			ID: id, Name: i18n.T(i18n.Default, "achievement."+id+".name"),
			Desc: i18n.T(i18n.Default, "achievement."+id+".desc"), Title: i18n.T(i18n.Default, "achievement."+id+".title"),
			Group: achievementGroup(group),
		}
	}
	account.Register(ach(achWave1, "combat"), ach(achWave8, "combat"), ach(achTinker, "gear"), ach(achBloom, "farm"), ach(achDragon, "progress"))
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
	if s.advanced(advDragon) {
		return
	}
	s.advance(tx, advDragon)
	s.broadcast(tx, fmt.Sprintf(i18n.T(i18n.Default, "project.dragon.done"), p.Name()))
	c := pos.Vec3Centre()
	for range 24 {
		at := c.Add(mgl64.Vec3{rand.Float64()*4 - 2, rand.Float64() * 3, rand.Float64()*4 - 2})
		tx.AddParticle(at, particle.EndermanTeleport{})
	}
	tx.PlaySound(c, sound.Explosion{})
}
