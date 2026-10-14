// Package perf measures the server load (world tick timing, process CPU and
// memory) and formats it as sidebar lines for operators.
package perf

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"runtime"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

// Process is the load of the server process.
type Process struct {
	// CPU is the CPU time used over the last sample, in percent of one
	// core (200 = two cores busy). CPUOK is false where it is not measured.
	CPU   float64
	CPUOK bool
	// HeapMB is the memory held by live Go objects.
	HeapMB float64
	// Goroutines is the number of goroutines.
	Goroutines int
}

var (
	mu       sync.Mutex
	last     Process
	lastAt   time.Time
	lastCPU  time.Duration
	cpuKnown bool
)

// ProcessStats samples the process at most once a second and returns the
// latest sample.
func ProcessStats() Process {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	if !lastAt.IsZero() && now.Sub(lastAt) < time.Second {
		return last
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	p := Process{HeapMB: float64(mem.HeapAlloc) / (1 << 20), Goroutines: runtime.NumGoroutine()}
	if used, ok := cpuTime(); ok {
		if cpuKnown && !lastAt.IsZero() {
			p.CPU = float64(used-lastCPU) / float64(now.Sub(lastAt)) * 100
			p.CPUOK = true
		}
		lastCPU, cpuKnown = used, true
	}
	last, lastAt = p, now
	return last
}

// loadColour colours a share of the 50ms tick budget.
func loadColour(load float64) string {
	switch {
	case load < 0.5:
		return "§a"
	case load < 1:
		return "§e"
	default:
		return "§c"
	}
}

// Line returns the server load as seen from w in one short line for the
// HUD's top left: the tick rate, the share of the 50ms tick budget used,
// the average and longest tick time of the last 5 seconds and the memory of
// live Go objects.
func Line(w *world.World) string {
	t := w.TickStats()
	load := float64(t.MSPT) / float64(time.Second/20)
	tpsColour := "§a"
	if t.TPS < 19 {
		tpsColour = "§e"
	}
	if t.TPS < 15 {
		tpsColour = "§c"
	}
	return fmt.Sprintf(i18n.T(i18n.Default, "perf.perf.Line.1"),
		tpsColour, t.TPS, loadColour(load), load*100, ms(t.MSPT), ms(t.MaxMSPT), ProcessStats().HeapMB)
}

// shown are the players (by name) who turned the load line on
// (/dfadmin tps).
var (
	shownMu sync.Mutex
	shown   = map[string]bool{}
)

// Toggle turns the load line on or off for the player of the name and
// reports whether it is now on.
func Toggle(name string) bool {
	shownMu.Lock()
	defer shownMu.Unlock()
	shown[name] = !shown[name]
	if !shown[name] {
		delete(shown, name)
	}
	return shown[name]
}

// Shown reports whether the player of the name turned the load line on.
func Shown(name string) bool {
	shownMu.Lock()
	defer shownMu.Unlock()
	return shown[name]
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}
