package dragonfactory

import (
	"fmt"
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The character (docs/design/08-storage-gear-magic.md §3): every player of a save
// has experience from fighting monsters, a level that grows with its square root,
// and a skill point per level to put into skills. Tools have levels of their own
// (tinker.go); this is the player's.

// RPGRecord is the character of one player.
type RPGRecord struct {
	XP     int            `json:"xp,omitempty"`
	Skills map[string]int `json:"skills,omitempty"`
}

const (
	rpgXPUnit   = 40
	rpgMaxLevel = 40
)

// rpgSkill is a skill a point can go into.
type rpgSkill struct {
	id  string
	max int
}

var rpgSkills = []rpgSkill{
	{"vitality", 10}, // +2 max health a point
	{"strength", 10}, // +6% damage to monsters a point
	{"mining", 10},   // haste from 3 points, a second level from 7
	{"agility", 10},  // speed from 4 points, a second level from 9
	{"fortune", 10},  // +3% monster drops a point
}

// rpgLevel is the level of an experience total.
func rpgLevel(xp int) int { return min(rpgMaxLevel, int(math.Sqrt(float64(xp)/rpgXPUnit))) }

// rpgXPFor is the experience a level starts at.
func rpgXPFor(level int) int { return level * level * rpgXPUnit }

// rpgOf returns the record of p (created on first use).
func (s *session) rpgOf(p *player.Player) *RPGRecord {
	if s.state.RPG == nil {
		s.state.RPG = map[string]*RPGRecord{}
	}
	id := instance.AccountID(p)
	r := s.state.RPG[id]
	if r == nil {
		r = &RPGRecord{Skills: map[string]int{}}
		s.state.RPG[id] = r
	}
	if r.Skills == nil {
		r.Skills = map[string]int{}
	}
	return r
}

func (r *RPGRecord) spent() int {
	n := 0
	for _, v := range r.Skills {
		n += v
	}
	return n
}

// points are the skill points not spent yet.
func (r *RPGRecord) points() int { return rpgLevel(r.XP) - r.spent() }

// damageMul is the multiplier of the damage p deals to monsters.
func (s *session) rpgDamageMul(p *player.Player) float64 {
	return 1 + 0.06*float64(s.rpgOf(p).Skills["strength"])
}

// rpgFortune is the percent added to the chance of monster drops.
func (s *session) rpgFortune(p *player.Player) int { return 3 * s.rpgOf(p).Skills["fortune"] }

// gainXP adds experience to p's character and says so on a level up.
func (s *session) gainXP(p *player.Player, n int) {
	r := s.rpgOf(p)
	before := rpgLevel(r.XP)
	r.XP += n
	if now := rpgLevel(r.XP); now > before {
		p.Messagef(i18n.T(i18n.Of(p), "rpg.levelUp"), now, r.points())
		for _, a := range []struct {
			level int
			id    string
		}{{5, advChar5}, {15, advChar15}, {30, advChar30}} {
			if now >= a.level {
				s.advance(p.Tx(), a.id)
			}
		}
		s.applyRPG(p)
		s.storeState()
	}
}

// applyRPG sets what the skills give: the most health, the speed and haste
// effects (renewed every second by tickToolEffects).
func (s *session) applyRPG(p *player.Player) {
	r := s.rpgOf(p)
	if want := 20 + 2*float64(r.Skills["vitality"]); p.MaxHealth() != want {
		p.SetMaxHealth(want)
	}
}

// rpgEffects gives the effects of the skills for the next few seconds.
func (s *session) rpgEffects(p *player.Player, toolHaste int) {
	r := s.rpgOf(p)
	haste := toolHaste
	if m := r.Skills["mining"]; m >= 7 {
		haste = max(haste, 2)
	} else if m >= 3 {
		haste = max(haste, 1)
	}
	if haste > 0 {
		p.AddEffect(effect.NewAmbient(effect.Haste, haste, 3*time.Second))
	}
	switch a := r.Skills["agility"]; {
	case a >= 9:
		p.AddEffect(effect.NewAmbient(effect.Speed, 2, 3*time.Second))
	case a >= 4:
		p.AddEffect(effect.NewAmbient(effect.Speed, 1, 3*time.Second))
	}
	s.applyRPG(p)
}

// showCharacter is the character screen: the level, and a button for each
// skill that spends a point.
func (s *session) showCharacter(p *player.Player) {
	lang := i18n.Of(p)
	r := s.rpgOf(p)
	lvl := rpgLevel(r.XP)
	body := fmt.Sprintf(i18n.T(lang, "rpg.body"), lvl, r.XP, rpgXPFor(min(lvl+1, rpgMaxLevel)), r.points())
	var opts []ui.Option
	for _, sk := range rpgSkills {
		cur := r.Skills[sk.id]
		label := fmt.Sprintf("%s  §7%d/%d", i18n.T(lang, "rpg.skill."+sk.id), cur, sk.max)
		opts = append(opts, ui.Option{Label: label, Run: func(p *player.Player, tx *world.Tx) {
			r := s.rpgOf(p)
			if r.points() > 0 && r.Skills[sk.id] < sk.max {
				r.Skills[sk.id]++
				s.advance(tx, advSkill)
				s.applyRPG(p)
				s.storeState()
			}
			s.showCharacter(p)
		}})
		body += "\n§f" + i18n.T(lang, "rpg.skill."+sk.id) + " §7" + i18n.T(lang, "rpg.info."+sk.id)
	}
	// the title settings live here too
	opts = append(opts, ui.Option{Label: i18n.T(lang, "dragonfactory.screens.mainMenu.7"), Icon: iconTitle, Run: func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }})
	ui.Menu(p, i18n.T(lang, "rpg.title"), body, opts...)
}
