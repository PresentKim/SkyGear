package dragonfactory

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The production record of an island (docs/design/12-production-monitor.md):
// what its machines put in their output slots and the energy they make and
// use. The last statMinutes minutes are kept in one-minute buckets; the
// totals are saved in the state.

// statMinutes is how many one-minute buckets are kept.
const statMinutes = 15

// statBucket is one minute of production.
type statBucket struct {
	items map[string]int
	total int
	// gen and use are the RE made and used.
	gen, use int
}

// prodStats is the production record of a session, owned by its world.
type prodStats struct {
	buckets [statMinutes]statBucket
	// head is the bucket of the minute now; secs the seconds it has run;
	// filled the minutes finished (up to statMinutes).
	head, secs, filled int
	// samples are a stack of each item key made, for the screens.
	samples map[string]item.Stack
	// fresh are the items made since the last second, by counter key, passed
	// to the players' counters once a second.
	fresh map[string]int64
}

func newProdStats() *prodStats {
	p := &prodStats{samples: map[string]item.Stack{}, fresh: map[string]int64{}}
	p.buckets[0].items = map[string]int{}
	return p
}

// madeCategory is the counter an item made by a machine counts under ("" for
// none besides the total).
func madeCategory(it world.Item) string {
	switch it.(type) {
	case Ingot, item.IronIngot, item.GoldIngot, item.CopperIngot, item.NetheriteIngot:
		return "made_ingot"
	case Dust, MetalDust:
		return "made_dust"
	case item.Diamond, item.Emerald, item.NetherQuartz, item.LapisLazuli, item.AmethystShard:
		return "made_gem"
	}
	switch it.(type) {
	case block.Cobblestone, block.Gravel, block.Sand, block.Dirt:
		return "made_block"
	}
	name, _ := it.EncodeItem()
	if strings.HasSuffix(name, "_log") || strings.HasSuffix(name, "_sapling") {
		return "made_wood"
	}
	return ""
}

// add records n items made.
func (p *prodStats) add(state *State, st item.Stack, n int, counted bool) {
	if n <= 0 || st.Empty() {
		return
	}
	key := itemKey(st.Item())
	b := &p.buckets[p.head]
	b.items[key] += n
	b.total += n
	if _, ok := p.samples[key]; !ok {
		p.samples[key] = item.NewStack(st.Item(), 1)
	}
	if state.Produced == nil {
		state.Produced = map[string]int64{}
	}
	state.Produced[key] += int64(n)
	if !counted {
		return
	}
	p.fresh["made_items"] += int64(n)
	if c := madeCategory(st.Item()); c != "" {
		p.fresh[c] += int64(n)
	}
}

// rotate closes the minute that ran out and returns its bucket.
func (p *prodStats) rotate() statBucket {
	done := p.buckets[p.head]
	p.head = (p.head + 1) % statMinutes
	p.buckets[p.head] = statBucket{items: map[string]int{}}
	p.secs = 0
	p.filled = min(p.filled+1, statMinutes-1)
	return done
}

// window sums the minute running and the last n minutes before it, and
// returns the minutes they cover.
func (p *prodStats) window(n int) (items map[string]int, minutes float64) {
	items = map[string]int{}
	done := min(n, p.filled)
	for i := 0; i <= done; i++ {
		for k, v := range p.buckets[(p.head-i+statMinutes)%statMinutes].items {
			items[k] += v
		}
	}
	return items, float64(done) + float64(p.secs)/60
}

// energyWindow sums the RE made and used in the last n minutes.
func (p *prodStats) energyWindow(n int) (gen, use int, ticks float64) {
	for i := 0; i <= min(n, p.filled); i++ {
		b := p.buckets[(p.head-i+statMinutes)%statMinutes]
		gen += b.gen
		use += b.use
	}
	_, minutes := p.window(n)
	return gen, use, minutes * 1200
}

// perMinute is the average per minute of count items over minutes (0 while
// less than ten seconds are recorded).
func perMinute(count int, minutes float64) float64 {
	if minutes < 1.0/6 {
		return 0
	}
	return float64(count) / minutes
}

// roomFor reports whether stacks fit in the output; when they do not, the
// machine is stuck on its output.
func (st *machineState) roomFor(stacks []item.Stack) bool {
	if fits(st.inv(), st.kind.output, stacks) {
		return true
	}
	st.stuck = stuckOutput
	return false
}

// emit puts a product in the output and records it as made by the island.
func (st *machineState) emit(it item.Stack) {
	put(st.inv(), st.kind.output, it)
	if st.sess != nil {
		st.sess.stats.add(st.sess.state, it, it.Count(), true)
	}
}

// tickWork spends one tick of energy and advances the work; it reports
// whether the machine could work.
func (st *machineState) tickWork() bool {
	if !st.consume() {
		return false
	}
	st.progress = min(st.progress+1, st.cycleLen())
	st.markWorked()
	return true
}

// recordMade records items a player's work made, such as the hand sieve's:
// the statistics show them, the achievements for what machines make do not
// count them.
func (s *session) recordMade(it item.Stack) {
	s.initMonitor()
	s.stats.add(s.state, it, it.Count(), false)
}

