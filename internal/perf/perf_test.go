package perf

import (
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

// TestLines: a ticking world reports a tick rate near 20 and the lines show
// it with the process stats and counts.
func TestLines(t *testing.T) {
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
	Counts = func() (int, int) { return 3, 2 }
	defer func() { Counts = nil }()
	text := strings.Join(Lines(w), "\n")
	for _, want := range []string{"TPS", "부하", "틱", "메모리", "접속 3명", "인스턴스 2개"} {
		if !strings.Contains(text, want) {
			t.Errorf("lines miss %q:\n%s", want, text)
		}
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
