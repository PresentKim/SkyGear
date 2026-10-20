package perf

import (
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

// TestLine: a ticking world reports a tick rate near 20 and the line shows
// it with the memory.
func TestLine(t *testing.T) {
	w := world.Config{Generator: world.NopGenerator{}}.New()
	defer w.Close()
	time.Sleep(600 * time.Millisecond)
	st := w.TickStats()
	if st.TPS < 10 || st.TPS > 20 {
		t.Errorf("TPS = %.1f, want about 20", st.TPS)
	}
	if st.MSPT <= 0 || st.MaxMSPT < st.MSPT {
		t.Errorf("MSPT = %v, max %v", st.MSPT, st.MaxMSPT)
	}
	text := Line(w)
	for _, want := range []string{"TPS", "부하", "틱", "메모리"} {
		if !strings.Contains(text, want) {
			t.Errorf("line misses %q: %s", want, text)
		}
	}
}

func TestToggle(t *testing.T) {
	if Shown("a") || !Toggle("a") || !Shown("a") || Toggle("a") || Shown("a") {
		t.Fatal("toggle")
	}
}

// TestProcessStatsCPU: the second sample measures CPU time where supported.
func TestProcessStatsCPU(t *testing.T) {
	first := ProcessStats()
	if first.HeapMB <= 0 || first.Goroutines <= 0 {
		t.Fatalf("empty sample %+v", first)
	}
	if _, ok := cpuTime(); !ok {
		t.Skip("CPU time not measured on this platform")
	}
	time.Sleep(1100 * time.Millisecond)
	if p := ProcessStats(); !p.CPUOK || p.CPU < 0 {
		t.Errorf("second sample %+v, want CPU", p)
	}
}
