package dragonfactory

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Monsters of the dimension gate (docs/design/05-sky-progression.md §4.1,
// docs/design/07-progression-expansion.md §7). Dragonfly has no mobs, so they are
// ours: an entity the client draws as the vanilla mob, which walks towards the
// nearest player, steps up blocks, falls, hits when close, and takes the damage
// of the player's weapon. Only the gate's waves spawn them (wave.go); a monster
// that does not belong to the current wave of its session is removed.

// monsterStats are the numbers of a mob, drafts.
type monsterStats struct {
	network       string  // the vanilla entity the client draws
	health        float64 // half hearts
	damage        float64 // per hit
	speed         float64 // blocks per tick
	width, height float64
	// ranged is the damage of an arrow the mob shoots (0: melee only) from between
	// holdRange and shootRange blocks away every shootEvery ticks.
	ranged, holdRange, shootRange float64
	shootEvery                    int
}

var monsterTable = map[string]monsterStats{
	MobZombie:   {network: "minecraft:zombie", health: 12, damage: 3, speed: 0.15, width: 0.6, height: 1.95},
	MobSkeleton: {network: "minecraft:skeleton", health: 10, damage: 2, speed: 0.17, width: 0.6, height: 1.99, ranged: 2, holdRange: 6, shootRange: 14, shootEvery: 50},
	MobCreeper:  {network: "minecraft:creeper", health: 10, damage: 3, speed: 0.16, width: 0.6, height: 1.7},
	MobSpider:   {network: "minecraft:spider", health: 12, damage: 2, speed: 0.21, width: 1.4, height: 0.9},
	MobSlime:    {network: "minecraft:slime", health: 8, damage: 2, speed: 0.12, width: 0.52, height: 0.52},
	MobDrowned:  {network: "minecraft:drowned", health: 14, damage: 3, speed: 0.15, width: 0.6, height: 1.95},
	MobWitch:    {network: "minecraft:witch", health: 16, damage: 2, speed: 0.15, width: 0.6, height: 1.95, ranged: 3, holdRange: 6, shootRange: 12, shootEvery: 70},
	MobPiglin:   {network: "minecraft:piglin", health: 18, damage: 4, speed: 0.18, width: 0.6, height: 1.95},
	MobWither:   {network: "minecraft:wither_skeleton", health: 26, damage: 5, speed: 0.17, width: 0.7, height: 2.4},
	MobEnder:    {network: "minecraft:enderman", health: 30, damage: 4, speed: 0.19, width: 0.6, height: 2.9},
	MobBlaze:    {network: "minecraft:blaze", health: 20, damage: 3, speed: 0.15, width: 0.6, height: 1.8, ranged: 3, holdRange: 5, shootRange: 12, shootEvery: 60},
}

const (
	// monsterReach is how close (horizontally) a monster hits.
	monsterReach = 1.5
	// monsterCooldown is the ticks between two hits.
	monsterCooldown = 20
	// monsterSight is how far a monster notices a player.
	monsterSight = 40.0
	// monsterVoid is how far below the island a monster falls before it is lost.
	monsterVoid = 12.0
)

// Monster is the entity of a mob of the gate.
type Monster struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
	kind   string
}

type monsterData struct {
	mob      string
	health   float64
	vel      mgl64.Vec3
	cooldown int
	onGround bool
	// lost is set when the monster fell into the void.
	lost bool
}

type monsterConfig struct{ mob string }

func (c monsterConfig) Apply(data *world.EntityData) {
	data.Data = &monsterData{mob: c.mob, health: monsterTable[c.mob].health}
}

func (m *Monster) H() *world.EntityHandle  { return m.handle }
func (m *Monster) Position() mgl64.Vec3    { return m.data.Pos }
func (m *Monster) Rotation() cube.Rotation { return m.data.Rot }
func (m *Monster) NameTag() string         { return m.data.Name }
func (m *Monster) AlwaysShowNameTag() bool { return false }

// Mob is the kind of the monster.
func (m *Monster) Mob() string { return m.kind }

func (m *Monster) md() *monsterData { return m.data.Data.(*monsterData) }

// Health is what the monster has left.
func (m *Monster) Health() float64 { return m.md().health }

