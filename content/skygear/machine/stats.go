package machine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// The production record of an island:
// what its machines put in their output slots and the energy they make and
// use. The last statMinutes minutes are kept in one-minute buckets; the
// totals are saved in the state.

// StatMinutes is how many one-minute buckets are kept.
const StatMinutes = 15

// statBucket is one minute of production.
type statBucket struct {
	items map[string]int
	Total int
	// gen and use are the RE made and used.
	gen, Use int
}

// prodStats is the production record of a session, owned by its world.
type prodStats struct {
	Buckets [StatMinutes]statBucket
	// head is the bucket of the minute now; secs the seconds it has run;
	// filled the minutes finished (up to statMinutes).
	Head, Secs, filled int
	// samples are a stack of each item key made, for the screens.
	samples map[string]item.Stack
	// fresh are the items made since the last second, by counter key, passed
	// to the players' counters once a second.
	fresh map[string]int64
}

func NewProdStats() *prodStats {
	p := &prodStats{samples: map[string]item.Stack{}, fresh: map[string]int64{}}
	p.Buckets[0].items = map[string]int{}
	return p
}

// madeCategory is the counter an item made by a machine counts under ("" for
// none besides the total).
func madeCategory(it world.Item) string {
	switch it.(type) {
	case sgitem.Ingot, item.IronIngot, item.GoldIngot, item.CopperIngot, item.NetheriteIngot:
		return "made_ingot"
	case sgitem.Dust, sgitem.MetalDust:
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
func (p *prodStats) Add(state *FactoryState, st item.Stack, n int, counted bool) {
	if n <= 0 || st.Empty() {
		return
	}
	key := naming.ItemKey(st.Item())
	b := &p.Buckets[p.Head]
	b.items[key] += n
	b.Total += n
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
func (p *prodStats) Rotate() statBucket {
	done := p.Buckets[p.Head]
	p.Head = (p.Head + 1) % StatMinutes
	p.Buckets[p.Head] = statBucket{items: map[string]int{}}
	p.Secs = 0
	p.filled = min(p.filled+1, StatMinutes-1)
	return done
}

// window sums the minute running and the last n minutes before it, and
// returns the minutes they cover.
func (p *prodStats) Window(n int) (items map[string]int, minutes float64) {
	items = map[string]int{}
	done := min(n, p.filled)
	for i := 0; i <= done; i++ {
		for k, v := range p.Buckets[(p.Head-i+StatMinutes)%StatMinutes].items {
			items[k] += v
		}
	}
	return items, float64(done) + float64(p.Secs)/60
}

// energyWindow sums the RE made and used in the last n minutes.
func (p *prodStats) energyWindow(n int) (gen, use int, ticks float64) {
	for i := 0; i <= min(n, p.filled); i++ {
		b := p.Buckets[(p.Head-i+StatMinutes)%StatMinutes]
		gen += b.gen
		use += b.Use
	}
	_, minutes := p.Window(n)
	return gen, use, minutes * 1200
}

// PerMinute is the average per minute of count items over minutes (0 while
// less than ten seconds are recorded).
func PerMinute(count int, minutes float64) float64 {
	if minutes < 1.0/6 {
		return 0
	}
	return float64(count) / minutes
}

// roomFor reports whether stacks fit in the output; when they do not, the
// machine is stuck on its output.
func (st *State) RoomFor(stacks []item.Stack) bool {
	if Fits(st.Inv(), st.Def.Output, stacks) {
		return true
	}
	st.Stuck = StuckOutput
	return false
}

// emit puts a product in the output and records it as made by the island.
func (st *State) Emit(it item.Stack) {
	put(st.Inv(), st.Def.Output, it)
	if st.Sess != nil {
		FactoryOf(st.Sess).Stats.Add(FactorySaved.Of(st.Sess), it, it.Count(), true)
	}
}

// tickWork spends one tick of energy and advances the work; it reports
// whether the machine could work.
func (st *State) TickWork() bool {
	if !st.consume() {
		return false
	}
	st.Progress = min(st.Progress+1, st.CycleLen())
	st.markWorked()
	return true
}

// recordMade records items a player's work made, such as the hand sieve's:
// the statistics show them, the achievements for what machines make do not
// count them.
func recordMade(s *core.Session, it item.Stack) {
	FactoryOf(s).Stats.Add(FactorySaved.Of(s), it, it.Count(), false)
}

// KeyStack returns a stack of a saved item key.
func KeyStack(s *core.Session, key string) item.Stack {
	if st, ok := FactoryOf(s).Stats.samples[key]; ok {
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
		FactoryOf(s).Stats.samples[key] = st
		return st
	}
	return item.Stack{}
}

// StatsTab is a page of the statistics screen.
type StatsTab int

const (
	StatsItems StatsTab = iota
	StatsPower
	StatsStalls
)

// ShowStats opens the statistics screen on a page.
func ShowStats(s *core.Session, p *player.Player, tab StatsTab) {
	achievement.Count(p, "stats_open", 1)
	showStatsPage(s, p, tab)
}

func statsTabs(s *core.Session, p *player.Player, tab StatsTab) []ui.Option {
	lang := i18n.Of(p)
	// Tabs carry an icon, like the tabs of the creative inventory.
	icons := map[StatsTab]string{
		StatsItems:  formicon.Of(Item(KindCrusher)),
		StatsPower:  formicon.Of(Item(KindGenerator)),
		StatsStalls: formicon.Locked,
	}
	mk := func(label string, t StatsTab) ui.Option {
		return ui.Tab(label, icons[t], tab == t, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.World {
				showStatsPage(s, p, t)
			}
		})
	}
	return []ui.Option{
		mk(i18n.T(lang, "monitor.tab.items"), StatsItems),
		mk(i18n.T(lang, "monitor.tab.power"), StatsPower),
		mk(fmt.Sprintf(i18n.T(lang, "monitor.tab.stalls"), StalledCount(s)), StatsStalls),
		s.BackOption(lang),
	}
}

// showStatsPage shows one page without counting an opening.
func showStatsPage(s *core.Session, p *player.Player, tab StatsTab) {
	lang := i18n.Of(p)
	title := i18n.T(lang, "monitor.title")
	switch tab {
	case StatsPower:
		ui.Menu(p, title, powerText(s, lang), statsTabs(s, p, tab)...)
	case StatsStalls:
		body, opts := StallOptions(s, p)
		ui.Menu(p, title, body, append(opts, statsTabs(s, p, tab)...)...)
	default:
		body, slots := itemSlots(s, lang)
		ui.ItemGrid(p, title, body, slots, statsTabs(s, p, tab)...)
	}
}

// itemSlots are the item slots of the production page and its heading.
func itemSlots(s *core.Session, lang string) (string, []ui.ItemSlot) {
	m1, t1 := FactoryOf(s).Stats.Window(1)
	m5, t5 := FactoryOf(s).Stats.Window(5)
	m15, t15 := FactoryOf(s).Stats.Window(15)
	type row struct {
		key         string
		r1, r5, r15 float64
		total       int64
	}
	keys := map[string]bool{}
	for k := range FactorySaved.Of(s).Produced {
		keys[k] = true
	}
	var rows []row
	var sum float64
	for k := range keys {
		r := row{key: k, r1: PerMinute(m1[k], t1), r5: PerMinute(m5[k], t5), r15: PerMinute(m15[k], t15), total: FactorySaved.Of(s).Produced[k]}
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
	body := fmt.Sprintf(i18n.T(lang, "monitor.items.head"), textfmt.PerMin(sum), len(rows))
	var slots []ui.ItemSlot
	for _, r := range rows {
		st := KeyStack(s, r.key)
		if st.Empty() || len(slots) >= 99 {
			continue
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  formicon.Of(st),
			Count: ui.CountText(int(r.r5 + 0.5)),
			Tip:   fmt.Sprintf(i18n.T(lang, "monitor.items.tip"), naming.UIItemName(lang, st), textfmt.PerMin(r.r5), textfmt.PerMin(r.r1), textfmt.PerMin(r.r15), textfmt.Int(r.total)),
		})
	}
	return body, slots
}

// powerText is the energy page.
func powerText(s *core.Session, lang string) string {
	gen, use, ticks := FactoryOf(s).Stats.energyWindow(5)
	var genRate, useRate float64
	if ticks >= 200 {
		genRate, useRate = float64(gen)/ticks, float64(use)/ticks
	}
	var stored, capacity, generators, running int
	for _, st := range FactoryOf(s).machines {
		switch {
		case st.Def.Cell:
			stored += st.Energy
			capacity += st.EnergyCap()
		case st.Def.Generates > 0:
			generators++
			if st.Mon.Status == StatusWorking {
				running++
			}
		}
	}
	text := fmt.Sprintf(i18n.T(lang, "monitor.power.body"), genRate, useRate, textfmt.Int(int64(stored)), textfmt.Int(int64(capacity)), generators, running)
	if genRate+0.05 < useRate {
		text += "\n" + i18n.T(lang, "monitor.power.low")
	}
	return text
}

// markWorked notes that the machine did its work this tick.
func (st *State) markWorked() {
	st.didWork = true
	st.worked++
}

// FactoryState is what a save keeps of its machines: the stall alerts
// settings and the production totals.
type FactoryState struct {
	// Produced counts the items the machines made, by item key.
	Produced map[string]int64 `json:"produced,omitempty"`
	// AlertsOn are the accounts that turned the stall alerts on; they are
	// off until then (monitor.go).
	AlertsOn map[string]bool `json:"alerts_on,omitempty"`
}

var FactorySaved = core.NewSaved[FactoryState](nil)
