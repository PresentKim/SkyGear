package skygear

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
)

// The tier order guard. A subscriber's place is its tier; inside a tier no
// subscriber may depend on where it stands. TestTierOrder proves it: it runs
// this test binary again with the subscribers of every hook reordered inside
// their tiers (coretest.HookOrderEnv: reversed, and shuffled with a few seeds)
// and compares what every pin scenario did.

// hookOrder is the reordering of this process: "" (the order of
// registration), "reverse" or "shuffle:<seed>".
var hookOrder = os.Getenv(coretest.HookOrderEnv)

// pinDump, when set, is the file every pin test appends its scenarios to:
// one line "<scenario> | <results> | <counters> | <inventories>" each
// (pinDumpLines).
var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")

// orderKept are the exceptions: the pairs of one tier whose order still
// matters, "<Hook> <first> <second>". The kernel runs first before second
// because first's feature is installed earlier (features.go), and a reordered
// run keeps them so. Each is a known dependency on the place inside a tier,
// kept until it is decided how to remove it (docs/tech/extension.md); the
// list does not grow.
var orderKept = []string{
	// a connector plate's settings open before the hourglass in hand winds
	// the machine the plate is on
	"UseOnBlock conduit.plate fairy.time",
}

// pinAccounts are the accounts of the players the tests spawned, in a run
// that dumps (spawnPlayer).
var (
	pinAccountsMu sync.Mutex
	pinAccounts   []string
	pinStoreDB    *db.DB
	pinDumpMu     sync.Mutex
	pinDumpLines  = map[string]string{}
	pinCounterIDs []string
)

func TestMain(m *testing.M) {
	coretest.ReorderHooks(orderKept...)
	if pinDump != "" {
		dir, err := os.MkdirTemp("", "pindump")
		if err != nil {
			panic(err)
		}
		d, err := db.Open(filepath.Join(dir, "db.sqlite"))
		if err != nil {
			panic(err)
		}
		pinStoreDB = d
		code := m.Run()
		_ = d.Close()
		_ = os.RemoveAll(dir)
		os.Exit(code)
	}
	os.Exit(m.Run())
}

// pinNotePlayer remembers the account of a spawned player in a run that
// dumps, and makes the account exist in the dump's store.
func pinNotePlayer(h *world.EntityHandle) {
	if pinDump == "" {
		return
	}
	acc := h.UUID().String()
	_, _ = pinStoreDB.TouchAccount(acc, "", "p")
	pinAccountsMu.Lock()
	pinAccounts = append(pinAccounts, acc)
	pinAccountsMu.Unlock()
}

// pinCounters returns the count of every counter, summed over the accounts of
// the players spawned so far, keyed by the counter's first achievement id. In
// a run that does not dump it returns nil.
func pinCounters() map[string]int64 {
	if pinDump == "" {
		return nil
	}
	// A test that used its own account store put none back.
	account.Init(pinStoreDB, slog.New(slog.DiscardHandler))
	if pinCounterIDs == nil {
		seen := map[string]bool{}
		for _, a := range account.All() {
			if _, _, ok := account.Progress("", a.ID); !ok {
				continue
			}
			key := a.ID[:strings.LastIndex(a.ID, "_")]
			if !seen[key] {
				seen[key] = true
				pinCounterIDs = append(pinCounterIDs, a.ID)
			}
		}
	}
	pinAccountsMu.Lock()
	accs := slices.Clone(pinAccounts)
	pinAccountsMu.Unlock()
	out := map[string]int64{}
	for _, acc := range accs {
		for _, id := range pinCounterIDs {
			if have, _, _ := account.Progress(acc, id); have != 0 {
				out[id[:strings.LastIndex(id, "_")]] += have
			}
		}
	}
	return out
}

