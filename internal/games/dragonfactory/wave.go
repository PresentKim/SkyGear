package dragonfactory

import (
	"fmt"
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The dimension gate (docs/design/05-sky-progression.md §4.1): using it starts a
// wave of monsters around it that the players must all kill before the time
// runs out. Clearing the highest level opens the next; each first clear gives
// essences, and every monster drops what its mob drops (the simulator's tables)
// and sometimes an essence. Values are drafts.

// waveGroup is a number of one kind of mob in a wave.
type waveGroup struct {
	Mob string
	N   int
}

// waveDef is one level of the gate.
type waveDef struct {
	Groups  []waveGroup
	Seconds int
	// Reward is what the first clear gives.
	Reward []item.Stack
}

// waveLevels are the levels of the gate, easiest first.
var waveLevels = []waveDef{
	{[]waveGroup{{MobZombie, 4}}, 90, []item.Stack{item.NewStack(Essence{Mob: MobZombie}, 3)}},
	{[]waveGroup{{MobZombie, 4}, {MobSkeleton, 3}}, 100, []item.Stack{item.NewStack(Essence{Mob: MobSkeleton}, 3), item.NewStack(Essence{Mob: MobCreeper}, 1)}},
	{[]waveGroup{{MobSpider, 4}, {MobCreeper, 3}, {MobZombie, 3}}, 120, []item.Stack{item.NewStack(Essence{Mob: MobSpider}, 3), item.NewStack(Essence{Mob: MobCreeper}, 3)}},
	{[]waveGroup{{MobSlime, 6}, {MobSkeleton, 4}, {MobZombie, 4}}, 120, []item.Stack{item.NewStack(Essence{Mob: MobSlime}, 4)}},
	{[]waveGroup{{MobEnder, 2}, {MobZombie, 6}, {MobSkeleton, 4}}, 150, []item.Stack{item.NewStack(Essence{Mob: MobEnder}, 3)}},
	{[]waveGroup{{MobBlaze, 2}, {MobCreeper, 6}, {MobSpider, 6}}, 150, []item.Stack{item.NewStack(Essence{Mob: MobBlaze}, 3)}},
	{[]waveGroup{{MobEnder, 3}, {MobBlaze, 3}, {MobZombie, 8}}, 180, []item.Stack{item.NewStack(Essence{Mob: MobEnder}, 4), item.NewStack(Essence{Mob: MobBlaze}, 4)}},
	{[]waveGroup{{MobEnder, 4}, {MobBlaze, 4}, {MobZombie, 6}, {MobSkeleton, 6}}, 240, []item.Stack{item.NewStack(Ingot{Metal: MetalSteel}, 8), item.NewStack(item.Diamond{}, 4)}},
}

// waveEssenceChance is the percent chance a killed monster drops its essence.
const waveEssenceChance = 12

// waveState is the wave being fought.
type waveState struct {
	level int
	gate  cube.Pos
	// left are the monsters alive, by handle.
	left     map[*world.EntityHandle]string
	deadline time.Time
}

func (w *waveState) owns(h *world.EntityHandle) bool { _, ok := w.left[h]; return ok }

// gateTier is the highest level cleared.
func (s *session) gateTier() int { return s.state.GateTier }

// canStartWave reports whether a level may be started: the next one to clear or
// any below.
func (s *session) canStartWave(level int) bool {
	return level >= 1 && level <= len(waveLevels) && level <= s.state.GateTier+1
}

// startWave spawns the wave of a level around the gate at pos.
func (s *session) startWave(tx *world.Tx, pos cube.Pos, level int) bool {
	if s.wave != nil || !s.canStartWave(level) {
		return false
	}
	def := waveLevels[level-1]
	w := &waveState{level: level, gate: pos, left: map[*world.EntityHandle]string{}, deadline: time.Now().Add(time.Duration(def.Seconds) * time.Second)}
	s.wave = w
	for _, g := range def.Groups {
		for range g.N {
			at, ok := s.waveSpawnPoint(tx, pos)
			if !ok {
				continue
			}
			h := world.EntitySpawnOpts{Position: at}.New(monsterTypes[g.Mob], monsterConfig{mob: g.Mob})
			w.left[h] = g.Mob
			tx.AddEntity(h)
		}
	}
	if len(w.left) == 0 {
		s.wave = nil
		return false
	}
	tx.PlaySound(pos.Vec3Centre(), sound.Explosion{})
	s.broadcast(tx, fmt.Sprintf(i18n.T(i18n.Default, "gate.start"), level, len(w.left), def.Seconds))
	return true
}

// waveSpawnPoint finds a free spot on solid ground 3 to 6 blocks from the gate.
func (s *session) waveSpawnPoint(tx *world.Tx, gate cube.Pos) (mgl64.Vec3, bool) {
	for range 40 {
		a := s.rand.Float64() * 2 * math.Pi
		r := 3 + s.rand.Float64()*3
		x, z := gate.X()+int(math.Round(r*math.Cos(a))), gate.Z()+int(math.Round(r*math.Sin(a)))
		for y := gate.Y() + 2; y >= gate.Y()-3; y-- {
			p := cube.Pos{x, y, z}
			if solidAt(tx, p) && !solidAt(tx, p.Add(cube.Pos{0, 1, 0})) && !solidAt(tx, p.Add(cube.Pos{0, 2, 0})) {
				return mgl64.Vec3{float64(x) + 0.5, float64(y) + 1, float64(z) + 0.5}, true
			}
		}
	}
	return mgl64.Vec3{}, false
}

// monsterGone handles a monster that died to p (or fell into the void when
// p is nil): it leaves the wave, drops its loot and may end the wave.
func (s *session) monsterGone(tx *world.Tx, m *Monster, p *player.Player) {
	w := s.wave
	if w == nil || !w.owns(m.handle) {
		_ = m.Close()
		return
	}
	pos := m.data.Pos
	delete(w.left, m.handle)
	if p != nil {
		looting := 0
		fortune := s.rpgFortune(p)
		s.gainXP(p, int(monsterTable[m.kind].health))
		if held, ok := toolOf(p); ok {
			looting += toolMods(held)["looting"]
			s.gainToolXP(p, int(monsterTable[m.kind].health/2)+1)
		}
		for _, v := range tx.Viewers(pos) {
			v.ViewEntityAction(m, entity.DeathAction{})
		}
		for _, d := range simDrops[m.kind] {
			if s.rand.IntN(100) < d.Chance+looting*8+fortune {
				s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(d.Item, d.Min+s.rand.IntN(d.Max-d.Min+1)))
			}
		}
		if s.rand.IntN(100) < waveEssenceChance+looting*4+fortune/2 {
			s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(Essence{Mob: m.kind}, 1))
		}
	}
	_ = m.Close()
	if len(w.left) == 0 {
		s.winWave(tx)
	}
}

