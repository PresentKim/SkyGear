package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The dimension gate anchor (docs/design/05-sky-progression.md §4.1): using it
// opens a gate over it and starts a wave of monsters that come out of the gate
// (gateportal.go), which the players must all kill before the time runs out. Clearing the highest level opens the next; each first clear gives
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
	{[]waveGroup{{MobZombie, 4}, {MobSkeleton, 3}}, 100, []item.Stack{item.NewStack(Essence{Mob: MobSkeleton}, 3), item.NewStack(Essence{Mob: MobCreeper}, 1), item.NewStack(PlantSeed{Seed: "flower_red"}, 1)}},
	{[]waveGroup{{MobSpider, 4}, {MobCreeper, 3}, {MobZombie, 3}, {MobWitch, 2}}, 130, []item.Stack{item.NewStack(Essence{Mob: MobSpider}, 3), item.NewStack(Essence{Mob: MobCreeper}, 3), item.NewStack(Essence{Mob: MobWitch}, 2)}},
	{[]waveGroup{{MobSlime, 6}, {MobSkeleton, 4}, {MobZombie, 4}, {MobDrowned, 3}}, 130, []item.Stack{item.NewStack(Essence{Mob: MobSlime}, 4), item.NewStack(Essence{Mob: MobDrowned}, 3)}},
	{[]waveGroup{{MobEnder, 2}, {MobZombie, 6}, {MobSkeleton, 4}}, 150, []item.Stack{item.NewStack(Essence{Mob: MobEnder}, 3)}},
	{[]waveGroup{{MobBlaze, 2}, {MobCreeper, 6}, {MobSpider, 6}, {MobPiglin, 3}}, 160, []item.Stack{item.NewStack(Essence{Mob: MobBlaze}, 3), item.NewStack(Essence{Mob: MobPiglin}, 3)}},
	{[]waveGroup{{MobEnder, 3}, {MobBlaze, 3}, {MobZombie, 8}, {MobPiglin, 3}}, 190, []item.Stack{item.NewStack(Essence{Mob: MobEnder}, 4), item.NewStack(Essence{Mob: MobBlaze}, 4), item.NewStack(Essence{Mob: MobPiglin}, 2)}},
	{[]waveGroup{{MobEnder, 4}, {MobBlaze, 4}, {MobZombie, 6}, {MobSkeleton, 6}, {MobWither, 3}}, 260, []item.Stack{item.NewStack(Ingot{Metal: MetalSteel}, 8), item.NewStack(item.Diamond{}, 4), item.NewStack(Essence{Mob: MobWither}, 3)}},
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
	// pending are the mobs still to come out of the gate, in order; emerge
	// counts the Fast calls since the last one (gateportal.go).
	pending []string
	emerge  int
	// out is the way out of the gate and portal the gate entity.
	out    mgl64.Vec3
	portal *world.EntityHandle
}

func (w *waveState) owns(h *world.EntityHandle) bool { _, ok := w.left[h]; return ok }

// canStartWave reports whether a level may be started: the next one to clear or
// any below.
func (s *session) canStartWave(level int) bool {
	return level >= 1 && level <= len(waveLevels) && level <= s.state.GateTier+1
}

// startWave opens the gate of a level over the anchor at pos, facing a player
// at from: its monsters come out of it one after another (gateportal.go).
func (s *session) startWave(tx *world.Tx, pos cube.Pos, from mgl64.Vec3, level int) bool {
	if s.wave != nil || !s.canStartWave(level) {
		return false
	}
	def := waveLevels[level-1]
	w := &waveState{level: level, gate: pos, left: map[*world.EntityHandle]string{}, out: gateFacing(pos, from),
		deadline: time.Now().Add(time.Duration(def.Seconds) * time.Second)}
	for _, g := range def.Groups {
		for range g.N {
			w.pending = append(w.pending, g.Mob)
		}
	}
	// the kinds come out mixed, not one group after the other
	s.rand.Shuffle(len(w.pending), func(i, j int) { w.pending[i], w.pending[j] = w.pending[j], w.pending[i] })
	s.wave = w
	s.openGatePortal(tx, w)
	tx.PlaySound(pos.Vec3Centre(), sound.Explosion{})
	s.broadcast(tx, "gate.start", level, len(w.pending), def.Seconds)
	return true
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
		s.advance(tx, advKill)
		count(p, "kill_total", 1)
		count(p, "kill_"+m.kind, 1)
		looting := 0
		if held, ok := toolOf(p); ok {
			looting += toolMods(held)["looting"]
			s.gainToolXP(p, int(monsterTable[m.kind].health/2)+1)
		}
		for _, v := range tx.Viewers(pos) {
			v.ViewEntityAction(m, entity.DeathAction{})
		}
		for _, d := range simDrops[m.kind] {
			if s.rand.IntN(100) < d.Chance+looting*8 {
				s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(d.Item, d.Min+s.rand.IntN(d.Max-d.Min+1)))
			}
		}
		if w.level >= 2 && s.rand.IntN(100) < 4 {
			s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(PlantSeed{Seed: "flower_" + petalColours[s.rand.IntN(len(petalColours))]}, 1))
		}
		if s.rand.IntN(100) < waveEssenceChance+looting*4 {
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
	s.closeGatePortal(tx, w)
	def := waveLevels[w.level-1]
	first := w.level > s.state.GateTier
	if first {
		s.state.GateTier = w.level
		s.storeState()
		for _, r := range def.Reward {
			s.dropAt(tx, w.gate.Vec3Centre().Add(mgl64.Vec3{0, 1.2, 0}), r)
		}
		s.advance(tx, advWave1)
		for _, a := range []struct {
			level int
			id    string
		}{{3, advWave3}, {5, advWave5}, {8, advWave8}} {
			if w.level >= a.level {
				s.advance(tx, a.id)
			}
		}
	}
	tx.PlaySound(w.gate.Vec3Centre(), sound.Experience{})
	countAll(tx, "wave_win", 1)
	countAll(tx, fmt.Sprintf("wave_level_%d", w.level), 1)
	s.broadcast(tx, "gate.win", w.level)
}

// failWave ends the wave without a clear: the monsters left vanish.
func (s *session) failWave(tx *world.Tx, key string) {
	w := s.wave
	if w == nil {
		return
	}
	s.wave = nil
	s.closeGatePortal(tx, w)
	countAll(tx, "wave_fail", 1)
	for h := range w.left {
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
	s.broadcast(tx, key)
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
	if len(w.left) == 0 && len(w.pending) == 0 {
		s.winWave(tx)
		return
	}
	for e := range tx.Players() {
		e.(*player.Player).SendTip(fmt.Sprintf(i18n.T(i18n.Of(e.(*player.Player)), "gate.tip"), w.level, len(w.left)+len(w.pending), left))
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
			s.startWave(tx, pos, p.Position(), level)
		}})
	}
	body := fmt.Sprintf(i18n.T(lang, "gate.body"), s.state.GateTier) + lines
	if s.wave != nil {
		body = i18n.T(lang, "gate.busy")
		opts = nil
	}
	ui.Menu(p, i18n.T(lang, "gate.title"), body, opts...)
}
