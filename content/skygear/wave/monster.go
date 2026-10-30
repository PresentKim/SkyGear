package wave

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Monsters of the dimension gate. Dragonfly has no mobs, so they are ours: an
// entity the client draws as the vanilla mob that walks to the nearest player
// and hits when close. A monster not in its session's current wave is removed.

// monsterStats are the numbers of a mob, drafts.
type monsterStats struct {
	network       string  // the vanilla entity the client draws
	Health        float64 // half hearts
	damage        float64 // per hit
	speed         float64 // blocks per tick
	width, height float64
	// ranged is the damage of an arrow the mob shoots (0: melee only) from between
	// holdRange and shootRange blocks away every shootEvery ticks.
	ranged, HoldRange, shootRange float64
	shootEvery                    int
}

var MonsterTable = map[string]monsterStats{
	sgitem.MobZombie:   {network: "minecraft:zombie", Health: 12, damage: 3, speed: 0.15, width: 0.6, height: 1.95},
	sgitem.MobSkeleton: {network: "minecraft:skeleton", Health: 10, damage: 2, speed: 0.17, width: 0.6, height: 1.99, ranged: 2, HoldRange: 6, shootRange: 14, shootEvery: 50},
	sgitem.MobCreeper:  {network: "minecraft:creeper", Health: 10, damage: 3, speed: 0.16, width: 0.6, height: 1.7},
	sgitem.MobSpider:   {network: "minecraft:spider", Health: 12, damage: 2, speed: 0.21, width: 1.4, height: 0.9},
	sgitem.MobSlime:    {network: "minecraft:slime", Health: 8, damage: 2, speed: 0.12, width: 0.52, height: 0.52},
	sgitem.MobDrowned:  {network: "minecraft:drowned", Health: 14, damage: 3, speed: 0.15, width: 0.6, height: 1.95},
	sgitem.MobWitch:    {network: "minecraft:witch", Health: 16, damage: 2, speed: 0.15, width: 0.6, height: 1.95, ranged: 3, HoldRange: 6, shootRange: 12, shootEvery: 70},
	sgitem.MobPiglin:   {network: "minecraft:piglin", Health: 18, damage: 4, speed: 0.18, width: 0.6, height: 1.95},
	sgitem.MobWither:   {network: "minecraft:wither_skeleton", Health: 26, damage: 5, speed: 0.17, width: 0.7, height: 2.4},
	sgitem.MobEnder:    {network: "minecraft:enderman", Health: 30, damage: 4, speed: 0.19, width: 0.6, height: 2.9},
	sgitem.MobBlaze:    {network: "minecraft:blaze", Health: 20, damage: 3, speed: 0.15, width: 0.6, height: 1.8, ranged: 3, HoldRange: 5, shootRange: 12, shootEvery: 60},
}

const (
	// monsterReach is how close (horizontally) a monster hits.
	monsterReach = 1.5
	// monsterCooldown is the ticks between two hits.
	monsterCooldown = 20
	// monsterSight is how far a monster notices a player.
	monsterSight = 40.0
	// monsterVoid is how far below the island a monster falls before the gate
	// takes it back.
	MonsterVoid = 12.0
)

// Monster is the entity of a mob of the gate.
type Monster struct {
	tx     *world.Tx
	Handle *world.EntityHandle
	Data   *world.EntityData
	kind   string
}

type monsterData struct {
	Mob      string
	Health   float64
	vel      mgl64.Vec3
	cooldown int
	onGround bool
	// exit is the way out of the gate it walks for exitTicks more ticks
	// before it goes for a player (gateportal.go).
	exit      mgl64.Vec3
	ExitTicks int
}

type MonsterConfig struct {
	Mob string
	// exit is the way out of the gate the monster came out of.
	exit mgl64.Vec3
}

func (c MonsterConfig) Apply(data *world.EntityData) {
	d := &monsterData{Mob: c.Mob, Health: MonsterTable[c.Mob].Health, exit: c.exit}
	if c.exit != (mgl64.Vec3{}) {
		d.ExitTicks = GateExitTicks
	}
	data.Data = d
}

