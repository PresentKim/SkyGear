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
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
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

// collectorOut is the collector's four output slots, which take nothing in.
type collectorOut struct{ st *machine.State }

func (o collectorOut) Len() int { return len(o.st.Def.Output) }
func (o collectorOut) Item(i int) item.Stack {
	st, _ := o.st.Inv().Item(o.st.Def.Output[i])
	return st
}
func (o collectorOut) Set(i int, st item.Stack)     { _ = o.st.Inv().SetItem(o.st.Def.Output[i], st) }
func (collectorOut) Accepts(int, item.Stack) bool { return false }

// ShowCollector is the collector's window: the cleared monsters to pick the
// one it receives (with their essences a minute), its output slots and the
// player's inventory.
func ShowCollector(s *core.Session, p *player.Player, pos cube.Pos) {
	lang := i18n.Of(p)
	st, ok := collectorAt(p.Tx(), pos)
	if !ok {
		return
	}
	var mobs []string
	var mobSlots []ui.InvSlot
	for _, h := range Hunts {
		per := PerMinute(s, h.Mob)
		if per == 0 {
			continue
		}
		sl := ui.InvSlot{Icon: formicon.Of(item.NewStack(sgitem.Essence{Mob: h.Mob}, 1)), Tip: fmt.Sprintf(i18n.T(lang, "dimension.collector.row"), MobName(lang, h.Mob), per)}
		if st.Mob == h.Mob {
			sl.Count, sl.Tip = "§a✔", "§a"+sl.Tip
		}
		mobs = append(mobs, h.Mob)
		mobSlots = append(mobSlots, sl)
	}
	tipOf := func(it item.Stack) string {
		if it.Empty() {
			return ""
		}
		return naming.ItemName(lang, it)
	}
	out := collectorOut{st}
	main, hot := invclick.Main(p.Inventory()), invclick.Hotbar(p.Inventory())
	slotsOf := func(a invclick.Area) []ui.InvSlot {
		sl := make([]ui.InvSlot, a.Len())
		for i := range sl {
			sl[i] = ui.StackSlot(a.Item(i), tipOf(a.Item(i)))
		}
		return sl
	}
	mobRow := ui.InvRow{Heading: i18n.T(lang, "dimension.collector.mobs"), Parts: []ui.InvPart{{Area: "mob", Cols: 9, Slots: mobSlots}}}
	if len(mobSlots) == 0 {
		mobRow.Note = i18n.T(lang, "dimension.collector.none")
	}
	again := func(p *player.Player, tx *world.Tx) { ShowCollector(s, p, pos) }
	ui.ShowInventory(p, ui.InventoryScreen{
		Title: i18n.T(lang, "machine.collector.name"),
		Info:  i18n.T(lang, "dimension.collector.body"),
		Rows: []ui.InvRow{
			mobRow,
			{Heading: i18n.T(lang, "dimension.collector.output"), Parts: []ui.InvPart{{Area: "out", Cols: 4, Slots: slotsOf(out)}}},
			{Heading: i18n.T(lang, "dimension.collector.inventory"), Parts: []ui.InvPart{{Area: "inv", Cols: 9, Slots: slotsOf(main)}}},
			{Parts: []ui.InvPart{{Area: "hotbar", Cols: 9, Slots: slotsOf(hot)}}},
		},
		Slot: func(p *player.Player, tx *world.Tx, a ui.SlotAct) {
			cur, ok := collectorAt(tx, pos)
			if !ok || cur != st || tx.World() != s.World {
				return
			}
			areas := map[string]invclick.Area{"out": out, "inv": main, "hotbar": hot}
			switch area, ok := areas[a.Area]; {
			case a.Area == "mob" && a.Index < len(mobs):
				st.SetMob(mobs[a.Index])
			case !ok || a.Index >= area.Len():
			case a.Shift && a.Area == "out":
				area.Set(a.Index, invclick.Move(area.Item(a.Index), hot, main))
			case a.Shift:
			default:
				ui.SetCarried(p, invclick.Click(area, a.Index, a.Carried, a.Right))
			}
			again(p, tx)
		},
		Again: again,
		Size:  "collector",
	})
}

// A click opens the collector's window.
func init() {
	machine.Kinds[machine.KindCollector].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
		if !s.Debounced(p) {
			ShowCollector(s, p, pos)
		}
		return true
	}
}
