package machine

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/achievement"
)

// What machines do, counted for the players of the island (counter.go).

func init() {
	for kind := range CycleOnReset {
		Kinds[kind].CycleOnReset = true
	}
}

// CycleOnReset marks the machines whose finished work cycle shows as their
// progress going back to zero from a full interval; the others count their
// cycles where they finish them (machineState.done).
var CycleOnReset = map[string]bool{
	KindCrusher: true, KindFurnace: true, KindAlloy: true, KindWasher: true, KindCrucible: true,
	KindCrafter: true, KindSimulator: true,
}

// countWork passes the work a machine did to the players' counters: its
// finished cycles at once, the RE it generated once a second.
func (st *State) countWork(tx *world.Tx, kind string, tick int64) {
	if st.Done > 0 {
		achievement.CountAll(tx, "cycle_total", int64(st.Done))
		achievement.CountAll(tx, "cycle_"+kind, int64(st.Done))
		st.Done = 0
	}
	for key, n := range st.tallyCounts {
		achievement.CountAll(tx, key, n)
		delete(st.tallyCounts, key)
	}
	if st.made > 0 && tick%20 == 0 {
		achievement.CountAll(tx, "energy_made", int64(st.made))
		if st.Sess != nil {
			FactoryOf(st.Sess).Stats.Buckets[FactoryOf(st.Sess).Stats.Head].gen += st.made
		}
		st.made = 0
	}
}

// tally adds to a count the players' counters get at the machine's next
// tick.
func (st *State) tally(key string, n int64) {
	if st.tallyCounts == nil {
		st.tallyCounts = map[string]int64{}
	}
	st.tallyCounts[key] += n
}
