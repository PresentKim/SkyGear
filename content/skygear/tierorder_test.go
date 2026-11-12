package skygear

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
)

// The tier order guard. A subscriber's place is its tier; inside a tier no
// subscriber may depend on where it stands. TestTierOrder proves it: it runs
// this test binary again with the subscribers of every hook reordered inside
// their tiers (SKYGEAR_HOOK_ORDER) and compares what every pin scenario did.
//
// Until the kernel has tiers, the tier of each subscriber is the one planned
// for it (plannedTiers), and a reordered run first sorts each hook by planned
// tier, keeping today's order inside a tier ("tiers"), then reverses
// ("reverse") or shuffles ("shuffle:<seed>") every tier.

// hookOrder is the reordering of this process: "" (the order of the code),
// "tiers", "reverse" or "shuffle:<seed>".
var hookOrder = os.Getenv("SKYGEAR_HOOK_ORDER")

// pinDump, when set, is the file every pin test appends its scenarios to:
// one line "<scenario> | <results> | <counters>" each (pinDumpLines).
var pinDump = os.Getenv("SKYGEAR_PIN_DUMP")

// tierNames are the tiers in the order they run.
var tierNames = []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}

// plannedTiers is the tier planned for every subscriber, one
// "<Hook> <TIER> <name>" per line, in today's order.
const plannedTiers = `
StartBreak LOWEST expand.guard
StartBreak LOW island.guard
StartBreak NORMAL tinker.autoTool
BlockBreak LOWEST expand.guard
BlockBreak LOW island.guard
BlockBreak MONITOR achievement.break
BlockBreak NORMAL tinker.break
BlockBreak NORMAL farm.break
BlockBreak NORMAL storage.break
BlockBreak NORMAL netstore.break
BlockBreak NORMAL machine.break
BlockBreak NORMAL conduit.break
BlockPlace LOWEST expand.guard
BlockPlace LOW island.guard
BlockPlace MONITOR achievement.place
BlockPlace HIGH autofill.place
BlockPlace NORMAL farm.plant
BlockPlace NORMAL storage.place
BlockPlace NORMAL machine.place
BlockPlace HIGH netstore.place
BlockPlace MONITOR conduit.place
BlockPlace MONITOR progress.dragonEgg
UseOnBlock LOWEST dimension.hunting
UseOnBlock LOWEST screens.menuItem
UseOnBlock LOWEST dimension.stone
UseOnBlock LOWEST expand.guard
UseOnBlock LOW island.guard
UseOnBlock NORMAL machine.noteFace
UseOnBlock LOW copier.use
UseOnBlock LOW wand.use
UseOnBlock NORMAL storage.sneakUse
UseOnBlock NORMAL conduit.plate
UseOnBlock NORMAL handwork.stick
UseOnBlock HIGH farm.composter
UseOnBlock NORMAL handwork.gather
UseOnBlock NORMAL fairy.time
UseOnBlock HIGH farm.crop
UseOnBlock HIGHEST expand.guardSide
UseOnBlock HIGHEST island.guardSide
UseOnEntity LOWEST dimension.hunting
UseOnEntity NORMAL fairy.touch
AttackEntity LOWEST dimension.hunter
AttackEntity NORMAL fairy.touch
Use HIGH autofill.use
Use LOWEST screens.menuItem
Use LOWEST dimension.hunting
Use LOWEST dimension.stone
Use NORMAL copier.clear
Use NORMAL wand.options
Use NORMAL tinker.veinToggle
Move LOWEST look.update
Move LOWEST expand.hold
Move HIGH achievement.walk
Move HIGHEST island.void
Sneak LOWEST expand.show
Sneak LOW achievement.sneak
Sneak NORMAL elevator.down
Sneak HIGH farm.crouchGrow
Sprint MONITOR achievement.sprint
Jump MONITOR achievement.jump
Jump NORMAL elevator.up
ItemDamage LOW tinker.ironHammer
ItemDamage HIGH autofill.broken
ItemConsume MONITOR achievement.eat
ItemConsume HIGH autofill.consume
ItemPickup MONITOR achievement.pickup
ItemDrop LOWEST dimension.hunting
ItemDrop MONITOR achievement.drop
Death MONITOR achievement.death
Death NORMAL dimension.died
Death NORMAL tinker.keepFood
Respawn LOW dimension.respawned
Respawn NORMAL tinker.restoreFood
Hurt LOWEST dimension.frozen
Hurt MONITOR achievement.hurt
Hurt HIGHEST dimension.fallen
PunchAir MONITOR achievement.punchAir
Chat MONITOR achievement.chat
BlockPick MONITOR achievement.blockPick
HeldSlot MONITOR achievement.heldSlot
Command MONITOR achievement.command
Load NORMAL farm.loadCrops
Start NORMAL island.keepLoaded
Start NORMAL climate.apply
Join NORMAL screens.menuItem
Join NORMAL fairy.backfillDust
Join NORMAL climate.showWeather
Join NORMAL core.shapes
Join NORMAL island.purgeLost
Join NORMAL progress.sendRecipes
Join MONITOR screens.board
Join HIGH dimension.joined
Leave NORMAL dimension.leave
Leave NORMAL expand.stop
Leave NORMAL core.parts
Leave HIGHEST look.clear
Leave NORMAL core.shapes
Leave HIGHEST screens.board
Leave HIGHEST progress.recipes
NewDay NORMAL island.purgeLost
Second NORMAL farm.growCrops
Second LOW progress.checkHeld
Second NORMAL farm.barrels
Second NORMAL fairy.tick
Second NORMAL climate.tick
Second NORMAL dimension.second
Second NORMAL tinker.effects
Second NORMAL machine.monitor
Second NORMAL farm.plants
Second MONITOR screens.boards
Second MONITOR achievement.playSeconds
Second MONITOR progress.collection
Fast NORMAL network.step
Fast NORMAL dimension.watchdog
Fast MONITOR look.updateAll
Autosave NORMAL netstore.flush
Autosave NORMAL conduit.shares
Close NORMAL storage.displays
Close NORMAL conduit.close
Close NORMAL netstore.flush
ColumnLoad NORMAL conduit.load
ColumnUnload NORMAL conduit.unload
EntityDespawn NORMAL island.lost
`

