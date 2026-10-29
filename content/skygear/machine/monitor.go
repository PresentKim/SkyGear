package machine

import (
	"fmt"
	"math"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/platform/instance"
)

// The machine list and the stall alerts: every machine that ticks is on the
// island's list; once a second the list is walked to judge what each machine
// is doing.

// machineStatus is what a machine is doing.
type machineStatus int

const (
	statusIdle machineStatus = iota
	StatusWorking
	StatusStarved
	statusPower
	StatusOutput
	statusFuel
)

// stuckKind is why a machine could not work in its last tick (0: nothing is
// wrong).
type stuckKind int

const (
	stuckNone stuckKind = iota
	stuckPower
	StuckOutput
	stuckFuel
)

// Timings of the alerts, in seconds.
const (
	AlertAfter     = 30
	alertSpacing   = 300
	starvedWindow  = 300
	alertLines     = 3
	machineStale   = 10
	machineForget  = 300
	uptimeMachines = 10
)

// monitor is the state of a machine for the stall alerts.
type monitor struct {
	Status machineStatus
	// since is when the status began, lastWork when the machine last
	// worked, lastAlert when it was last alerted (seconds of the island's
	// clock; 0: never).
	since, lastWork, LastAlert int
	Alerted                    bool
	// seen is the tick it last ran.
	seen int64
}

func (s machineStatus) alerts() bool {
	return s == statusPower || s == StatusOutput || s == statusFuel
}

// reasonKey is the text key of a status.
func (s machineStatus) reasonKey() string {
	switch s {
	case StatusWorking:
		return "monitor.reason.working"
	case StatusStarved:
		return "monitor.reason.starved"
	case statusPower:
		return "monitor.reason.power"
	case StatusOutput:
		return "monitor.reason.output"
	case statusFuel:
		return "monitor.reason.fuel"
	}
	return "monitor.reason.idle"
}

// judge decides what the machine did in the last second.
func (st *MachineState) judge(now int) machineStatus {
	m := &st.Mon
	var cur machineStatus
	switch {
	case st.didWork:
		cur = StatusWorking
		m.lastWork = now
	case st.Stuck == stuckPower:
		cur = statusPower
	case st.Stuck == StuckOutput:
		cur = StatusOutput
	case st.Stuck == stuckFuel && m.lastWork > 0:
		cur = statusFuel
	case m.lastWork > 0 && now-m.lastWork < starvedWindow:
		cur = StatusStarved
	}
	st.didWork = false
	if cur != m.Status {
		m.Status, m.since = cur, now
	}
	return cur
}

// reason is the status of a machine in words; a starved crafter says what it
// misses.
func (st *MachineState) reason(lang string) string {
	text := i18n.T(lang, st.Mon.Status.reasonKey())
	if st.Mon.Status == StatusStarved && st.Def.Crafter && st.Recipe != nil {
		if missing := CrafterMissing(lang, st); missing != "" {
			text += ": " + missing
		}
	}
	return text
}

// StalledCount is the number of machines on the island's list that are stuck
// or starved.
func StalledCount(s *core.Session) int {
	n := 0
	for _, st := range FactoryOf(s).machines {
		if st.Mon.Status.alerts() {
			n++
		}
	}
	return n
}