// winWave ends the wave with a clear: the next level opens, the first clear of a
// level gives its reward.
func (s *session) winWave(tx *world.Tx) {
	w := s.wave
	s.wave = nil
	def := waveLevels[w.level-1]
	first := w.level > s.state.GateTier
	if first {
		s.state.GateTier = w.level
		s.storeState()
		for _, r := range def.Reward {
			s.dropAt(tx, w.gate.Vec3Centre().Add(mgl64.Vec3{0, 1.2, 0}), r)
		}
		s.advance(tx, advWave1)
		if w.level >= 5 {
			s.advance(tx, advWave5)
		}
	}
	tx.PlaySound(w.gate.Vec3Centre(), sound.Experience{})
	s.broadcast(tx, fmt.Sprintf(i18n.T(i18n.Default, "gate.win"), w.level))
	for e := range tx.Players() {
		s.gainXP(e.(*player.Player), 15*w.level)
	}
}

// failWave ends the wave without a clear: the monsters left vanish.
func (s *session) failWave(tx *world.Tx, key string) {
	w := s.wave
	if w == nil {
		return
	}
	s.wave = nil
	for h := range w.left {
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
	s.broadcast(tx, i18n.T(i18n.Default, key))
}

// tickWave runs every second: it tells the players how the wave goes and ends it
// when the time is out or the gate is gone.
func (s *session) tickWave(tx *world.Tx) {
	w := s.wave
	if w == nil {
		return
	}
	if m, ok := tx.Block(w.gate).(Machine); !ok || m.Kind != KindGate {
		s.failWave(tx, "gate.gone")
		return
	}
	left := int(time.Until(w.deadline).Seconds())
	if left <= 0 {
		s.failWave(tx, "gate.timeout")
		return
	}
	// monsters that were lost (closed elsewhere) do not count
	for h := range w.left {
		if h.Closed() {
			delete(w.left, h)
		}
	}
	if len(w.left) == 0 {
		s.winWave(tx)
		return
	}
	for e := range tx.Players() {
		e.(*player.Player).SendTip(fmt.Sprintf(i18n.T(i18n.Of(e.(*player.Player)), "gate.tip"), w.level, len(w.left), left))
	}
}

// showGateForm lets a player pick the level to start.
func (s *session) showGateForm(p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	var opts []ui.Option
	var lines string
	for i, def := range waveLevels {
		level := i + 1
		if !s.canStartWave(level) {
			break
		}
		label := fmt.Sprintf(i18n.T(lang, "gate.level"), level)
		if level <= s.state.GateTier {
			label = "§a" + label
		}
		var parts string
		for j, g := range def.Groups {
			if j > 0 {
				parts += ", "
			}
			parts += fmt.Sprintf("%s ×%d", i18n.T(lang, "mob."+g.Mob), g.N)
		}
		lines += fmt.Sprintf("\n§f%d §7%s (%d%s)", level, parts, def.Seconds, i18n.T(lang, "gate.seconds"))
		opts = append(opts, ui.Option{Label: label, Run: func(p *player.Player, tx *world.Tx) {
			s.startWave(tx, pos, level)
		}})
	}
	body := fmt.Sprintf(i18n.T(lang, "gate.body"), s.state.GateTier) + lines
	if s.wave != nil {
		body = i18n.T(lang, "gate.busy")
		opts = nil
	}
	ui.Menu(p, i18n.T(lang, "gate.title"), body, opts...)
}