func (m *Monster) H() *world.EntityHandle { return m.Handle }

func (m *Monster) Position() mgl64.Vec3 { return m.Data.Pos }

func (m *Monster) Rotation() cube.Rotation { return m.Data.Rot }

func (m *Monster) NameTag() string { return m.Data.Name }

func (m *Monster) AlwaysShowNameTag() bool { return false }

// Mob is the kind of the monster.
func (m *Monster) Mob() string { return m.kind }

func (m *Monster) Md() *monsterData { return m.Data.Data.(*monsterData) }

// Health is what the monster has left.
func (m *Monster) Health() float64 { return m.Md().Health }

func (m *Monster) Close() error {
	m.tx.RemoveEntity(m)
	return m.Handle.Close()
}

// MobTypes are the world.EntityTypes of the mobs, one each (an entity type
// has a single vanilla look).
var MobTypes = func() map[string]monsterType {
	out := map[string]monsterType{}
	for mob := range MonsterTable {
		out[mob] = monsterType{Mob: mob}
	}
	return out
}()

// MonsterTypes returns the entity types of the monsters.
func MonsterTypes() []world.EntityType {
	var out []world.EntityType
	for _, mob := range sgitem.MobKinds {
		out = append(out, MobTypes[mob])
	}
	return out
}

type monsterType struct{ Mob string }

func (t monsterType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &monsterData{Mob: t.Mob, Health: MonsterTable[t.Mob].Health}
	}
	return &Monster{tx: tx, Handle: handle, Data: data, kind: t.Mob}
}

func (t monsterType) EncodeEntity() string { return "skygear:monster_" + t.Mob }

func (t monsterType) NetworkEncodeEntity() string { return MonsterTable[t.Mob].network }

func (t monsterType) BBox(world.Entity) cube.BBox {
	st := MonsterTable[t.Mob]
	return cube.Box(-st.width/2, 0, -st.width/2, st.width/2, st.height, st.width/2)
}

func (t monsterType) DecodeNBT(_ map[string]any, d *world.EntityData) {
	d.Data = &monsterData{Mob: t.Mob, Health: MonsterTable[t.Mob].Health}
}

func (monsterType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// Tick walks the monster towards the nearest player and hits when close.
func (m *Monster) Tick(tx *world.Tx, _ int64) {
	s, ok := core.Of(tx)
	if !ok || LocalOf(s).Wave == nil || !LocalOf(s).Wave.Owns(m.Handle) {
		// from an earlier session or a finished wave
		_ = m.Close()
		return
	}
	d, st := m.Md(), MonsterTable[m.kind]
	pos := m.Data.Pos
	if pos.Y() < island.IslandY-MonsterVoid {
		returnToGate(s, tx, m)
		return
	}
	if d.ExitTicks > 0 {
		// still coming out of the gate
		d.ExitTicks--
		m.Data.Rot = cube.Rotation{dirYaw(d.exit), 0}
		m.move(tx, pos, d.exit, MonsterTable[m.kind])
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
			hold = st.HoldRange
		}
		if l := flat.Len(); l > hold {
			dir = flat.Mul(1 / l)
		}
		if st.ranged > 0 && d.cooldown == 0 && to.Len() <= st.shootRange && m.canSee(tx, pos, target) {
			d.cooldown = st.shootEvery
			m.shoot(tx, pos, target, st)
		}
		m.Data.Rot = cube.Rotation{mgl64.RadToDeg(math.Atan2(-to[0], to[2])), 0}
		if flat.Len() < monsterReach && math.Abs(to[1]) < 2 && d.cooldown == 0 {
			d.cooldown = monsterCooldown
			if _, hurt := target.Hurt(st.damage, entity.AttackDamageSource{Attacker: m}); hurt {
				away := mgl64.Vec3{-to[0], 0, -to[2]}.Normalize()
				target.KnockBack(m.Data.Pos, 0.4, 0.3)
				_ = away
			}
		}
	}
	m.move(tx, pos, dir, st)
}

