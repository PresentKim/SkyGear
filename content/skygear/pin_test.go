package skygear

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// The pin harness records, for one event, the names the legacy dispatch code
// reached (core.Mark) and the event's outcome, and compares them with a text
// golden under testdata/pin. A golden is recorded once, from the code as it
// was before the dispatch moved to hooks. Re-record it with
//
//	SKYGEAR_GOLDEN=update go test ./content/skygear -run <its test>
//
// only when the legacy behaviour itself is being changed on purpose; never to
// get a failing pin green.

// pinSink collects the names written to core.Trace.
type pinSink struct {
	mu    sync.Mutex
	names []string
}

// pinWorldNames are the names of the world handler's subscribers
// (core.WorldHandler). The world fires them when it loads a column or closes an
// entity, whatever event a scenario is running, so a sink keeps them only
// while pinWorld is set (TestPinLifecycle).
var pinWorldNames = map[string]bool{"conduit.load": true, "conduit.unload": true, "island.lost": true}

var pinWorld atomic.Bool

func (k *pinSink) add(name string) {
	if pinWorldNames[name] && !pinWorld.Load() {
		return
	}
	k.mu.Lock()
	k.names = append(k.names, name)
	k.mu.Unlock()
}

func (k *pinSink) take() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := k.names
	k.names = nil
	return out
}

// pinRun sets core.Trace to a sink, runs call, resets core.Trace to nil and
// stores "<names space separated> | <outcome>" as lines[scenario]. outcome is
// read after call and may be nil (empty outcome). A scenario is recorded once.
func pinRun(t *testing.T, lines map[string]string, scenario string, outcome func() string, call func()) {
	t.Helper()
	if _, dup := lines[scenario]; dup {
		t.Fatalf("pin scenario %q recorded twice", scenario)
	}
	var k pinSink
	core.Trace = k.add
	func() {
		defer func() { core.Trace = nil }()
		call()
	}()
	out := ""
	if outcome != nil {
		out = outcome()
	}
	lines[scenario] = strings.Join(k.take(), " ") + " | " + out
}

// pinGolden compares lines with the text file (one "scenario: names |
// outcome" line per scenario, sorted by scenario). With SKYGEAR_GOLDEN=update
// it writes the file instead. A difference prints the scenario, the wanted
// line and the line it got.
func pinGolden(t *testing.T, file string, lines map[string]string) {
	t.Helper()
	keys := make([]string, 0, len(lines))
	for name := range lines {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, name := range keys {
		fmt.Fprintf(&b, "%s: %s\n", name, lines[name])
	}
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("golden %s: %v\nrecord it with SKYGEAR_GOLDEN=update", file, err)
	}
	want := map[string]string{}
	var order []string
	for _, ln := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		name, rest, ok := strings.Cut(ln, ": ")
		if !ok {
			t.Fatalf("golden %s: bad line %q", file, ln)
		}
		want[name] = rest
		order = append(order, name)
	}
	failed := false
	for _, name := range order {
		got, ok := lines[name]
		switch {
		case !ok:
			t.Errorf("golden %s: scenario %s is not recorded any more\nwant: %s", file, name, want[name])
			failed = true
		case got != want[name]:
			t.Errorf("golden %s: scenario %s\nwant: %s\ngot:  %s", file, name, want[name], got)
			failed = true
		}
	}
	for _, name := range keys {
		if _, ok := want[name]; !ok {
			t.Errorf("golden %s: scenario %s is new\ngot: %s", file, name, lines[name])
			failed = true
		}
	}
	if failed {
		t.Logf("record with SKYGEAR_GOLDEN=update only if a step was added, removed or reordered on purpose")
	}
}

// pinMarksIn returns the mark names written in the source of file from the
// first line containing from onward.
func pinMarksIn(t *testing.T, file, from string) []string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	i := strings.Index(src, from)
	if i < 0 {
		t.Fatalf("%s: %q not found", file, from)
	}
	var out []string
	seen := map[string]bool{}
	for _, ln := range strings.Split(src[i:], "\n") {
		ln = strings.TrimSpace(ln)
		if rest, ok := strings.CutPrefix(ln, `core.Mark("`); ok {
			name, _, _ := strings.Cut(rest, `"`)
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// pinCheckReached fails for every name that no recorded line holds.
func pinCheckReached(t *testing.T, names []string, lines map[string]string) {
	t.Helper()
	reached := map[string]bool{}
	for _, ln := range lines {
		names, _, _ := strings.Cut(ln, " | ")
		for _, n := range strings.Fields(names) {
			reached[n] = true
		}
	}
	for _, n := range names {
		if !reached[n] {
			t.Errorf("mark %s is reached by no scenario", n)
		}
	}
}