func (m *Monster) Close() error {
	m.tx.RemoveEntity(m)
	return m.handle.Close()
}

// monsterTypes are the world.EntityTypes of the mobs, one each (an entity type
// has a single vanilla look).
var monsterTypes = func() map[string]monsterType {
	out := map[string]monsterType{}
	for mob := range monsterTable {
		out[mob] = monsterType{mob: mob}
	}
	return out
}()

// MonsterTypes returns the entity types of the monsters.
func MonsterTypes() []world.EntityType {
	var out []world.EntityType
	for _, mob := range mobKinds {
		out = append(out, monsterTypes[mob])
	}
	return out
}

type monsterType struct{ mob string }

func (t monsterType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &monsterData{mob: t.mob, health: monsterTable[t.mob].health}
	}
	return &Monster{tx: tx, handle: handle, data: data, kind: t.mob}
}
func (t monsterType) EncodeEntity() string        { return "dragonfactory:monster_" + t.mob }
func (t monsterType) NetworkEncodeEntity() string { return monsterTable[t.mob].network }
func (t monsterType) BBox(world.Entity) cube.BBox {
	st := monsterTable[t.mob]
	return cube.Box(-st.width/2, 0, -st.width/2, st.width/2, st.height, st.width/2)
}
func (t monsterType) DecodeNBT(_ map[string]any, d *world.EntityData) {
	d.Data = &monsterData{mob: t.mob, health: monsterTable[t.mob].health}
}
func (monsterType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// solidAt reports whether the block at p stops a monster.
func solidAt(tx *world.Tx, p cube.Pos) bool {
	b := tx.Block(p)
	if _, liquid := b.(world.Liquid); liquid {
		return false
	}
	return len(b.Model().BBox(p, tx)) > 0
}

// Tick walks the monster towards the nearest player and hits when close.
func (m *Monster) Tick(tx *world.Tx, _ int64) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok || s.wave == nil || !s.wave.owns(m.handle) {
		// from an earlier session or a finished wave
		_ = m.Close()
		return
	}
	d, st := m.md(), monsterTable[m.kind]
	pos := m.data.Pos
	if pos.Y() < IslandY-monsterVoid {
		d.lost = true
		s.monsterGone(tx, m, nil)
		return
	}
	var target *player.Player
	best := monsterSight
	for e := range tx.Players() {
		p := e.(*player.Player)
		if p.Dead() || !p.GameMode().AllowsTakingDamage() {
			continue
		}
		if dist := p.Position().Sub(pos).Len(); dist < best {
			best, target = dist, p
		}
	}
	if d.cooldown > 0 {
		d.cooldown--
	}
	var dir mgl64.Vec3
	if target != nil {
		to := target.Position().Sub(pos)
		flat := mgl64.Vec3{to[0], 0, to[2]}
		hold := monsterReach * 0.6
		if st.ranged > 0 {
			hold = st.holdRange
		}
		if l := flat.Len(); l > hold {
			dir = flat.Mul(1 / l)
		}
		if st.ranged > 0 && d.cooldown == 0 && to.Len() <= st.shootRange && m.canSee(tx, pos, target) {
			d.cooldown = st.shootEvery
			m.shoot(tx, pos, target, st)
		}
		m.data.Rot = cube.Rotation{mgl64.RadToDeg(math.Atan2(-to[0], to[2])), 0}
		if flat.Len() < monsterReach && math.Abs(to[1]) < 2 && d.cooldown == 0 {
			d.cooldown = monsterCooldown
			if _, hurt := target.Hurt(st.damage, entity.AttackDamageSource{Attacker: m}); hurt {
				away := mgl64.Vec3{-to[0], 0, -to[2]}.Normalize()
				target.KnockBack(m.data.Pos, 0.4, 0.3)
				_ = away
			}
		}
	}
	m.move(tx, pos, dir, st)
}

