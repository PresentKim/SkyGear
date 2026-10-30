package wave

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
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
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

// Levels are the levels of the gate, easiest first.
var Levels = []waveDef{
	{[]waveGroup{{sgitem.MobZombie, 4}}, 90, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 3)}},
	{[]waveGroup{{sgitem.MobZombie, 4}, {sgitem.MobSkeleton, 3}}, 100, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSkeleton}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobCreeper}, 1), item.NewStack(farm.PlantSeed{Seed: "flower_red"}, 1)}},
	{[]waveGroup{{sgitem.MobSpider, 4}, {sgitem.MobCreeper, 3}, {sgitem.MobZombie, 3}, {sgitem.MobWitch, 2}}, 130, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSpider}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobCreeper}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobWitch}, 2)}},
	{[]waveGroup{{sgitem.MobSlime, 6}, {sgitem.MobSkeleton, 4}, {sgitem.MobZombie, 4}, {sgitem.MobDrowned, 3}}, 130, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobSlime}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobDrowned}, 3)}},
	{[]waveGroup{{sgitem.MobEnder, 2}, {sgitem.MobZombie, 6}, {sgitem.MobSkeleton, 4}}, 150, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobEnder}, 3)}},
	{[]waveGroup{{sgitem.MobBlaze, 2}, {sgitem.MobCreeper, 6}, {sgitem.MobSpider, 6}, {sgitem.MobPiglin, 3}}, 160, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobBlaze}, 3), item.NewStack(sgitem.Essence{Mob: sgitem.MobPiglin}, 3)}},
	{[]waveGroup{{sgitem.MobEnder, 3}, {sgitem.MobBlaze, 3}, {sgitem.MobZombie, 8}, {sgitem.MobPiglin, 3}}, 190, []item.Stack{item.NewStack(sgitem.Essence{Mob: sgitem.MobEnder}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobBlaze}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobPiglin}, 2)}},
	{[]waveGroup{{sgitem.MobEnder, 4}, {sgitem.MobBlaze, 4}, {sgitem.MobZombie, 6}, {sgitem.MobSkeleton, 6}, {sgitem.MobWither, 3}}, 260, []item.Stack{item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 8), item.NewStack(item.Diamond{}, 4), item.NewStack(sgitem.Essence{Mob: sgitem.MobWither}, 3)}},
}

// waveEssenceChance is the percent chance a killed monster drops its essence.
const waveEssenceChance = 12

// State is the wave being fought.
type State struct {
	Level int
	Gate  cube.Pos
	// left are the monsters alive, by handle.
	Left     map[*world.EntityHandle]string
	Deadline time.Time
	// pending are the mobs still to come out of the gate, in order; emerge
	// counts the Fast calls since the last one (gateportal.go).
	Pending []string
	emerge  int
	// out is the way out of the gate and portal the gate entity.
	Out    mgl64.Vec3
	Portal *world.EntityHandle
}

func (w *State) Owns(h *world.EntityHandle) bool { _, ok := w.Left[h]; return ok }

// CanStart reports whether a level may be started: the next one to clear or
// any below.
func CanStart(s *core.Session, level int) bool {
	return level >= 1 && level <= len(Levels) && level <= StateOf(s).GateTier+1
}

// Start opens the gate of a level over the anchor at pos, facing a player
// at from: its monsters come out of it one after another (gateportal.go).
func Start(s *core.Session, tx *world.Tx, pos cube.Pos, from mgl64.Vec3, level int) bool {
	if LocalOf(s).Wave != nil || !CanStart(s, level) {
		return false
	}
	def := Levels[level-1]
	w := &State{Level: level, Gate: pos, Left: map[*world.EntityHandle]string{}, Out: gateFacing(pos, from),
		Deadline: time.Now().Add(time.Duration(def.Seconds) * time.Second)}
	for _, g := range def.Groups {
		for range g.N {
			w.Pending = append(w.Pending, g.Mob)
		}
	}
	// the kinds come out mixed, not one group after the other
	s.Rand.Shuffle(len(w.Pending), func(i, j int) { w.Pending[i], w.Pending[j] = w.Pending[j], w.Pending[i] })
	LocalOf(s).Wave = w
	openGatePortal(s, tx, w)
	tx.PlaySound(pos.Vec3Centre(), sound.Explosion{})
	s.Broadcast(tx, "gate.start", level, len(w.Pending), def.Seconds)
	return true
}

// monsterGone handles a monster that died to p (or was lost when p is nil,
// with no loot): it leaves the wave, drops its loot and ends the wave when no
// monster is left out or to come.
func monsterGone(s *core.Session, tx *world.Tx, m *Monster, p *player.Player) {
	w := LocalOf(s).Wave
	if w == nil || !w.Owns(m.Handle) {
		_ = m.Close()
		return
	}
	pos := m.Data.Pos
	delete(w.Left, m.Handle)
	if p != nil {
		progress.Advance(s, tx, advancement.Kill)
		achievement.Count(p, "kill_total", 1)
		achievement.Count(p, "kill_"+m.kind, 1)
		looting := 0
		if held, ok := tinker.ToolOf(p); ok {
			looting += tinker.ToolMods(held)["looting"]
			tinker.GainXP(s, p, int(MonsterTable[m.kind].Health/2)+1)
		}
		for _, v := range tx.Viewers(pos) {
			v.ViewEntityAction(m, entity.DeathAction{})
		}
		for _, d := range sgitem.SimDrops[m.kind] {
			if s.Rand.IntN(100) < d.Chance+looting*8 {
				s.DropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(d.Item, d.Min+s.Rand.IntN(d.Max-d.Min+1)))
			}
		}
		if w.Level >= 2 && s.Rand.IntN(100) < 4 {
			s.DropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(farm.PlantSeed{Seed: "flower_" + sgitem.PetalColours[s.Rand.IntN(len(sgitem.PetalColours))]}, 1))
		}
		if s.Rand.IntN(100) < waveEssenceChance+looting*4 {
			s.DropAt(tx, pos.Add(mgl64.Vec3{0, 0.5, 0}), item.NewStack(sgitem.Essence{Mob: m.kind}, 1))
		}
	}
	_ = m.Close()
	if len(w.Left) == 0 && len(w.Pending) == 0 {
		winWave(s, tx)
	}
}

