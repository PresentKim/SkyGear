package skygear

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
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/skygear/advancement"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
	"github.com/presentkim/skygear/internal/skygear/tinker"
)

// The dimension gate anchor: using it opens a gate and starts a wave of
// monsters the players must all kill before the time runs out. Clearing the
// highest level opens the next; first clears give essences, and monsters drop
// their mob's simulator drops. Values are drafts.

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
	{[]waveGroup{{sgitem.MobZombie, 4}}, 90, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 3)}},
	{[]waveGroup{{sgitem.MobZombie, 4}, {sgitem.MobSkeleton, 3}}, 100, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSkeleton}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobCreeper}, 1), item.NewStack(PlantSeed{Seed: "flower_red"}, 1)}},
	{[]waveGroup{{sgitem.MobSpider, 4}, {sgitem.MobCreeper, 3}, {sgitem.MobZombie, 3}, {sgitem.MobWitch, 2}}, 130, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSpider}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobCreeper}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobWitch}, 2)}},
	{[]waveGroup{{sgitem.MobSlime, 6}, {sgitem.MobSkeleton, 4}, {sgitem.MobZombie, 4}, {sgitem.MobDrowned, 3}}, 130, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSlime}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobDrowned}, 3)}},
	{[]waveGroup{{sgitem.MobEnder, 2}, {sgitem.MobZombie, 6}, {sgitem.MobSkeleton, 4}}, 150, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobEnder}, 3)}},
	{[]waveGroup{{sgitem.MobBlaze, 2}, {sgitem.MobCreeper, 6}, {sgitem.MobSpider, 6}, {sgitem.MobPiglin, 3}}, 160, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobBlaze}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobPiglin}, 3)}},
	{[]waveGroup{{sgitem.MobEnder, 3}, {sgitem.MobBlaze, 3}, {sgitem.MobZombie, 8}, {sgitem.MobPiglin, 3}}, 190, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobEnder}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobBlaze}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobPiglin}, 2)}},
	{[]waveGroup{{sgitem.MobEnder, 4}, {sgitem.MobBlaze, 4}, {sgitem.MobZombie, 6}, {sgitem.MobSkeleton, 6}, {sgitem.MobWither, 3}}, 260, []item.Stack{item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 8), item.NewStack(item.Diamond{}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobWither}, 3)}},
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

// monsterGone handles a monster that died to p (or to the void when p is
// nil, with no loot): it leaves the wave, drops its loot and ends the wave
// when no monster is left out or to come. A monster dying in the void stays
// until its death is over (Monster.Tick).
func (s *session) monsterGone(tx *world.Tx, m *Monster, p *player.Player) {
	w := s.wave
	if w == nil || !w.owns(m.handle) {
		_ = m.Close()
		return
	}
	pos := m.data.Pos
	delete(w.left, m.handle)
	if p != nil {
		s.advance(tx, advancement.Kill)
		count(p, "kill_total", 1)
		count(p, "kill_"+m.kind, 1)
		looting := 0
		if held, ok := toolOf(p); ok {
			looting += tinker.ToolMods(held)["looting"]
			s.gainToolXP(p, int(monsterTable[m.kind].health/2)+1)
		}
		for _, v := range tx.Viewers(pos) {
			v.ViewEntityAction(m, entity.DeathAction{})
		}
		for _, d := range sgitem.SimDrops[m.kind] {
			if s.rand.IntN(100) < d.Chance+looting*8 {
				s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(d.Item, d.Min+s.rand.IntN(d.Max-d.Min+1)))
			}
		}
		if w.level >= 2 && s.rand.IntN(100) < 4 {
			s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(PlantSeed{Seed: "flower_" + sgitem.PetalColours[s.rand.IntN(len(sgitem.PetalColours))]}, 1))
		}
		if s.rand.IntN(100) < waveEssenceChance+looting*4 {
			s.dropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(sgitem.Essence{Mob: m.kind}, 1))
		}
	}
	if m.md().dying == 0 {
		_ = m.Close()
	}
	if len(w.left) == 0 && len(w.pending) == 0 {
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
		s.advance(tx, advancement.Wave1)
		for _, a := range []struct {
			level int
			id    string
		}{{3, advancement.Wave3}, {5, advancement.Wave5}, {8, advancement.Wave8}} {
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