// orderKept are the pairs that still depend on their order inside one tier:
// "<Hook> <first> <second>". A reordered run keeps first before second. Each
// entry is a dependency to remove (or, for the last ones, an exception the
// owner decides on); the list only shrinks.
var orderKept = []string{
	// a click the drawer menu or a plate takes counts no stick use
	"UseOnBlock storage.sneakUse handwork.stick",
	"UseOnBlock conduit.plate handwork.stick",
	// a plate's settings open before the hourglass winds the machine
	"UseOnBlock conduit.plate fairy.time",
	// the panel updates on the move island expansion mode refuses
	"Move look.update expand.hold",
	// the player's entries are dropped after the features used them
	"Leave dimension.leave core.parts",
	"Leave expand.stop core.parts",
}

func tierIndex(name string) int {
	i := slices.Index(tierNames, name)
	if i < 0 {
		panic("unknown tier " + name)
	}
	return i
}

// plannedTierOf returns the planned tier of every subscriber, keyed
// "<Hook> <name>".
func plannedTierOf() map[string]int {
	out := map[string]int{}
	for _, ln := range strings.Split(strings.TrimSpace(plannedTiers), "\n") {
		f := strings.Fields(ln)
		out[f[0]+" "+f[2]] = tierIndex(f[1])
	}
	return out
}

// reorderHooks applies mode to the kernel's hooks (core.Reorder).
func reorderHooks(mode string) {
	tiers := plannedTierOf()
	kind, arg, _ := strings.Cut(mode, ":")
	var seed uint64
	if kind == "shuffle" {
		if _, err := fmt.Sscan(arg, &seed); err != nil {
			panic("SKYGEAR_HOOK_ORDER: " + mode)
		}
	} else if kind != "tiers" && kind != "reverse" {
		panic("SKYGEAR_HOOK_ORDER: " + mode)
	}
	core.Reorder(func(hook string, subs []core.Subscription) []int {
		groups := make([][]int, len(tierNames))
		for i, s := range subs {
			t, ok := tiers[hook+" "+s.Name]
			if !ok {
				panic(fmt.Sprintf("plannedTiers has no line for %s %s", hook, s.Name))
			}
			groups[t] = append(groups[t], i)
		}
		h := fnv.New64a()
		h.Write([]byte(hook))
		r := rand.New(rand.NewPCG(seed, h.Sum64()))
		var idx []int
		for _, g := range groups {
			switch kind {
			case "reverse":
				slices.Reverse(g)
			case "shuffle":
				r.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
			}
			idx = append(idx, g...)
		}
		// Put the kept pairs back in their order: a first that ended up after
		// its second moves to just before it.
		for moved := true; moved; {
			moved = false
			for _, k := range orderKept {
				f := strings.Fields(k)
				if f[0] != hook {
					continue
				}
				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
				if a < 0 || b < 0 {
					panic("orderKept names no subscriber: " + k)
				}
				if a > b {
					v := idx[a]
					idx = slices.Delete(idx, a, a+1)
					idx = slices.Insert(idx, b, v)
					moved = true
				}
			}
		}
		return idx
	})
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
	if hookOrder != "" {
		reorderHooks(hookOrder)
	}
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
var tierOrders = []string{"tiers", "reverse", "shuffle:1", "shuffle:2", "shuffle:3"}

// TestTierOrder runs every test of this package once per reordering of
// tierOrders, in a process of its own, and once in the order of the code. A
// reordered run must pass, which includes that every pin scenario has the
// results of its golden (the trace before the first " | " may differ), and
// must count exactly what the run in the order of the code counted in every
// scenario (the achievement counters of the players, which no golden holds).
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
		cmd.Env = append(os.Environ(), "SKYGEAR_HOOK_ORDER="+mode, "SKYGEAR_PIN_DUMP="+dump)
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
		t.Fatalf("the run in the order of the code dumped %d scenarios", len(want))
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