// winWave ends the wave with a clear: the next level opens, the first clear of a
// level gives its reward.
func winWave(s *core.Session, tx *world.Tx) {
	w := LocalOf(s).Wave
	LocalOf(s).Wave = nil
	closeGatePortal(s, tx, w)
	def := Levels[w.Level-1]
	first := w.Level > StateOf(s).GateTier
	if first {
		StateOf(s).GateTier = w.Level
		s.MarkDirty()
		for _, r := range def.Reward {
			s.DropAt(tx, w.Gate.Vec3Centre().Add(mgl64.Vec3{0, 1.2, 0}), r)
		}
		progress.Advance(s, tx, advancement.Wave1)
		for _, a := range []struct {
			level int
			id    string
		}{{3, advancement.Wave3}, {5, advancement.Wave5}, {8, advancement.Wave8}} {
			if w.Level >= a.level {
				progress.Advance(s, tx, a.id)
			}
		}
	}
	tx.PlaySound(w.Gate.Vec3Centre(), sound.Experience{})
	achievement.CountAll(tx, "wave_win", 1)
	achievement.CountAll(tx, fmt.Sprintf("wave_level_%d", w.Level), 1)
	s.Broadcast(tx, "gate.win", w.Level)
}

// failWave ends the wave without a clear: the monsters left vanish.
func failWave(s *core.Session, tx *world.Tx, key string) {
	w := LocalOf(s).Wave
	if w == nil {
		return
	}
	LocalOf(s).Wave = nil
	closeGatePortal(s, tx, w)
	achievement.CountAll(tx, "wave_fail", 1)
	for h := range w.Left {
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
	s.Broadcast(tx, key)
}

// Tick runs every second: it tells the players how the wave goes and ends it
// when the time is out or the gate is gone.
func Tick(s *core.Session, tx *world.Tx) {
	w := LocalOf(s).Wave
	if w == nil {
		return
	}
	if m, ok := tx.Block(w.Gate).(machine.Machine); !ok || m.Kind != machine.KindGate {
		failWave(s, tx, "gate.gone")
		return
	}
	left := int(time.Until(w.Deadline).Seconds())
	if left <= 0 {
		failWave(s, tx, "gate.timeout")
		return
	}
	// monsters that were lost (closed elsewhere) do not count
	for h := range w.Left {
		if h.Closed() {
			delete(w.Left, h)
		}
	}
	if len(w.Left) == 0 && len(w.Pending) == 0 {
		winWave(s, tx)
		return
	}
	for e := range tx.Players() {
		e.(*player.Player).SendTip(fmt.Sprintf(i18n.T(i18n.Of(e.(*player.Player)), "gate.tip"), w.Level, len(w.Left)+len(w.Pending), left))
	}
}

// ShowGate lets a player pick the level to start.
func ShowGate(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	var opts []ui.Option
	var lines string
	for i, def := range Levels {
		level := i + 1
		if !CanStart(s, level) {
			break
		}
		label := fmt.Sprintf(i18n.T(lang, "gate.level"), level)
		if level <= StateOf(s).GateTier {
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
			Start(s, tx, pos, p.Position(), level)
		}})
	}
	body := fmt.Sprintf(i18n.T(lang, "gate.body"), StateOf(s).GateTier) + lines
	if LocalOf(s).Wave != nil {
		body = i18n.T(lang, "gate.busy")
		opts = nil
	}
	ui.Menu(p, i18n.T(lang, "gate.title"), body, opts...)
}

// gateState is what a save keeps of its dimension gate: its fields are part of the
// save's state document (core.Saved).
type gateState struct {
	// GateTier is the highest level of the dimension gate cleared (wave.go).
	GateTier int `json:"gate_tier,omitempty"`
}

var gateSaved = core.NewSaved[gateState](nil)

// StateOf is what the save of s keeps of its dimension gate.
func StateOf(s *core.Session) *gateState { return gateSaved.Of(s) }

// wavesLocal is what a running save keeps for its waves.
type wavesLocal struct {
	// wave is the wave of the dimension gate being fought (wave.go).
	Wave *State
}

var wavesPart = core.NewPart(func() *wavesLocal { return &wavesLocal{} })

// LocalOf is what the save of s keeps for its waves.
func LocalOf(s *core.Session) *wavesLocal { return wavesPart.Of(s) }

// The dimension gate is a machine kind whose click opens the gate screen.
func init() {
	machine.Kinds[machine.KindGate].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
		if !s.Debounced(p) {
			ShowGate(s, p, tx, pos)
		}
		return true
	}
}