// move applies one tick of walking, jumping and falling.
func (m *Monster) move(tx *world.Tx, pos, dir mgl64.Vec3, st monsterStats) {
	d := m.md()
	feet := func(p mgl64.Vec3) cube.Pos { return cube.PosFromVec3(p) }
	// horizontal: blocked by a wall unless it can be stepped over
	if dir.Len() > 0 {
		next := pos.Add(dir.Mul(st.speed))
		front := feet(next.Add(dir.Mul(st.width / 2)))
		blockedLow := solidAt(tx, front)
		blockedHigh := solidAt(tx, front.Add(cube.Pos{0, 1, 0}))
		switch {
		case !blockedLow && !blockedHigh:
			pos = mgl64.Vec3{next[0], pos[1], next[2]}
		case blockedLow && !blockedHigh && !solidAt(tx, front.Add(cube.Pos{0, 2, 0})) && d.onGround:
			d.vel[1] = 0.42 // jump over the block
			pos = mgl64.Vec3{next[0], pos[1], next[2]}
		}
	}
	// vertical: gravity and the floor
	d.vel[1] = (d.vel[1] - 0.08) * 0.98
	pos[1] += d.vel[1]
	cell := feet(pos)
	d.onGround = false
	if d.vel[1] <= 0 && solidAt(tx, cell) {
		pos[1] = float64(cell.Y()) + 1
		d.vel[1] = 0
		d.onGround = true
	} else if d.vel[1] <= 0 && solidAt(tx, feet(pos.Sub(mgl64.Vec3{0, 0.001, 0}))) {
		d.onGround = true
		d.vel[1] = 0
	}
	// knockback decays on the ground
	pos[0] += d.vel[0]
	pos[2] += d.vel[2]
	d.vel[0], d.vel[2] = d.vel[0]*0.7, d.vel[2]*0.7
	m.data.Pos = pos
	for _, v := range tx.Viewers(pos) {
		v.ViewEntityMovement(m, pos, m.data.Rot, d.onGround)
	}
}

// Hurt takes the damage of an attack by p: it flinches, is thrown back and, at
// no health, dies (the gate counts it, its drops fall).
func (m *Monster) Hurt(tx *world.Tx, dmg float64, from *player.Player) {
	d := m.md()
	d.health -= dmg
	away := m.data.Pos.Sub(from.Position())
	away[1] = 0
	if away.Len() > 0 {
		away = away.Normalize().Mul(0.35)
	}
	d.vel = mgl64.Vec3{away[0], 0.3, away[2]}
	d.onGround = false
	for _, v := range tx.Viewers(m.data.Pos) {
		v.ViewEntityAction(m, entity.HurtAction{})
	}
	tx.PlaySound(m.data.Pos, sound.Attack{Damage: true})
	if d.health <= 0 {
		if s, ok := activeGame.sessionOf(tx.World()); ok {
			s.monsterGone(tx, m, from)
		}
	}
}

// mobName is the name of a mob in lang.
func mobName(lang, mob string) string { return i18n.T(lang, "mob."+mob) }

// canSee reports whether no solid block lies between the monster's eyes and p's.
func (m *Monster) canSee(tx *world.Tx, pos mgl64.Vec3, p *player.Player) bool {
	from, to := pos.Add(mgl64.Vec3{0, 1.5, 0}), p.Position().Add(mgl64.Vec3{0, 1.2, 0})
	line := to.Sub(from)
	steps := int(line.Len() * 2)
	for i := 1; i < steps; i++ {
		if solidAt(tx, cube.PosFromVec3(from.Add(line.Mul(float64(i)/float64(steps))))) {
			return false
		}
	}
	return true
}

// shoot lets a ranged mob shoot an arrow at p.
func (m *Monster) shoot(tx *world.Tx, pos mgl64.Vec3, p *player.Player, st monsterStats) {
	from := pos.Add(mgl64.Vec3{0, 1.5, 0})
	to := p.Position().Add(mgl64.Vec3{0, 1.2, 0})
	dir := to.Sub(from)
	dist := dir.Len()
	// aim a little high for the drop of the arrow over the distance
	dir = dir.Add(mgl64.Vec3{0, dist * 0.04, 0}).Normalize().Mul(1.7)
	tx.AddEntity(entity.NewArrowWithDamage(world.EntitySpawnOpts{Position: from, Velocity: dir, Rotation: m.data.Rot}, st.ranged, m))
	tx.PlaySound(from, sound.BowShoot{})
}
