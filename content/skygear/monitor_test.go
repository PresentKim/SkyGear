package skygear

import (
	"testing"

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
		s.World.Do(func(tx *world.Tx) { s.tickMonitor(tx) })
	}
}

// TestProductionRecorded: what a machine puts in its output is recorded in the
// window and the saved totals.
func TestProductionRecorded(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 3))
	tick(s, 3*(machineKinds[KindFurnace].interval+2))
	made := outputCount(st, block.Stone{})
	if made == 0 {
		t.Fatal("nothing was smelted")
	}
	key := naming.ItemKey(block.Stone{})
	if got := s.state.Produced[key]; got != int64(made) {
		t.Fatalf("saved total %d, want %d", got, made)
	}
	items, _ := s.stats.window(1)
	if items[key] != made {
		t.Fatalf("window has %d, want %d", items[key], made)
	}
	if got := s.keyStack(key); got.Empty() || naming.ItemKey(got.Item()) != key {
		t.Fatalf("key %q does not give its item back: %v", key, got)
	}
}

// TestMinuteWindows: the minutes rotate and the windows cover the right time.
func TestMinuteWindows(t *testing.T) {
	p := newProdStats()
	state := newState()
	p.add(state, item.NewStack(block.Cobblestone{}, 1), 60, true)
	for range 60 {
		p.secs++
	}
	p.rotate()
	p.add(state, item.NewStack(block.Cobblestone{}, 1), 30, true)
	p.secs = 30
	items, minutes := p.window(5)
	key := naming.ItemKey(block.Cobblestone{})
	if items[key] != 90 || minutes != 1.5 {
		t.Fatalf("5 minute window: %d items over %v minutes", items[key], minutes)
	}
	if got := perMinute(items[key], minutes); got != 60 {
		t.Fatalf("rate %v, want 60", got)
	}
	for range statMinutes + 3 {
		p.rotate()
	}
	if items, _ := p.window(15); len(items) != 0 {
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
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 64))
	for _, slot := range st.kind.output {
		_ = st.inv().SetItem(slot, item.NewStack(block.Dirt{}, 64))
	}
	// A worked-before furnace: it finished one, then the output filled.
	tick(s, machineKinds[KindFurnace].interval+5)
	runSeconds(s, 1)
	tick(s, 5)
	runSeconds(s, 1)
	if st.mon.status != statusOutput {
		t.Fatalf("status %v, want output blocked", st.mon.status)
	}
	runSeconds(s, alertAfter-2)
	if st.mon.alerted {
		t.Fatal("alerted too early")
	}
	runSeconds(s, 3)
	if !st.mon.alerted {
		t.Fatal("not alerted after the wait")
	}
	first := st.mon.lastAlert
	runSeconds(s, 40)
	if st.mon.lastAlert != first {
		t.Fatal("alerted again for the same stall")
	}
	if n := s.stalledCount(); n != 1 {
		t.Fatalf("%d machines stalled, want 1", n)
	}
	// Free the output: the machine works and the alert ends.
	for _, slot := range st.kind.output {
		_ = st.inv().SetItem(slot, item.Stack{})
	}
	tick(s, 10)
	runSeconds(s, 1)
	if st.mon.alerted || st.mon.status != statusWorking {
		t.Fatalf("alerted %v status %v after freeing the output", st.mon.alerted, st.mon.status)
	}
}

// TestStarvedIsQuiet: a machine that ran dry is listed but never alerted.
func TestStarvedIsQuiet(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 1))
	tick(s, machineKinds[KindFurnace].interval+5)
	runSeconds(s, 1)
	runSeconds(s, 2*alertAfter)
	if st.mon.alerted {
		t.Fatal("a machine without input was alerted")
	}
	if st.mon.status != statusStarved {
		t.Fatalf("status %v, want starved", st.mon.status)
	}
}

// TestStatsScreens: every page of the statistics screen opens, with and
// without records, and the alert switch is saved per account.
func TestStatsScreens(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindFurnace, "")
	_ = st.inv().SetItem(11, item.NewStack(block.Cobblestone{}, 2))
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, tab := range []statsTab{statsItems, statsPower, statsStalls} {
			s.showStats(p, tab)
		}
	})
	tick(s, 2*(machineKinds[KindFurnace].interval+2))
	runSeconds(s, 3)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		for _, tab := range []statsTab{statsItems, statsPower, statsStalls} {
			s.showStats(p, tab)
		}
		body, opts := s.stallOptions(p)
		if body == "" || len(opts) == 0 {
			t.Fatal("empty stall page")
		}
		opts[0].Run(p, tx)
		if !s.state.AlertsOn[instance.AccountID(p)] {
			t.Fatal("the alert switch was not saved")
		}
	})
}
