package skygear

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestCustomBlockCounts(t *testing.T) {
	blocks, states, perms := CustomBlockCounts()
	if blocks == 0 || states < blocks || perms < 0 {
		t.Fatalf("blocks %d, states %d, permutations %d", blocks, states, perms)
	}
}

// logRecords runs f with a logger that writes its lines (message and
// attributes, level first) into the returned buffer.
func logRecords(f func(log *slog.Logger)) string {
	var buf bytes.Buffer
	f(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf.String()
}

// TestBlockNameCapWarning: the log warns of each name over 65,536 states and
// of none else.
func TestBlockNameCapWarning(t *testing.T) {
	out := logRecords(func(log *slog.Logger) {
		logBlockNameCounts(log, []blockNameCount{{name: "skygear:big", states: 70000}, {name: "skygear:ok", states: 65536}})
	})
	if strings.Count(out, "level=WARN") != 1 || strings.Contains(out, "level=INFO") || !strings.Contains(out, "skygear:big") {
		t.Errorf("log:\n%s", out)
	}
	// the registry of this run: the custom blocks line, and no name over the cap
	out = logRecords(LogCustomBlockCounts)
	if !strings.Contains(out, "max_states_per_name=") {
		t.Errorf("custom blocks line has no max states per name:\n%s", out)
	}
	if strings.Contains(out, "level=WARN") {
		t.Errorf("a custom block is over the cap:\n%s", out)
	}
}