// TickMonitor runs every second: it judges the machines, sends the alerts
// and passes the production to the players' counters.
func TickMonitor(s *core.Session, tx *world.Tx) {
	FactoryOf(s).now++
	st := FactoryOf(s).Stats
	for k, n := range st.fresh {
		achievement.CountAll(tx, k, n)
		delete(st.fresh, k)
	}
	working, newAlerts := 0, 0
	var alerts []*alertLine
	for pos, ms := range FactoryOf(s).machines {
		if FactoryOf(s).lastTick-ms.Mon.seen > machineStale*20 {
			if FactoryOf(s).lastTick-ms.Mon.seen > machineForget*20 {
				delete(FactoryOf(s).machines, pos)
			}
			continue
		}
		cur := ms.judge(FactoryOf(s).now)
		m := &ms.Mon
		switch {
		case cur == StatusWorking:
			working++
			if m.Alerted {
				m.Alerted = false
				achievement.CountAll(tx, "stall_fixed", 1)
			}
		case cur.alerts() && !m.Alerted && FactoryOf(s).now-m.since >= AlertAfter && (m.LastAlert == 0 || FactoryOf(s).now-m.LastAlert >= alertSpacing):
			m.Alerted, m.LastAlert = true, FactoryOf(s).now
			alerts = append(alerts, &alertLine{pos: pos, State: ms})
		case !cur.alerts() && m.Alerted:
			m.Alerted = false
		}
	}
	for _, ms := range FactoryOf(s).machines {
		if ms.Mon.Alerted {
			newAlerts++
		}
	}
	sendAlerts(s, tx, alerts)
	tickRecords(s, tx, working, newAlerts, len(alerts) > 0)
	if st.Secs++; st.Secs >= 60 {
		done := st.Rotate()
		minuteRecords(s, tx, done)
	}
}

// alertLine is one machine to alert.
type alertLine struct {
	pos   cube.Pos
	State *MachineState
}

// sendAlerts tells the players with alerts on about stuck machines: a line
// each for the first few, the rest together.
func sendAlerts(s *core.Session, tx *world.Tx, alerts []*alertLine) {
	if len(alerts) == 0 {
		return
	}
	sort.Slice(alerts, func(i, j int) bool { return alerts[i].State.Mon.since < alerts[j].State.Mon.since })
	for e := range tx.Players() {
		p := e.(*player.Player)
		if !FactorySaved.Of(s).AlertsOn[instance.AccountID(p)] {
			continue
		}
		lang := i18n.Of(p)
		for i, a := range alerts {
			if i == alertLines {
				p.Message(fmt.Sprintf(i18n.T(lang, "monitor.alert.more"), len(alerts)-alertLines))
				break
			}
			p.Message(fmt.Sprintf(i18n.T(lang, "monitor.alert"), a.State.Def.Title(lang), a.pos.X(), a.pos.Y(), a.pos.Z(), i18n.T(lang, a.State.Mon.Status.reasonKey())))
		}
		p.PlaySound(sound.Note{Instrument: sound.Bass(), Pitch: 6})
		achievement.Count(p, "alerts_seen", int64(len(alerts)))
	}
}

// Uptime records, in minutes without a stall alert while enough machines work.
var uptimeRecords = []int{30, 60, 180}

// tickRecords runs the per-second records that are not about a minute: the
// time the factory ran without a stall.
func tickRecords(s *core.Session, tx *world.Tx, working, alerted int, alerting bool) {
	if working >= uptimeMachines && alerted == 0 && !alerting {
		FactoryOf(s).uptime++
	} else {
		FactoryOf(s).uptime = 0
	}
	for _, m := range uptimeRecords {
		if FactoryOf(s).uptime == m*60 {
			achievement.CountAll(tx, fmt.Sprintf("uptime_%d", m), 1)
		}
	}
}

// Production records per minute and the power records in RE/t.
var (
	rateRecords  = []int{60, 300, 1000, 3000, 10000}
	powerRecords = []int{100, 500, 2000, 10000}
)

// minuteRecords passes the records of a finished minute to the players.
func minuteRecords(s *core.Session, tx *world.Tx, b statBucket) {
	for _, r := range rateRecords {
		if b.Total >= r {
			achievement.CountAll(tx, fmt.Sprintf("rate_%d", r), 1)
		}
	}
	avg := float64(b.gen) / 1200
	for _, r := range powerRecords {
		if avg >= float64(r) {
			achievement.CountAll(tx, fmt.Sprintf("power_%d", r), 1)
		}
	}
}

