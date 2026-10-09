// Package perf measures the server load (world tick timing, process CPU and
// memory) and formats it as sidebar lines for operators.
package perf

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

// Counts returns the number of online players and active instances. main
// sets it; it is nil in tests.
var Counts func() (players, instances int)

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

// Lines returns the sidebar lines of the server load as seen from w: the
// tick rate and tick time of w, the process CPU and memory, and the player
// and instance counts.
func Lines(w *world.World) []string {
	t := w.TickStats()
	load := float64(t.MSPT) / float64(time.Second/20)
	tpsColour := "§a"
	if t.TPS < 19 {
		tpsColour = "§e"
	}
	if t.TPS < 15 {
		tpsColour = "§c"
	}
	lines := []string{
		"§8── 서버 상태 ──",
		fmt.Sprintf("§7TPS %s%.1f §7· 부하 %s%.0f%%", tpsColour, t.TPS, loadColour(load), load*100),
		fmt.Sprintf("§7틱 %s §7(최대 %s)", ms(t.MSPT), ms(t.MaxMSPT)),
	}
	p := ProcessStats()
	cpu := "§7CPU -"
	if p.CPUOK {
		cpu = fmt.Sprintf("§7CPU %.0f%%", p.CPU)
	}
	lines = append(lines, fmt.Sprintf("%s §7· 메모리 %.0fMB", cpu, p.HeapMB))
	if Counts != nil {
		players, instances := Counts()
		lines = append(lines, fmt.Sprintf("§7접속 %d명 · 인스턴스 %d개 · G %d", players, instances, p.Goroutines))
	}
	return lines
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}
