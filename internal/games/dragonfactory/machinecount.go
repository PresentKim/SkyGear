package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/world"
)

// What machines do, counted for the players of the island (counter.go).

// cycleOnReset marks the machines whose finished work cycle shows as their
// progress going back to zero from a full interval; the others count their
// cycles where they finish them (machineState.done).
var cycleOnReset = map[string]bool{
	KindCrusher: true, KindFurnace: true, KindAlloy: true, KindWasher: true, KindCrucible: true,
	KindCrafter: true, KindSimulator: true,
}

// countWork passes the work a machine did to the players' counters: its
// finished cycles at once, the RE it generated once a second.
func (st *machineState) countWork(tx *world.Tx, kind string, tick int64) {
	if st.done > 0 {
		countAll(tx, "cycle_total", int64(st.done))
		countAll(tx, "cycle_"+kind, int64(st.done))
		st.done = 0
	}
	for key, n := range st.tallyCounts {
		countAll(tx, key, n)
		delete(st.tallyCounts, key)
	}
	if st.made > 0 && tick%20 == 0 {
		countAll(tx, "energy_made", int64(st.made))
		if st.sess != nil {
			st.sess.stats.buckets[st.sess.stats.head].gen += st.made
		}
		st.made = 0
	}
}

// tally adds to a count the players' counters get at the machine's next
// tick.
func (st *machineState) tally(key string, n int64) {
	if st.tallyCounts == nil {
		st.tallyCounts = map[string]int64{}
	}
	st.tallyCounts[key] += n
}