// StallOptions are the page of the stuck machines: a button each that takes
// the player there, the alert switch above.
func StallOptions(s *core.Session, p *player.Player) (string, []ui.Option) {
	lang := i18n.Of(p)
	id := instance.AccountID(p)
	state := "monitor.alerts.off"
	if FactorySaved.Of(s).AlertsOn[id] {
		state = "monitor.alerts.on"
	}
	var list []*MachineState
	for _, st := range FactoryOf(s).machines {
		if st.Mon.Status.alerts() {
			list = append(list, st)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Mon.since != list[j].Mon.since {
			return list[i].Mon.since < list[j].Mon.since
		}
		return list[i].pos.X() < list[j].pos.X()
	})
	body := i18n.T(lang, state)
	if len(list) == 0 {
		body += "\n" + i18n.T(lang, "monitor.stalls.none")
	}
	opts := []ui.Option{s.Opt(i18n.T(lang, "monitor.alerts.toggle"), "textures/ui/icon_bell", func(p *player.Player, _ *world.Tx) {
		if FactorySaved.Of(s).AlertsOn == nil {
			FactorySaved.Of(s).AlertsOn = map[string]bool{}
		}
		if FactorySaved.Of(s).AlertsOn[id] {
			delete(FactorySaved.Of(s).AlertsOn, id)
		} else {
			FactorySaved.Of(s).AlertsOn[id] = true
		}
		s.MarkDirty()
		showStatsPage(s, p, StatsStalls)
	})}
	for i, st := range list {
		if i >= 40 {
			break
		}
		pos := st.pos
		secs := FactoryOf(s).now - st.Mon.since
		label := fmt.Sprintf("%s (%d, %d, %d)\n§7%s · %s", st.Def.Title(lang), pos.X(), pos.Y(), pos.Z(), st.reason(lang), textfmt.Span(lang, secs))
		opts = append(opts, s.Opt(label, "", func(p *player.Player, tx *world.Tx) { goToMachine(s, p, tx, pos) }))
	}
	return body, opts
}

// goToMachine moves p next to the machine at pos: a free spot of two blocks
// high around it, inside the island.
func goToMachine(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	for _, d := range []cube.Pos{{0, 1, 0}, {1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {1, 1, 0}, {-1, 1, 0}, {0, 1, 1}, {0, 1, -1}} {
		at := pos.Add(d)
		if core.SolidAt(tx, at) || core.SolidAt(tx, at.Add(cube.Pos{0, 1, 0})) {
			continue
		}
		if !island.Owns(s, island.ChunkPos{int32(at.X() >> 4), int32(at.Z() >> 4)}) {
			continue
		}
		from := mgl64.Vec3{float64(at.X()) + 0.5, float64(at.Y()), float64(at.Z()) + 0.5}
		p.Teleport(from)
		faceTowards(p, from.Add(mgl64.Vec3{0, 1.62, 0}), pos.Vec3Centre())
		return
	}
}

// faceTowards turns p, standing at from (its eyes), to look at target.
func faceTowards(p *player.Player, eyes, target mgl64.Vec3) {
	d := target.Sub(eyes)
	yaw := -math.Atan2(d[0], d[2]) * 180 / math.Pi
	pitch := -math.Atan2(d[1], math.Hypot(d[0], d[2])) * 180 / math.Pi
	cur := p.Rotation()
	p.Move(mgl64.Vec3{}, yaw-cur.Yaw(), pitch-cur.Pitch())
}

// factory is what a save knows of its machines while it runs: the
// production record, the machines that tick by position, the seconds the
// island ran (now), the last machine tick seen and the seconds without a
// stall (uptime).
type factory struct {
	Stats    *prodStats
	machines map[cube.Pos]*MachineState
	now      int
	lastTick int64
	uptime   int
}

var factoryPart = core.NewPart(func() *factory {
	return &factory{Stats: NewProdStats(), machines: map[cube.Pos]*MachineState{}}
})

// FactoryOf is what the save of s knows of its machines.
func FactoryOf(s *core.Session) *factory { return factoryPart.Of(s) }
