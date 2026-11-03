package dimension

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The essence collectors: what the island hunts well keeps coming, a little
// of it. Every cleared monster fills a pool of its essences by its best time
// (a share of what hunting at that pace gives), whether or not a collector
// takes them; every collectEvery the pool hands its whole
// essences out one by one to the collectors that receive that monster, in
// turn, skipping full ones. More collectors of a monster split its essences;
// they do not make more. Values are drafts.

const (
	// collectShare is the share of the essences a minute of hunting at the
	// best run's pace gives (kills a minute times the kill chance) that the
	// pool gets a minute: hunting gives many, the collectors a trickle.
	collectShare = 0.1
	// collectEvery is how often (seconds) the pools hand out essences.
	collectEvery = 15
	// poolMax is the most essences a pool keeps waiting.
	poolMax = 64
)

// PerMinute is how many essences of mob the island's pool gets a minute: 0
// before its first clear.
func PerMinute(s *core.Session, mob string) float64 {
	rec := RecordOf(s, mob)
	h, ok := huntOf(mob)
	if !ok || rec.Best <= 0 {
		return 0
	}
	return float64(h.Count) / (float64(rec.Best) / 60) * killEssence / 100 * collectShare
}

// Pool is how many essences of mob wait in the island's pool.
func Pool(s *core.Session, mob string) float64 { return dimensionSaved.Of(s).Pool[mob] }

// collectLocal is what a running save keeps for its collectors.
type collectLocal struct {
	// seconds counts the seconds to the next hand-out; next is the turn of
	// each monster's collectors.
	seconds int
	next    map[string]int
}

var collectPart = core.NewPart(func() *collectLocal { return &collectLocal{next: map[string]int{}} })

// TickCollectors fills the pools and, every collectEvery seconds, hands them
// out (session.Second).
func TickCollectors(s *core.Session, tx *world.Tx) {
	st := dimensionSaved.Of(s)
	for _, h := range Hunts {
		if per := PerMinute(s, h.Mob); per > 0 {
			if st.Pool == nil {
				st.Pool = map[string]float64{}
			}
			// A pool that is full stays as it is, so the save is not
			// marked changed every second.
			if v := min(poolMax, st.Pool[h.Mob]+per/60); v != st.Pool[h.Mob] {
				st.Pool[h.Mob] = v
				s.MarkDirty()
			}
		}
	}
	l := collectPart.Of(s)
	if l.seconds++; l.seconds < collectEvery {
		return
	}
	l.seconds = 0
	// the old essence bloom harvest counter counts the essences collected
	if n := handOut(s); n > 0 {
		achievement.CountAll(tx, "bloom_ess", int64(n))
	}
}

// handOut gives the pools' whole essences to the collectors, one essence to
// each in turn, and reports how many it gave.
func handOut(s *core.Session) int {
	given := 0
	st := dimensionSaved.Of(s)
	byMob := map[string][]*machine.State{}
	collectors := machine.Ticking(s, machine.KindCollector)
	for _, pos := range slices.SortedFunc(maps.Keys(collectors), comparePos) {
		if c := collectors[pos]; c.Mob != "" {
			byMob[c.Mob] = append(byMob[c.Mob], c)
		}
	}
	l := collectPart.Of(s)
	for mob, cs := range byMob {
		one := []item.Stack{item.NewStack(sgitem.Essence{Mob: mob}, 1)}
		for st.Pool[mob] >= 1 {
			gave := false
			for range cs {
				c := cs[l.next[mob]%len(cs)]
				l.next[mob] = (l.next[mob] + 1) % len(cs)
				if c.RoomFor(one) {
					c.Emit(one[0])
					st.Pool[mob]--
					gave = true
					given++
					break
				}
			}
			if !gave {
				break // every collector of the monster is full
			}
		}
		s.MarkDirty()
	}
	return given
}

// collectorAt is the state of the essence collector at pos.
func collectorAt(tx *world.Tx, pos cube.Pos) (*machine.State, bool) {
	m, ok := tx.Block(pos).(machine.Machine)
	if !ok || m.Kind != machine.KindCollector || m.State == nil {
		return nil, false
	}
	return m.State, true
}

func comparePos(a, b cube.Pos) int {
	return cmp.Or(cmp.Compare(a.X(), b.X()), cmp.Compare(a.Y(), b.Y()), cmp.Compare(a.Z(), b.Z()))
}

// ShowCollector lets p pick the monster the collector at pos receives: the
// cleared ones, with their essences a minute.
func ShowCollector(s *core.Session, p *player.Player, pos cube.Pos) {
	lang := i18n.Of(p)
	st, ok := collectorAt(p.Tx(), pos)
	if !ok {
		return
	}
	var opts []ui.Option
	for _, h := range Hunts {
		per := PerMinute(s, h.Mob)
		if per == 0 {
			continue
		}
		label := fmt.Sprintf(i18n.T(lang, "dimension.collector.row"), MobName(lang, h.Mob), per)
		if st.Mob == h.Mob {
			label = "§a" + label
		}
		opts = append(opts, s.Opt(label, formicon.Of(item.NewStack(sgitem.Essence{Mob: h.Mob}, 1)), func(p *player.Player, tx *world.Tx) {
			if st, ok := collectorAt(tx, pos); ok {
				st.SetMob(h.Mob)
			}
		}))
	}
	body := i18n.T(lang, "dimension.collector.body")
	if len(opts) == 0 {
		body = i18n.T(lang, "dimension.collector.none")
	}
	ui.Menu(p, i18n.T(lang, "machine.collector.name"), body, opts...)
}

// The collector's sneaking click picks its monster; a plain click opens its
// window.
func init() {
	machine.Kinds[machine.KindCollector].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
		if !p.Sneaking() {
			return false
		}
		if !s.Debounced(p) {
			ShowCollector(s, p, pos)
		}
		return true
	}
}
