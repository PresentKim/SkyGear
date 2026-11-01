package dimension

import (
	"fmt"
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// A hunter fights with the relics of its kit (kit.go) and only walks:
// weapon relics strike on their own, charms make it stronger. Kills leave experience gems; the
// hunter picks them up within its magnet's reach, and every level stops the
// hunt for a choice of three upgrades. What the relics and the levels give
// lasts the run. Values are drafts.

const (
	// weaponMaxLevel is the highest level a weapon reaches in a run.
	weaponMaxLevel = 5
	// gemXP is the experience a gem gives; baseMagnet is the reach (blocks)
	// gems are picked up from.
	gemXP      = 2
	baseMagnet = 3.0
)

// gear is what a hunter brings into a run and gains in it.
type gear struct {
	weapons []*weapon
	charms  map[string]int
	// picks count the common upgrades chosen at level-ups.
	picks map[string]int
	// level is the hunter's level in the run; gems are the experience
	// gems lying on the ground (marks.go).
	level int
	gems  []*gem
	regen float64
	// speed is the speed factor the gear applies now.
	speed float64
	// was is how the hunter was before the run, given back after it.
	was struct {
		maxHealth, speed float64
		xpLevel          int
		xpProgress       float64
	}
}

// weapon is a weapon relic in a run.
type weapon struct {
	kind         string
	grade, level int
	// cool counts down to the next strike; hit counts down per monster
	// until it may be hit again.
	cool int
	hit  map[*world.EntityHandle]int
	// angle turns the blades; the boomerang's flight is out (ticks into it,
	// 0 when it is in hand) along dir.
	angle float64
	out   int
	dir   mgl64.Vec3
}

// The common upgrades of a level-up.
const (
	upAttack = "attack"
	upSpeed  = "speed"
	upHealth = "health"
	upRegen  = "regen"
	upMagnet = "magnet"
)

var commonUps = []string{upAttack, upSpeed, upHealth, upRegen, upMagnet}

// charmGrade is the grade of a charm the hunter brought, 0 when none.
func (g *gear) charmGrade(kind string) int { return g.charms[kind] }

// charmBonus is base for a charm of grade 1 and step more per grade above.
func (g *gear) charmBonus(kind string, base, step float64) float64 {
	gr := g.charmGrade(kind)
	if gr == 0 {
		return 0
	}
	return base + step*float64(gr-1)
}

func (g *gear) attack() float64 { return 1 + 0.1*float64(g.picks[upAttack]) }

func (g *gear) cooldown() float64 {
	return max(0.5, 1-g.charmBonus(sgitem.RelicHourglass, 0.08, 0.03))
}

func (g *gear) reach() float64 { return 1 + g.charmBonus(sgitem.RelicLens, 0.15, 0.05) }

func (g *gear) magnet() float64 {
	return baseMagnet * (1 + g.charmBonus(sgitem.RelicMagnet, 0.5, 0.15) + 0.3*float64(g.picks[upMagnet]))
}

func (g *gear) maxHealth() float64 {
	return 20 + g.charmBonus(sgitem.RelicLife, 4, 2) + 4*float64(g.picks[upHealth])
}

func (g *gear) speedMul() float64 {
	return 1 + g.charmBonus(sgitem.RelicWind, 0.1, 0.05) + 0.08*float64(g.picks[upSpeed])
}

// regenPerSecond is the health the hunter gets back a second.
func (g *gear) regenPerSecond() float64 {
	return g.charmBonus(sgitem.RelicRegen, 0.3, 0.15) + 0.2*float64(g.picks[upRegen])
}

// gearOf is the gear p brings: the relics of its kit (kit.go). Two of a
// kind count once, the higher grade.
func gearOf(s *core.Session, p *player.Player) *gear {
	g := &gear{charms: map[string]int{}, picks: map[string]int{}}
	weapons := map[string]*weapon{}
	for _, it := range kitRelicStacks(s, p) {
		r, grade := it.Item().(sgitem.Relic), sgitem.RelicGrade(it)
		switch {
		case !r.IsWeapon():
			g.charms[r.Kind] = max(g.charms[r.Kind], grade)
		case weapons[r.Kind] != nil:
			weapons[r.Kind].grade = max(weapons[r.Kind].grade, grade)
		default:
			w := &weapon{kind: r.Kind, grade: grade, level: 1, hit: map[*world.EntityHandle]int{}}
			weapons[r.Kind] = w
			g.weapons = append(g.weapons, w)
		}
	}
	return g
}

// carried is the highest grade of the relic kind p carries.
func carried(p *player.Player, kind string) (int, bool) {
	best := 0
	for _, it := range p.Inventory().Items() {
		if r, ok := it.Item().(sgitem.Relic); ok && r.Kind == kind {
			best = max(best, sgitem.RelicGrade(it))
		}
	}
	return best, best > 0
}

// dress makes p the hunter its gear says, keeping how it was.
func (g *gear) dress(p *player.Player) {
	g.was.maxHealth, g.was.speed = p.MaxHealth(), p.Speed()
	g.was.xpLevel, g.was.xpProgress = p.ExperienceLevel(), p.ExperienceProgress()
	p.SetExperienceLevel(0)
	p.SetExperienceProgress(0)
	g.refit(p)
	p.Heal(p.MaxHealth(), entity.FoodHealingSource{})
}

// refit applies the gear's health and speed to p. The speed changes by the
// factor, so an effect on it (slowness) undoes itself rightly later.
func (g *gear) refit(p *player.Player) {
	p.SetMaxHealth(g.maxHealth())
	if g.speed == 0 {
		g.speed = 1
	}
	p.SetSpeed(p.Speed() * g.speedMul() / g.speed)
	g.speed = g.speedMul()
}

// undress gives p back how it was before the run. The effects that change
// the speed end first, or their end would change the speed given back.
func (g *gear) undress(p *player.Player) {
	p.SetMobile() // a run that ends during a level-up choice
	p.RemoveEffect(effect.Slowness)
	p.RemoveEffect(effect.Speed)
	p.SetMaxHealth(g.was.maxHealth)
	p.SetSpeed(g.was.speed)
	p.SetExperienceLevel(g.was.xpLevel)
	p.SetExperienceProgress(g.was.xpProgress)
}

// tick strikes with the weapons, heals, draws and picks up the gems.
func (g *gear) tick(s *core.Session, tx *world.Tx, p *player.Player, r *Run) {
	mons := r.monsters(tx)
	for _, w := range g.weapons {
		for h, n := range w.hit {
			if n <= 1 {
				delete(w.hit, h)
			} else {
				w.hit[h] = n - 1
			}
		}
		w.strike(s, tx, p, g, mons)
		if RunOf(s) != r {
			return
		}
		mons = slices.DeleteFunc(mons, func(m *Monster) bool { return !r.Owns(m.Handle) })
	}
	if g.regen += g.regenPerSecond() / 20; g.regen >= 0.5 {
		p.Heal(g.regen, entity.FoodHealingSource{})
		g.regen = 0
	}
	g.collect(tx, p, r)
}

// rotateY turns v by deg degrees around the vertical.
func rotateY(v mgl64.Vec3, deg float64) mgl64.Vec3 {
	r := deg * math.Pi / 180
	c, s := math.Cos(r), math.Sin(r)
	return mgl64.Vec3{v[0]*c - v[2]*s, v[1], v[0]*s + v[2]*c}
}

// gem is an experience gem on the ground: its entity, and whether the
// hunter's magnet caught it (then it flies to the hunter, however far it
// gets).
type gem struct {
	h      *world.EntityHandle
	caught bool
	speed  float64
}

const (
	// gemPull is how fast (blocks a tick) a caught gem starts flying,
	// gemPullUp how much faster it gets each tick; within gemTake of the
	// hunter it is picked up.
	gemPull   = 0.25
	gemPullUp = 0.05
	gemTake   = 0.8
	// gemHeight is how high over the ground a gem lies.
	gemHeight = 1.0
)

// drop leaves a gem on the ground where a monster died.
func (g *gear) drop(tx *world.Tx, p *player.Player, r *Run, at mgl64.Vec3) {
	at[1] = float64(core.ArenaCentre.Y()) + gemHeight
	h := world.EntitySpawnOpts{Position: at}.New(GemType, markConfig{run: r})
	tx.AddEntity(h)
	g.gems = append(g.gems, &gem{h: h})
	g.collect(tx, p, r)
}

// collect lets the magnet catch the gems within its reach, flies the caught
// ones to the hunter and picks up those that reach it. The others lie still.
func (g *gear) collect(tx *world.Tx, p *player.Player, r *Run) {
	reach := g.magnet()
	to := p.Position().Add(mgl64.Vec3{0, 0.6, 0})
	got := 0
	g.gems = slices.DeleteFunc(g.gems, func(gm *gem) bool {
		e, ok := gm.h.Entity(tx)
		if !ok {
			return true
		}
		m := e.(*mark)
		way := to.Sub(m.Data.Pos)
		dist := way.Len()
		if !gm.caught && dist > reach {
			return false // lying where it fell (the client turns and bobs it)
		}
		if !gm.caught {
			gm.caught, gm.speed = true, gemPull
		}
		if dist <= gemTake+gm.speed {
			_ = m.Close()
			got++
			return true
		}
		m.moveTo(tx, m.Data.Pos.Add(way.Mul(gm.speed/dist)))
		gm.speed += gemPullUp
		return false
	})
	if got > 0 {
		p.AddExperience(got * gemXP)
		tx.PlaySound(p.Position(), sound.Experience{})
		if p.ExperienceLevel() > g.level {
			r.levelUp(tx, p)
		}
	}
}

// levelUp stops the hunt and offers p three upgrades.
func (r *Run) levelUp(tx *world.Tx, p *player.Player) {
	if r.paused {
		return // the next one comes after this choice
	}
	r.gear.level++
	r.paused = true
	r.pauses++
	p.SetImmobile() // everything stops: the hunter too
	p.SetVelocity(mgl64.Vec3{})
	r.choices = r.offer(r.sess)
	tx.PlaySound(p.Position(), sound.LevelUp{})
	r.showChoices(p)
}

// showChoices shows the hunter the cards of the level-up it waits on. The
// hunt waits as long as it takes; closing the cards shows them again.
func (r *Run) showChoices(p *player.Player) {
	pause := r.pauses
	lang := i18n.Of(p)
	var cards []ui.PickCard
	for _, c := range r.choices {
		card := c.card(lang)
		card.Run = func(p *player.Player, tx *world.Tx) {
			if sess, ok := core.Of(tx); ok && sess == r.sess && RunOf(r.sess) == r && r.paused && r.pauses == pause {
				r.choose(tx, p, c)
			}
		}
		cards = append(cards, card)
	}
	ui.PickCardsModal(p, fmt.Sprintf(i18n.T(lang, "dimension.levelUp.title"), r.gear.level), i18n.T(lang, "dimension.levelUp.body"), cards,
		func(p *player.Player, tx *world.Tx) {
			if sess, ok := core.Of(tx); ok && sess == r.sess && RunOf(r.sess) == r && r.paused && r.pauses == pause {
				r.showChoices(p)
			}
		})
}

// card draws a choice as a card: a weapon's next level, or a common upgrade.
func (c choice) card(lang string) ui.PickCard {
	if c.weapon != nil {
		rarity := "rare"
		if c.weapon.level+1 >= weaponMaxLevel {
			rarity = "epic"
		}
		return ui.PickCard{
			Icon:  formicon.Of(item.NewStack(sgitem.Relic{Kind: c.weapon.kind}, 1)),
			Tag:   fmt.Sprintf(i18n.T(lang, "dimension.card.level"), c.weapon.level+1),
			Title: i18n.T(lang, "relic."+c.weapon.kind), Rarity: rarity,
			Text: levelGains(lang, c.weapon.kind, c.weapon.level+1),
		}
	}
	return ui.PickCard{
		Icon: formicon.Of(item.NewStack(upIcons[c.common], 1)), Tag: i18n.T(lang, "dimension.card.common"),
		Title: i18n.T(lang, "dimension.up."+c.common), Rarity: "common",
		Text: i18n.T(lang, "dimension.up."+c.common+".about"),
	}
}

// upIcons are the items the common upgrades' cards show.
var upIcons = map[string]world.Item{
	upAttack: item.Sword{Tier: item.ToolTierIron}, upSpeed: item.Feather{}, upHealth: item.GoldenApple{},
	upRegen: item.GlisteringMelonSlice{}, upMagnet: item.Compass{},
}

// choice is an upgrade offered at a level-up: a weapon's next level or a
// common upgrade.
type choice struct {
	weapon *weapon
	common string
}

// offer picks three upgrades at random among the weapons' next levels and
// the common upgrades.
func (r *Run) offer(s *core.Session) []choice {
	var all []choice
	for _, w := range r.gear.weapons {
		if w.level < weaponMaxLevel {
			all = append(all, choice{weapon: w})
		}
	}
	for _, c := range commonUps {
		all = append(all, choice{common: c})
	}
	s.Rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	return all[:min(3, len(all))]
}

// choose applies an upgrade and lets the hunt go on (or offers the next
// level's).
func (r *Run) choose(tx *world.Tx, p *player.Player, c choice) {
	if c.weapon != nil {
		c.weapon.level = min(weaponMaxLevel, c.weapon.level+1)
	} else {
		r.gear.picks[c.common]++
		r.gear.refit(p)
		if c.common == upHealth {
			p.Heal(4, entity.FoodHealingSource{})
		}
	}
	r.paused, r.choices = false, nil
	p.SetMobile()
	if p.ExperienceLevel() > r.gear.level {
		r.levelUp(tx, p)
	}
}

// strike lets a weapon do what it does this tick.
func (w *weapon) strike(s *core.Session, tx *world.Tx, p *player.Player, g *gear, mons []*Monster) {
	dmg := func(base float64) float64 {
		return base * (1 + 0.25*float64(w.grade-1)) * (1 + 0.2*float64(w.level-1)) * g.attack()
	}
	ready := func(every int) bool {
		if w.cool > 0 {
			w.cool--
			return false
		}
		w.cool = max(1, int(float64(every)*g.cooldown()))
		return true
	}
	eye := p.Position().Add(mgl64.Vec3{0, 1.2, 0})
	switch w.kind {
	case sgitem.RelicBlade:
		w.spin(tx, p, g, mons, dmg(weaponDamage(w.kind)))
	case sgitem.RelicCrossbow:
		if !ready(20) {
			return
		}
		n := 1
		if w.level >= 2 {
			n++
		}
		if w.level >= 4 {
			n++
		}
		for _, m := range nearest(mons, p.Position(), 16*g.reach(), n) {
			line(tx, eye, m.Position().Add(mgl64.Vec3{0, 0.6, 0}), sparkBolt)
			m.Hurt(tx, dmg(weaponDamage(w.kind)), p)
		}
		tx.PlaySound(eye, sound.BowShoot{})
	case sgitem.RelicFlameRing:
		if !ready(10) {
			return
		}
		radius := (3 + 0.4*float64(w.level-1)) * g.reach()
		for i := range 12 {
			a := float64(i) * math.Pi / 6
			tx.AddParticle(p.Position().Add(mgl64.Vec3{math.Cos(a) * radius, 0.3, math.Sin(a) * radius}), particle.Flame{})
		}
		for _, m := range mons {
			if flatDist(m.Position(), p.Position()) <= radius {
				m.Hurt(tx, dmg(weaponDamage(w.kind)), p)
			}
		}
	case sgitem.RelicLightning:
		if !ready(40) {
			return
		}
		var near []*Monster
		for _, m := range mons {
			if flatDist(m.Position(), p.Position()) <= 12*g.reach() {
				near = append(near, m)
			}
		}
		s.Rand.Shuffle(len(near), func(i, j int) { near[i], near[j] = near[j], near[i] })
		for _, m := range near[:min(len(near), 2+w.level)] {
			at := m.Position()
			line(tx, at.Add(mgl64.Vec3{0, 6, 0}), at, sparkBolt)
			tx.PlaySound(at, sound.Thunder{})
			m.Hurt(tx, dmg(weaponDamage(w.kind)), p)
		}
	case sgitem.RelicBoomerang:
		w.fly(tx, p, g, mons, dmg(weaponDamage(w.kind)))
	}
}

// spin turns the blades around the hunter and cuts the monsters they pass:
// two, three from level 3, four at level 5.
func (w *weapon) spin(tx *world.Tx, p *player.Player, g *gear, mons []*Monster, dmg float64) {
	count := 2
	if w.level >= 3 {
		count++
	}
	if w.level >= 5 {
		count++
	}
	w.angle = math.Mod(w.angle+bladeTurn, 2*math.Pi)
	inner, outer := bladeInner*g.reach(), bladeOuter*g.reach()
	centre := p.Position().Add(mgl64.Vec3{0, 0.9, 0})
	dirs := make([]mgl64.Vec3, count)
	for i := range count {
		a := w.angle + float64(i)*2*math.Pi/float64(count)
		dirs[i] = mgl64.Vec3{math.Cos(a), 0, math.Sin(a)}
		for k := range bladeSparks {
			spark(tx, centre.Add(dirs[i].Mul(inner+(outer-inner)*float64(k)/(bladeSparks-1))), sparkBlade)
		}
	}
	for _, m := range mons {
		if w.hit[m.Handle] > 0 {
			continue
		}
		off := m.Position().Sub(p.Position())
		if math.Abs(off[1]) > 2.5 {
			continue
		}
		flat := mgl64.Vec3{off[0], 0, off[2]}
		for _, dir := range dirs {
			along := flat.Dot(dir)
			if along < inner-bladeReach || along > outer+bladeReach {
				continue
			}
			if flat.Sub(dir.Mul(along)).Len() <= bladeReach+MonsterTable[m.kind].width*m.Scale()/2 {
				w.hit[m.Handle] = bladeCooldown
				m.Hurt(tx, dmg, p)
				break
			}
		}
	}
}

// fly throws the boomerang the way the hunter looks and brings it back,
// cutting and slowing what it passes.
func (w *weapon) fly(tx *world.Tx, p *player.Player, g *gear, mons []*Monster, dmg float64) {
	const flight = 30
	if w.out == 0 {
		if w.cool > 0 {
			w.cool--
			return
		}
		look := p.Rotation().Vec3()
		w.dir = mgl64.Vec3{look[0], 0, look[2]}
		if w.dir.Len() < 0.01 {
			w.dir = mgl64.Vec3{1, 0, 0}
		}
		w.dir = w.dir.Normalize()
		tx.PlaySound(p.Position(), sound.ItemThrow{})
	}
	w.out++
	span := (9 + float64(w.level-1)) * g.reach()
	along := span * (1 - math.Abs(float64(w.out)-flight/2)/(flight/2))
	at := p.Position().Add(mgl64.Vec3{0, 1, 0}).Add(w.dir.Mul(along))
	spark(tx, at, sparkFrost)
	for _, m := range mons {
		if w.hit[m.Handle] == 0 && m.Position().Add(mgl64.Vec3{0, 0.5, 0}).Sub(at).Len() <= 1.4 {
			w.hit[m.Handle] = 10
			m.Md().slow = 40
			m.Hurt(tx, dmg, p)
		}
	}
	if w.out >= flight {
		w.out = 0
		w.cool = max(1, int(30*g.cooldown()))
	}
}

// nearest are up to n monsters within reach of pos, the nearest first.
func nearest(mons []*Monster, pos mgl64.Vec3, reach float64, n int) []*Monster {
	var in []*Monster
	for _, m := range mons {
		if m.Position().Sub(pos).Len() <= reach {
			in = append(in, m)
		}
	}
	slices.SortFunc(in, func(a, b *Monster) int {
		return int(math.Copysign(1, a.Position().Sub(pos).Len()-b.Position().Sub(pos).Len()))
	})
	return in[:min(n, len(in))]
}

func flatDist(a, b mgl64.Vec3) float64 { return math.Hypot(a[0]-b[0], a[2]-b[2]) }

// line draws a straight bolt of sparks from a to b.
func line(tx *world.Tx, a, b mgl64.Vec3, kind string) {
	d := b.Sub(a)
	steps := max(1, int(d.Len()*2))
	for i := range steps + 1 {
		spark(tx, a.Add(d.Mul(float64(i)/float64(steps))), kind)
	}
}