// keyStack returns a stack of a saved item key.
func (s *session) keyStack(key string) item.Stack {
	if st, ok := s.stats.samples[key]; ok {
		return st
	}
	i := strings.LastIndex(key, ":")
	if i < 0 {
		return item.Stack{}
	}
	meta, err := strconv.Atoi(key[i+1:])
	if err != nil {
		return item.Stack{}
	}
	if it, ok := world.ItemByName(key[:i], int16(meta)); ok {
		st := item.NewStack(it, 1)
		s.stats.samples[key] = st
		return st
	}
	return item.Stack{}
}

// statsTab is a page of the statistics screen.
type statsTab int

const (
	statsItems statsTab = iota
	statsPower
	statsStalls
)

// showStats opens the statistics screen on a page.
func (s *session) showStats(p *player.Player, tab statsTab) {
	s.initMonitor()
	count(p, "stats_open", 1)
	s.showStatsPage(p, tab)
}

func (s *session) statsTabs(p *player.Player, tab statsTab) []ui.Option {
	lang := i18n.Of(p)
	mk := func(label string, t statsTab) ui.Option {
		return ui.Tab(label, "", tab == t, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showStatsPage(p, t)
			}
		})
	}
	return []ui.Option{
		mk(i18n.T(lang, "monitor.tab.items"), statsItems),
		mk(i18n.T(lang, "monitor.tab.power"), statsPower),
		mk(fmt.Sprintf(i18n.T(lang, "monitor.tab.stalls"), s.stalledCount()), statsStalls),
		s.backOption(),
	}
}

// showStatsPage shows one page without counting an opening.
func (s *session) showStatsPage(p *player.Player, tab statsTab) {
	lang := i18n.Of(p)
	title := i18n.T(lang, "monitor.title")
	switch tab {
	case statsPower:
		ui.Menu(p, title, s.powerText(lang), s.statsTabs(p, tab)...)
	case statsStalls:
		body, opts := s.stallOptions(p)
		ui.Menu(p, title, body, append(opts, s.statsTabs(p, tab)...)...)
	default:
		body, slots := s.itemSlots(lang)
		ui.ItemGrid(p, title, body, slots, s.statsTabs(p, tab)...)
	}
}

// itemSlots are the item slots of the production page and its heading.
func (s *session) itemSlots(lang string) (string, []ui.ItemSlot) {
	m1, t1 := s.stats.window(1)
	m5, t5 := s.stats.window(5)
	m15, t15 := s.stats.window(15)
	type row struct {
		key         string
		r1, r5, r15 float64
		total       int64
	}
	keys := map[string]bool{}
	for k := range s.state.Produced {
		keys[k] = true
	}
	var rows []row
	var sum float64
	for k := range keys {
		r := row{key: k, r1: perMinute(m1[k], t1), r5: perMinute(m5[k], t5), r15: perMinute(m15[k], t15), total: s.state.Produced[k]}
		sum += r.r5
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].r5 != rows[j].r5 {
			return rows[i].r5 > rows[j].r5
		}
		if rows[i].total != rows[j].total {
			return rows[i].total > rows[j].total
		}
		return rows[i].key < rows[j].key
	})
	body := fmt.Sprintf(i18n.T(lang, "monitor.items.head"), formatPerMin(sum), len(rows))
	var slots []ui.ItemSlot
	for _, r := range rows {
		st := s.keyStack(r.key)
		if st.Empty() || len(slots) >= 99 {
			continue
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  iconOf(st),
			Count: ui.CountText(int(r.r5 + 0.5)),
			Tip:   fmt.Sprintf(i18n.T(lang, "monitor.items.tip"), itemName(st), formatPerMin(r.r5), formatPerMin(r.r1), formatPerMin(r.r15), formatInt(r.total)),
		})
	}
	return body, slots
}

// formatPerMin writes a per-minute amount: whole numbers from ten up, one
// decimal below.
func formatPerMin(v float64) string {
	if v >= 10 {
		return formatInt(int64(v + 0.5))
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// powerText is the energy page.
func (s *session) powerText(lang string) string {
	gen, use, ticks := s.stats.energyWindow(5)
	var genRate, useRate float64
	if ticks >= 200 {
		genRate, useRate = float64(gen)/ticks, float64(use)/ticks
	}
	var stored, capacity, generators, running int
	for _, st := range s.machines {
		switch {
		case st.kind.cell:
			stored += st.energy
			capacity += st.energyCap()
		case st.kind.generates > 0:
			generators++
			if st.mon.status == statusWorking {
				running++
			}
		}
	}
	text := fmt.Sprintf(i18n.T(lang, "monitor.power.body"), genRate, useRate, formatInt(int64(stored)), formatInt(int64(capacity)), generators, running)
	if genRate+0.05 < useRate {
		text += "\n" + i18n.T(lang, "monitor.power.low")
	}
	return text
}

// markWorked notes that the machine did its work this tick.
func (st *machineState) markWorked() {
	st.didWork = true
	st.worked++
}
