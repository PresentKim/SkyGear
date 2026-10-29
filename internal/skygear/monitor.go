package skygear

import (
	"fmt"
	"math"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/instance"
	"github.com/presentkim/skygear/internal/skygear/advancement"
	"github.com/presentkim/skygear/internal/skygear/textfmt"
)

// The machine list and the stall alerts: every machine that ticks is on the
// island's list; once a second the list is walked to judge what each machine
// is doing.

// machineStatus is what a machine is doing.
type machineStatus int

const (
	statusIdle machineStatus = iota
	statusWorking
	statusStarved
	statusPower
	statusOutput
	statusFuel
)

// stuckKind is why a machine could not work in its last tick (0: nothing is
// wrong).
type stuckKind int

const (
	stuckNone stuckKind = iota
	stuckPower
	stuckOutput
	stuckFuel
)

// Timings of the alerts, in seconds.
const (
	alertAfter     = 30
	alertSpacing   = 300
	starvedWindow  = 300
	alertLines     = 3
	machineStale   = 10
	machineForget  = 300
	uptimeMachines = 10
)

// monitor is the state of a machine for the stall alerts.
type monitor struct {
	status machineStatus
	// since is when the status began, lastWork when the machine last
	// worked, lastAlert when it was last alerted (seconds of the island's
	// clock; 0: never).
	since, lastWork, lastAlert int
	alerted                    bool
	// seen is the tick it last ran.
	seen int64
}

func (s machineStatus) alerts() bool {
	return s == statusPower || s == statusOutput || s == statusFuel
}

// reasonKey is the text key of a status.
func (s machineStatus) reasonKey() string {
	switch s {
	case statusWorking:
		return "monitor.reason.working"
	case statusStarved:
		return "monitor.reason.starved"
	case statusPower:
		return "monitor.reason.power"
	case statusOutput:
		return "monitor.reason.output"
	case statusFuel:
		return "monitor.reason.fuel"
	}
	return "monitor.reason.idle"
}

// judge decides what the machine did in the last second.
func (st *machineState) judge(now int) machineStatus {
	m := &st.mon
	var cur machineStatus
	switch {
	case st.didWork:
		cur = statusWorking
		m.lastWork = now
	case st.stuck == stuckPower:
		cur = statusPower
	case st.stuck == stuckOutput:
		cur = statusOutput
	case st.stuck == stuckFuel && m.lastWork > 0:
		cur = statusFuel
	case m.lastWork > 0 && now-m.lastWork < starvedWindow:
		cur = statusStarved
	}
	st.didWork = false
	if cur != m.status {
		m.status, m.since = cur, now
	}
	return cur
}

// reason is the status of a machine in words; a starved crafter says what it
// misses.
func (st *machineState) reason(lang string) string {
	text := i18n.T(lang, st.mon.status.reasonKey())
	if st.mon.status == statusStarved && st.kind.crafter && st.recipe != nil {
		if missing := crafterMissing(lang, st); missing != "" {
			text += ": " + missing
		}
	}
	return text
}

// stalledCount is the number of machines on the island's list that are stuck
// or starved.
func (s *session) stalledCount() int {
	n := 0
	for _, st := range s.machines {
		if st.mon.status.alerts() {
			n++
		}
	}
	return n
}

// tickMonitor runs every second: it judges the machines, sends the alerts
// and passes the production to the players' counters.
func (s *session) tickMonitor(tx *world.Tx) {
	s.initMonitor()
	s.now++
	st := s.stats
	for k, n := range st.fresh {
		countAll(tx, k, n)
		delete(st.fresh, k)
	}
	working, newAlerts := 0, 0
	var alerts []*alertLine
	for pos, ms := range s.machines {
		if s.lastTick-ms.mon.seen > machineStale*20 {
			if s.lastTick-ms.mon.seen > machineForget*20 {
				delete(s.machines, pos)
			}
			continue
		}
		cur := ms.judge(s.now)
		m := &ms.mon
		switch {
		case cur == statusWorking:
			working++
			if m.alerted {
				m.alerted = false
				countAll(tx, "stall_fixed", 1)
			}
		case cur.alerts() && !m.alerted && s.now-m.since >= alertAfter && (m.lastAlert == 0 || s.now-m.lastAlert >= alertSpacing):
			m.alerted, m.lastAlert = true, s.now
			alerts = append(alerts, &alertLine{pos: pos, st: ms})
		case !cur.alerts() && m.alerted:
			m.alerted = false
		}
	}
	for _, ms := range s.machines {
		if ms.mon.alerted {
			newAlerts++
		}
	}
	s.sendAlerts(tx, alerts)
	s.tickRecords(tx, working, newAlerts, len(alerts) > 0)
	if st.secs++; st.secs >= 60 {
		done := st.rotate()
		s.minuteRecords(tx, done)
	}
}

// initMonitor sets up the record and the machine list of a session built
// without them.
func (s *session) initMonitor() {
	if s.stats == nil {
		s.stats = newProdStats()
	}
	if s.machines == nil {
		s.machines = map[cube.Pos]*machineState{}
	}
}

// alertLine is one machine to alert.
type alertLine struct {
	pos cube.Pos
	st  *machineState
}