// pinCountersSince is what was counted since before, as
// "<counter>+<n>" sorted and joined by commas, "-" for nothing.
func pinCountersSince(before map[string]int64) string {
	var out []string
	for key, n := range pinCounters() {
		if d := n - before[key]; d != 0 {
			out = append(out, fmt.Sprintf("%s+%d", strings.TrimPrefix(key, "sg_"), d))
		}
	}
	if len(out) == 0 {
		return "-"
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// pinInventories are the inventories of the players of pinInventoryWorld,
// the world of the last player a test ran with (withPlayer), in a run that
// dumps.
var (
	pinInventories    []*inventory.Inventory
	pinInventoryWorld *world.World
)

// pinNoteInventory remembers the inventory of a player a test runs with, in
// a run that dumps. A player of another world than the last starts the list
// again: each scenario has a world of its own.
func pinNoteInventory(p *player.Player) {
	if pinDump == "" {
		return
	}
	pinDumpMu.Lock()
	defer pinDumpMu.Unlock()
	if w := p.Tx().World(); w != pinInventoryWorld {
		pinInventoryWorld, pinInventories = w, nil
	}
	if inv := p.Inventory(); !slices.Contains(pinInventories, inv) {
		pinInventories = append(pinInventories, inv)
	}
}

// pinInventoryDump is what the players of the last scenario's world hold, one
// "<slot>=<item>x<count>" per used slot, the players separated by "/"; "-"
// for a player that holds nothing.
func pinInventoryDump() string {
	pinDumpMu.Lock()
	defer pinDumpMu.Unlock()
	var players []string
	for _, inv := range pinInventories {
		var slots []string
		for slot, it := range inv.Slots() {
			if !it.Empty() {
				slots = append(slots, fmt.Sprintf("%d=%sx%d", slot, naming.ItemKey(it.Item()), it.Count()))
			}
		}
		if len(slots) == 0 {
			slots = []string{"-"}
		}
		players = append(players, strings.Join(slots, ","))
	}
	if len(players) == 0 {
		return "-"
	}
	return strings.Join(players, "/")
}

// pinResults is the part of a pin line after the trace.
func pinResults(line string) string {
	_, res, ok := strings.Cut(line, " | ")
	if !ok {
		return line
	}
	return res
}

// pinWriteDump appends the dump lines of the scenarios of lines to pinDump.
func pinWriteDump(t *testing.T, lines map[string]string) {
	t.Helper()
	if pinDump == "" {
		return
	}
	f, err := os.OpenFile(pinDump, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pinDumpMu.Lock()
	defer pinDumpMu.Unlock()
	for name, ln := range lines {
		fmt.Fprintf(f, "%s | %s | %s\n", name, pinResults(ln), pinDumpLines[name])
	}
}

// tierOrders are the reorderings TestTierOrder runs.
var tierOrders = []string{"reverse", "shuffle:1", "shuffle:2", "shuffle:3", "shuffle:4"}

// TestTierOrder runs every test of this package once per reordering of
// tierOrders, in a process of its own, and once in the order of
// registration. A reordered run must pass, which includes that every pin
// scenario has the results of its golden (the trace before the first " | "
// may differ) and that order.golden lists the same subscribers in every tier,
// and it must count exactly what the run in the order of registration counted
// in every scenario (the achievement counters of the players, which no golden
// holds) and leave the players the scenario ran with holding exactly the
// same items in the same slots.
func TestTierOrder(t *testing.T) {
	if hookOrder != "" || pinDump != "" {
		return // this is a run TestTierOrder started
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	run := func(mode string) (dump string, out []byte, err error) {
		dump = filepath.Join(dir, strings.ReplaceAll(mode+".dump", ":", "-"))
		cmd := exec.Command(exe, "-test.count=1")
		cmd.Env = append(os.Environ(), coretest.HookOrderEnv+"="+mode, "SKYGEAR_PIN_DUMP="+dump)
		out, err = cmd.CombinedOutput()
		return dump, out, err
	}
	read := func(file string) map[string]string {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, ln := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
			name, rest, _ := strings.Cut(ln, " | ")
			out[name] = rest
		}
		return out
	}
	type result struct {
		mode, dump string
		out        []byte
		err        error
	}
	modes := append([]string{""}, tierOrders...)
	results := make([]result, len(modes))
	var wg sync.WaitGroup
	for i, mode := range modes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dump, out, err := run(mode)
			results[i] = result{mode, dump, out, err}
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r.err != nil {
			t.Fatalf("order %q: the tests fail: %v\n%s", r.mode, r.err, failLines(r.out))
		}
	}
	want := read(results[0].dump)
	if len(want) < 200 {
		t.Fatalf("the run in the order of registration dumped %d scenarios", len(want))
	}
	for _, r := range results[1:] {
		got := read(r.dump)
		for name, w := range want {
			if g, ok := got[name]; !ok || g != w {
				t.Errorf("order %q: scenario %s\nwant: %s\ngot:  %s", r.mode, name, w, g)
			}
		}
		if len(got) != len(want) {
			t.Errorf("order %q: %d scenarios, want %d", r.mode, len(got), len(want))
		}
	}
	t.Logf("%d orders, %d scenarios each", len(results)-1, len(want))
}

// failLines keeps the lines of a test run's output that say what failed.
func failLines(out []byte) string {
	var keep []string
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "20") && strings.Contains(ln, " INFO ") {
			continue
		}
		keep = append(keep, ln)
	}
	if len(keep) > 80 {
		keep = keep[:80]
	}
	return strings.Join(keep, "\n")
}
