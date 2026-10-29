package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/platform/instance"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// runSeconds runs the monitor's second n times.
func runSeconds(s *session, n int) {
	for range n {
		s.World.Do(func(tx *world.Tx) { machine.TickMonitor(s.Session, tx) })
	}
}

// TestProductionRecorded: what a machine puts in its output is recorded in the
// window and the saved totals.
func TestProductionRecorded(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 3))
	tick(s, 3*(machine.Kinds[machine.KindFurnace].Interval+2))
	made := machine.OutputCount(st, block.Stone{})
	if made == 0 {
		t.Fatal("nothing was smelted")
	}
	key := naming.ItemKey(block.Stone{})
	if got := machine.FactorySaved.Of(s.Session).Produced[key]; got != int64(made) {
		t.Fatalf("saved total %d, want %d", got, made)
	}
	items, _ := machine.FactoryOf(s.Session).Stats.Window(1)
	if items[key] != made {
		t.Fatalf("window has %d, want %d", items[key], made)
	}
	if got := machine.KeyStack(s.Session, key); got.Empty() || naming.ItemKey(got.Item()) != key {
		t.Fatalf("key %q does not give its item back: %v", key, got)
	}
}

// TestMinuteWindows: the minutes rotate and the windows cover the right time.
func TestMinuteWindows(t *testing.T) {
	p := machine.NewProdStats()
	state := &machine.FactoryState{}
	p.Add(state, item.NewStack(block.Cobblestone{}, 1), 60, true)
	for range 60 {
		p.Secs++
	}
	p.Rotate()
	p.Add(state, item.NewStack(block.Cobblestone{}, 1), 30, true)
	p.Secs = 30
	items, minutes := p.Window(5)
	key := naming.ItemKey(block.Cobblestone{})
	if items[key] != 90 || minutes != 1.5 {
		t.Fatalf("5 minute window: %d items over %v minutes", items[key], minutes)
	}
	if got := machine.PerMinute(items[key], minutes); got != 60 {
		t.Fatalf("rate %v, want 60", got)
	}
	for range machine.StatMinutes + 3 {
		p.Rotate()
	}
	if items, _ := p.Window(15); len(items) != 0 {
		t.Fatalf("old minutes were not forgotten: %v", items)
	}
	if state.Produced[key] != 90 {
		t.Fatalf("saved total %d, want 90", state.Produced[key])
	}
}

// TestStallAlert: a machine that cannot put its product away is alerted after
// thirty seconds, once, and its recovery is counted.
func TestStallAlert(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 64))
	for _, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, 64))
	}
	// A worked-before furnace: it finished one, then the output filled.
	tick(s, machine.Kinds[machine.KindFurnace].Interval+5)
	runSeconds(s, 1)
	tick(s, 5)
	runSeconds(s, 1)
	if st.Mon.Status != machine.StatusOutput {
		t.Fatalf("status %v, want output blocked", st.Mon.Status)
	}
	runSeconds(s, machine.AlertAfter-2)
	if st.Mon.Alerted {
		t.Fatal("alerted too early")
	}
	runSeconds(s, 3)
	if !st.Mon.Alerted {
		t.Fatal("not alerted after the wait")
	}
	first := st.Mon.LastAlert
	runSeconds(s, 40)
	if st.Mon.LastAlert != first {
		t.Fatal("alerted again for the same stall")
	}
	if n := machine.StalledCount(s.Session); n != 1 {
		t.Fatalf("%d machines stalled, want 1", n)
	}
	// Free the output: the machine works and the alert ends.
	for _, slot := range st.Def.Output {
		_ = st.Inv().SetItem(slot, item.Stack{})
	}
	tick(s, 10)
	runSeconds(s, 1)
	if st.Mon.Alerted || st.Mon.Status != machine.StatusWorking {
		t.Fatalf("alerted %v status %v after freeing the output", st.Mon.Alerted, st.Mon.Status)
	}
}

// TestStarvedIsQuiet: a machine that ran dry is listed but never alerted.
func TestStarvedIsQuiet(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 1))
	tick(s, machine.Kinds[machine.KindFurnace].Interval+5)
	runSeconds(s, 1)
	runSeconds(s, 2*machine.AlertAfter)
	if st.Mon.Alerted {
		t.Fatal("a machine without input was alerted")
	}
	if st.Mon.Status != machine.StatusStarved {
		t.Fatalf("status %v, want starved", st.Mon.Status)
	}
}

// TestStatsScreens: every page of the statistics screen opens, with and
// without records, and the alert switch is saved per account.
func TestStatsScreens(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindFurnace, "")
	_ = st.Inv().SetItem(11, item.NewStack(block.Cobblestone{}, 2))
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, tab := range []machine.StatsTab{machine.StatsItems, machine.StatsPower, machine.StatsStalls} {
			machine.ShowStats(s.Session, p, tab)
		}
	})
	tick(s, 2*(machine.Kinds[machine.KindFurnace].Interval+2))
	runSeconds(s, 3)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, tab := range []machine.StatsTab{machine.StatsItems, machine.StatsPower, machine.StatsStalls} {
			machine.ShowStats(s.Session, p, tab)
		}
		body, opts := machine.StallOptions(s.Session, p)
		if body == "" || len(opts) == 0 {
			t.Fatal("empty stall page")
		}
		opts[0].Run(p, tx)
		if !machine.FactorySaved.Of(s.Session).AlertsOn[instance.AccountID(p)] {
			t.Fatal("the alert switch was not saved")
		}
	})
}
