package world

import (
	"sync"
	"time"
)

// dragonfactory patch: tick timing of a World, for showing the server load.

// TickStats summarises the recent ticks of a World.
type TickStats struct {
	// TPS is the number of ticks per second over the last few seconds. It
	// stays near 20 unless ticks take longer than 50ms, in which case the
	// ticker drops ticks.
	TPS float64
	// MSPT is the average time a tick took over the last second, including
	// the time it waited behind other transactions of the World.
	MSPT time.Duration
	// MaxMSPT is the longest tick of the last second.
	MaxMSPT time.Duration
}

// tickStatsWindow is how far back TPS counts ticks.
const tickStatsWindow = 5 * time.Second

type tickRecord struct {
	at  time.Time
	dur time.Duration
}

// tickStats keeps the ticks of the last tickStatsWindow.
type tickStats struct {
	mu      sync.Mutex
	records []tickRecord
	started time.Time
}

func (s *tickStats) add(at time.Time, dur time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started.IsZero() {
		s.started = at
	}
	s.records = append(s.records, tickRecord{at: at, dur: dur})
	cut := 0
	for cut < len(s.records) && at.Sub(s.records[cut].at) > tickStatsWindow {
		cut++
	}
	s.records = s.records[cut:]
}

// TickStats returns the timing of the World's recent ticks. Synchronous
// Worlds and Worlds that have not ticked yet return zero values.
func (w *World) TickStats() TickStats {
	s := &w.tickStats
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var out TickStats
	ticks, recent := 0, 0
	var total time.Duration
	for _, r := range s.records {
		age := now.Sub(r.at)
		if age > tickStatsWindow {
			continue
		}
		ticks++
		if age <= time.Second {
			recent++
			total += r.dur
			out.MaxMSPT = max(out.MaxMSPT, r.dur)
		}
	}
	if recent > 0 {
		out.MSPT = total / time.Duration(recent)
	}
	if window := min(tickStatsWindow, now.Sub(s.started)); ticks > 0 && window > 0 {
		out.TPS = min(20, float64(ticks)/window.Seconds())
	}
	return out
}