// move applies one tick of walking, jumping and falling.
func (m *Monster) move(tx *world.Tx, pos, dir mgl64.Vec3, st monsterStats) {
	d := m.Md()
	feet := func(p mgl64.Vec3) cube.Pos { return cube.PosFromVec3(p) }
	// horizontal: blocked by a wall unless it can be stepped over
	if dir.Len() > 0 {
		next := pos.Add(dir.Mul(st.speed))
		front := feet(next.Add(dir.Mul(st.width / 2)))
		blockedLow := core.SolidAt(tx, front)
		blockedHigh := core.SolidAt(tx, front.Add(cube.Pos{0, 1, 0}))
		switch {
		case !blockedLow && !blockedHigh:
			pos = mgl64.Vec3{next[0], pos[1], next[2]}
		case blockedLow && !blockedHigh && !core.SolidAt(tx, front.Add(cube.Pos{0, 2, 0})) && d.onGround:
			d.vel[1] = 0.42 // jump over the block
			pos = mgl64.Vec3{next[0], pos[1], next[2]}
		}
	}
	// vertical: gravity and the floor
	d.vel[1] = (d.vel[1] - 0.08) * 0.98
	pos[1] += d.vel[1]
	cell := feet(pos)
	d.onGround = false
	if d.vel[1] <= 0 && core.SolidAt(tx, cell) {
		pos[1] = float64(cell.Y()) + 1
		d.vel[1] = 0
		d.onGround = true
	} else if d.vel[1] <= 0 && core.SolidAt(tx, feet(pos.Sub(mgl64.Vec3{0, 0.001, 0}))) {
		d.onGround = true
		d.vel[1] = 0
	}
	// knockback decays on the ground
	pos[0] += d.vel[0]
	pos[2] += d.vel[2]
	d.vel[0], d.vel[2] = d.vel[0]*0.7, d.vel[2]*0.7
	m.Data.Pos = pos
	for _, v := range tx.Viewers(pos) {
		v.ViewEntityMovement(m, pos, m.Data.Rot, d.onGround)
	}
}

// Hurt takes the damage of an attack by p: it flinches, is thrown back and, at
// no health, dies (the gate counts it, its drops fall).
func (m *Monster) Hurt(tx *world.Tx, dmg float64, from *player.Player) {
	d := m.Md()
	d.Health -= dmg
	away := m.Data.Pos.Sub(from.Position())
	away[1] = 0
	if away.Len() > 0 {
		away = away.Normalize().Mul(0.35)
	}
	d.vel = mgl64.Vec3{away[0], 0.3, away[2]}
	d.onGround = false
	for _, v := range tx.Viewers(m.Data.Pos) {
		v.ViewEntityAction(m, entity.HurtAction{})
	}
	tx.PlaySound(m.Data.Pos, sound.Attack{Damage: true})
	if d.Health <= 0 {
		if s, ok := core.Of(tx); ok {
			monsterGone(s, tx, m, from)
		}
	}
}

func MobName(lang, mob string) string { return i18n.T(lang, "mob."+mob) }

// canSee reports whether no solid block lies between the monster's eyes and p's.
func (m *Monster) canSee(tx *world.Tx, pos mgl64.Vec3, p *player.Player) bool {
	from, to := pos.Add(mgl64.Vec3{0, 1.5, 0}), p.Position().Add(mgl64.Vec3{0, 1.2, 0})
	line := to.Sub(from)
	steps := int(line.Len() * 2)
	for i := 1; i < steps; i++ {
		if core.SolidAt(tx, cube.PosFromVec3(from.Add(line.Mul(float64(i)/float64(steps))))) {
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
	tx.AddEntity(entity.NewArrowWithDamage(world.EntitySpawnOpts{Position: from, Velocity: dir, Rotation: m.Data.Rot}, st.ranged, m))
	tx.PlaySound(from, sound.BowShoot{})
}