// sendAlerts tells the players with alerts on about stuck machines: a line
// each for the first few, the rest together.
func (s *session) sendAlerts(tx *world.Tx, alerts []*alertLine) {
	if len(alerts) == 0 {
		return
	}
	sort.Slice(alerts, func(i, j int) bool { return alerts[i].st.mon.since < alerts[j].st.mon.since })
	for e := range tx.Players() {
		p := e.(*player.Player)
		if !s.state.AlertsOn[instance.AccountID(p)] {
			continue
		}
		lang := i18n.Of(p)
		for i, a := range alerts {
			if i == alertLines {
				p.Message(fmt.Sprintf(i18n.T(lang, "monitor.alert.more"), len(alerts)-alertLines))
				break
			}
			p.Message(fmt.Sprintf(i18n.T(lang, "monitor.alert"), a.st.kind.title(lang), a.pos.X(), a.pos.Y(), a.pos.Z(), i18n.T(lang, a.st.mon.status.reasonKey())))
		}
		p.PlaySound(sound.Note{Instrument: sound.Bass(), Pitch: 6})
		count(p, "alerts_seen", int64(len(alerts)))
	}
}

// Uptime records, in minutes without a stall alert while enough machines work.
var uptimeRecords = []int{30, 60, 180}

// tickRecords runs the per-second records that are not about a minute: the
// time the factory ran without a stall.
func (s *session) tickRecords(tx *world.Tx, working, alerted int, alerting bool) {
	if working >= uptimeMachines && alerted == 0 && !alerting {
		s.uptime++
	} else {
		s.uptime = 0
	}
	for _, m := range uptimeRecords {
		if s.uptime == m*60 {
			countAll(tx, fmt.Sprintf("uptime_%d", m), 1)
		}
	}
}

// Production records per minute and the power records in RE/t.
var (
	rateRecords  = []int{60, 300, 1000, 3000, 10000}
	powerRecords = []int{100, 500, 2000, 10000}
)

// minuteRecords passes the records of a finished minute to the players.
func (s *session) minuteRecords(tx *world.Tx, b statBucket) {
	for _, r := range rateRecords {
		if b.total >= r {
			countAll(tx, fmt.Sprintf("rate_%d", r), 1)
		}
	}
	avg := float64(b.gen) / 1200
	for _, r := range powerRecords {
		if avg >= float64(r) {
			countAll(tx, fmt.Sprintf("power_%d", r), 1)
		}
	}
}

// stallOptions are the page of the stuck machines: a button each that takes
// the player there, the alert switch above.
func (s *session) stallOptions(p *player.Player) (string, []ui.Option) {
	lang := i18n.Of(p)
	id := instance.AccountID(p)
	state := "monitor.alerts.off"
	if s.state.AlertsOn[id] {
		state = "monitor.alerts.on"
	}
	var list []*machineState
	for _, st := range s.machines {
		if st.mon.status.alerts() {
			list = append(list, st)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].mon.since != list[j].mon.since {
			return list[i].mon.since < list[j].mon.since
		}
		return list[i].pos.X() < list[j].pos.X()
	})
	body := i18n.T(lang, state)
	if len(list) == 0 {
		body += "\n" + i18n.T(lang, "monitor.stalls.none")
	}
	opts := []ui.Option{s.opt(i18n.T(lang, "monitor.alerts.toggle"), "textures/ui/icon_bell", func(p *player.Player, _ *world.Tx) {
		if s.state.AlertsOn == nil {
			s.state.AlertsOn = map[string]bool{}
		}
		if s.state.AlertsOn[id] {
			delete(s.state.AlertsOn, id)
		} else {
			s.state.AlertsOn[id] = true
		}
		s.storeState()
		s.showStatsPage(p, statsStalls)
	})}
	for i, st := range list {
		if i >= 40 {
			break
		}
		pos := st.pos
		secs := s.now - st.mon.since
		label := fmt.Sprintf("%s (%d, %d, %d)\n§7%s · %s", st.kind.title(lang), pos.X(), pos.Y(), pos.Z(), st.reason(lang), textfmt.Span(lang, secs))
		opts = append(opts, s.opt(label, "", func(p *player.Player, tx *world.Tx) { s.goToMachine(p, tx, pos) }))
	}
	return body, opts
}

// goToMachine moves p next to the machine at pos: a free spot of two blocks
// high around it, inside the island.
func (s *session) goToMachine(p *player.Player, tx *world.Tx, pos cube.Pos) {
	for _, d := range []cube.Pos{{0, 1, 0}, {1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {1, 1, 0}, {-1, 1, 0}, {0, 1, 1}, {0, 1, -1}} {
		at := pos.Add(d)
		if solidAt(tx, at) || solidAt(tx, at.Add(cube.Pos{0, 1, 0})) {
			continue
		}
		if !s.owns(ChunkPos{int32(at.X() >> 4), int32(at.Z() >> 4)}) {
			continue
		}
		from := mgl64.Vec3{float64(at.X()) + 0.5, float64(at.Y()), float64(at.Z()) + 0.5}
		p.Teleport(from)
		faceTowards(p, from.Add(mgl64.Vec3{0, 1.62, 0}), pos.Vec3Centre())
		return
	}
}

// statsCmd is /통계 (and /stats): the production statistics.
type statsCmd struct{ g *Game }

func (c statsCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	s.advance(tx, advancement.Stats)
	s.showStats(p, statsItems)
}

// faceTowards turns p, standing at from (its eyes), to look at target.
func faceTowards(p *player.Player, eyes, target mgl64.Vec3) {
	d := target.Sub(eyes)
	yaw := -math.Atan2(d[0], d[2]) * 180 / math.Pi
	pitch := -math.Atan2(d[1], math.Hypot(d[0], d[2])) * 180 / math.Pi
	cur := p.Rotation()
	p.Move(mgl64.Vec3{}, yaw-cur.Yaw(), pitch-cur.Pitch())
}
