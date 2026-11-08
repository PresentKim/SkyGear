package skygear

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/assets"
)

// TestStateProbeSplitCounts: all4split has 7 block names a layer, 14 in all,
// each with at most 65,536 states: a singles block of 769 states and 12
// permutations and 6 pair blocks of 36,864 states and 9 permutations; the
// states add up to those of all4 (which has 2 names over the cap).
func TestStateProbeSplitCounts(t *testing.T) {
	counts := stateProbeNameCounts(assets.StateProbeAll4Split)
	if len(counts) != 14 {
		t.Fatalf("%d names, want 14", len(counts))
	}
	seen, sum := map[string]bool{}, 0
	for i, c := range counts {
		if seen[c.name] {
			t.Errorf("name %s twice", c.name)
		}
		seen[c.name] = true
		if c.states > stateProbeNameCap || c.states == 0 {
			t.Errorf("%s: %d states", c.name, c.states)
		}
		wantStates, wantPerms := stateProbePairBlock, 9
		if i%7 == 0 {
			wantStates, wantPerms = 769, 12
		}
		if c.states != wantStates || c.permutations != wantPerms {
			t.Errorf("%s: %d states and %d permutations, want %d and %d", c.name, c.states, c.permutations, wantStates, wantPerms)
		}
		sum += c.states
	}
	if sum != 443906 {
		t.Errorf("%d states in all", sum)
	}
	if stateProbePairBlock != 36864 {
		t.Errorf("pair block has %d states", stateProbePairBlock)
	}
	over := 0
	for _, c := range stateProbeNameCounts(assets.StateProbeAll4) {
		if c.states > stateProbeNameCap {
			over++
		}
	}
	if over != 2 {
		t.Errorf("all4 has %d names over the cap, want 2", over)
	}
	for _, mode := range probeModes {
		blocks := 0
		for l := range stateProbeLayers {
			blocks += stateProbeBlocks(mode, l)
		}
		want := 2
		if mode == assets.StateProbeAll4Split {
			want = 14
		}
		if blocks != want {
			t.Errorf("%s: %d blocks", mode, blocks)
		}
	}
	if stateProbeBlocks(assets.StateProbeOff, 0) != 0 || len(stateProbeNameCounts(assets.StateProbeOff)) != 0 {
		t.Error("blocks with the probe off")
	}
}

// TestStateProbeSplitLocate: every value of all4split (all of them, both
// layers) is the state of exactly one block: the first value of the block plus
// the state, the state less than the block's states, and a pair block holds
// the pair of kinds its index names. Other modes are one block, state = value.
func TestStateProbeSplitLocate(t *testing.T) {
	mode := assets.StateProbeAll4Split
	for l := range stateProbeLayers {
		next := 0 // the value that must come next: blocks follow each other
		for blk := range stateProbeBlocks(mode, l) {
			lo, n := stateProbeBlockSpan(mode, l, blk)
			if lo != next {
				t.Fatalf("layer %d block %d starts at %d, want %d", l, blk, lo, next)
			}
			next = lo + n
			for s := range n {
				v := lo + s
				if gb, gs := stateProbeLocate(mode, l, v); gb != blk || gs != s {
					t.Fatalf("layer %d value %d is block %d state %d, want %d %d", l, v, gb, gs, blk, s)
				}
				if blk > 0 && s%97 == 0 {
					look, ok := stateProbeDecode(mode, l, v)
					if !ok || look.n != 2 || [2]int{look.k[0].kind, look.k[1].kind} != stateProbeKindPairs[blk-1] {
						t.Fatalf("layer %d block %d state %d: %+v", l, blk, s, look)
					}
				}
				if blk == 0 && s > 0 {
					if look, _ := stateProbeDecode(mode, l, v); look.n != 1 {
						t.Fatalf("singles state %d: %+v", s, look)
					}
				}
			}
		}
		if next != stateProbeLayerValues(mode, l) {
			t.Errorf("layer %d blocks cover %d of %d values", l, next, stateProbeLayerValues(mode, l))
		}
	}
	for _, mode := range []string{assets.StateProbePerm, assets.StateProbeAll4, assets.StateProbeSame4} {
		if b, s := stateProbeLocate(mode, 1, 4000); b != 0 || s != 4000 {
			t.Errorf("%s: %d %d", mode, b, s)
		}
		if lo, n := stateProbeBlockSpan(mode, 1, 0); lo != 0 || n != stateProbeLayerValues(mode, 1) {
			t.Errorf("%s: span %d %d", mode, lo, n)
		}
	}
}

// probeBoneShown evaluates the bone_visibility expression of a bone at the
// state s.
func probeBoneShown(t *testing.T, e string, s int) bool {
	t.Helper()
	if e == "true" {
		return true
	}
	m := probeBit.FindStringSubmatch(e)
	if m == nil {
		t.Fatalf("bone visibility %q", e)
	}
	lo, _ := strconv.Atoi(m[1])
	div, _ := strconv.Atoi(m[2])
	return (s-lo)/div%2 == 1
}

// TestStateProbeSplitPermutations: in each block of all4split every state
// (sampled) matches exactly one permutation, and it draws what the permutation
// of the same look in all4 (conditions on values) draws: the same geometry and
// textures, and the same bones shown.
func TestStateProbeSplitPermutations(t *testing.T) {
	mode := assets.StateProbeAll4Split
	bones, _ := probeFourGeometry(t)
	for l := range stateProbeLayers {
		whole := stateProbePermutations(mode, l)
		if len(whole) != 66 {
			t.Fatalf("%d permutations in all", len(whole))
		}
		match := func(conds []string, s int) (idx []int) {
			for i, c := range conds {
				m := probeRange.FindStringSubmatch(c)
				if m == nil {
					t.Fatalf("condition %q", c)
				}
				lo, _ := strconv.Atoi(m[1])
				hi, _ := strconv.Atoi(m[2])
				if s >= lo && s <= hi {
					idx = append(idx, i)
				}
			}
			return idx
		}
		var wholeConds []string
		for _, p := range whole {
			wholeConds = append(wholeConds, p.Condition)
		}
		for blk := range stateProbeBlocks(mode, l) {
			perms := stateProbeBlockPermutations(mode, l, blk)
			var conds []string
			for _, p := range perms {
				conds = append(conds, p.Condition)
			}
			lo, n := stateProbeBlockSpan(mode, l, blk)
			for s := 0; s < n; s++ {
				if s > 5000 && s%97 != 0 && s%64 != 0 && s%64 != 63 {
					continue
				}
				if s == 0 && blk == 0 {
					if len(match(conds, 0)) != 0 {
						t.Fatalf("block 0 state 0 matches a permutation")
					}
					continue
				}
				got, want := match(conds, s), match(wholeConds, lo+s)
				if len(got) != 1 || len(want) != 1 {
					t.Fatalf("layer %d block %d state %d: %d and %d matches", l, blk, s, len(got), len(want))
				}
				pg, pw := perms[got[0]].Properties, whole[want[0]].Properties
				if pg.Geometry != pw.Geometry || fmt.Sprint(pg.Textures) != fmt.Sprint(pw.Textures) {
					t.Fatalf("layer %d block %d state %d: %s %v, want %s %v", l, blk, s, pg.Geometry, pg.Textures, pw.Geometry, pw.Textures)
				}
				for _, b := range bones[pg.Geometry] {
					if probeBoneShown(t, pg.BoneVisibility[b], s) != probeBoneShown(t, pw.BoneVisibility[b], lo+s) {
						t.Fatalf("layer %d block %d state %d: bone %s differs", l, blk, s, b)
					}
				}
			}
		}
	}
}

// TestStateProbeSplitRegistered: with all4split on (go test with
// SKYGEAR_STATE_PROBE=all4split), the 14 names are custom blocks with their
// own states and permutations, StateProbe blocks round trip through their name
// and state, States() lists the block's own states, and fill puts registered
// blocks.
func TestStateProbeSplitRegistered(t *testing.T) {
	if stateProbeMode != assets.StateProbeAll4Split {
		t.Skipf("probe mode %q", stateProbeMode)
	}
	custom := world.DefaultBlockRegistry.CustomBlocks()
	for _, c := range stateProbeNameCounts(stateProbeMode) {
		cb, ok := custom[c.name]
		if !ok {
			t.Fatalf("%s is not a custom block", c.name)
		}
		p := cb.(block.Permutable)
		if got := len(p.States()[stateProbeState]); got != c.states {
			t.Errorf("%s: States lists %d, want %d", c.name, got, c.states)
		}
		if got := len(p.Permutations()); got != c.permutations {
			t.Errorf("%s: %d permutations, want %d", c.name, got, c.permutations)
		}
	}
	check := func(b world.Block) {
		t.Helper()
		sp, ok := b.(StateProbe)
		if !ok {
			t.Fatalf("%#v is not a probe block", b)
		}
		name, props := b.EncodeBlock()
		got, ok := world.DefaultBlockRegistry.BlockByName(name, props)
		if !ok || got != b {
			t.Fatalf("%s %v: registered as %#v (%v)", name, props, got, ok)
		}
		info, _ := stateProbeInfoOf(b)
		if info.name != name || int32(info.state) != props[stateProbeState] || info.index != info.state || info.count > stateProbeNameCap {
			t.Fatalf("%#v: info %+v for %s %v", sp, info, name, props)
		}
	}
	for x := -12; x < 12; x++ {
		for z := -12; z < 12; z++ {
			c, _, ok := fillLook4(cube.Pos{x, 64, z}).cell(cube.Pos{x, 64, z})
			if !ok {
				t.Fatal("no cell")
			}
			check(c.b0)
			if c.b1 != nil {
				check(c.b1)
			}
		}
	}
}

// logRecords runs f with a logger that writes its lines (message and
// attributes, level first) into the returned buffer.
func logRecords(f func(log *slog.Logger)) string {
	var buf bytes.Buffer
	f(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf.String()
}

// TestStateProbeCapWarning: the log lists each probe block name with its
// states, and warns of each name over 65,536 states: for all4 (2 names), not
// for all4split nor the other modes.
func TestStateProbeCapWarning(t *testing.T) {
	for _, mode := range probeModes {
		out := logRecords(func(log *slog.Logger) { logBlockNameCounts(log, stateProbeNameCounts(mode)) })
		warns, infos := strings.Count(out, "level=WARN"), strings.Count(out, "level=INFO")
		wantWarns := 0
		if mode == assets.StateProbeAll4 {
			wantWarns = 2
		}
		if warns != wantWarns {
			t.Errorf("%s: %d warnings, want %d:\n%s", mode, warns, wantWarns, out)
		}
		if want := len(stateProbeNameCounts(mode)); infos != want {
			t.Errorf("%s: %d name lines, want %d", mode, infos, want)
		}
		if mode == assets.StateProbeAll4 && !strings.Contains(out, "states=221953") {
			t.Errorf("all4: the warning does not give the states:\n%s", out)
		}
	}
	// a block that is not a probe is warned of but not listed
	out := logRecords(func(log *slog.Logger) {
		logBlockNameCounts(log, []blockNameCount{{name: "skygear:big", states: 70000}, {name: "skygear:ok", states: 65536}})
	})
	if strings.Count(out, "level=WARN") != 1 || strings.Contains(out, "level=INFO") || !strings.Contains(out, "skygear:big") {
		t.Errorf("log:\n%s", out)
	}
	// the registry of this run: its warnings are those of its own mode
	out = logRecords(LogCustomBlockCounts)
	if !strings.Contains(out, "max_states_per_name=") {
		t.Errorf("custom blocks line has no max states per name:\n%s", out)
	}
	if got, want := strings.Count(out, "level=WARN"), map[bool]int{false: 0, true: 2}[stateProbeMode == assets.StateProbeAll4]; got != want {
		t.Errorf("mode %q: %d warnings, want %d:\n%s", stateProbeMode, got, want, out)
	}
}

// TestDebugLayersLook: /sgdebug layers look prints the name, state and index
// of the probe block (and the cap) on each layer of the block looked at, and
// says so when there is none.
func TestDebugLayersLook(t *testing.T) {
	if stateProbeMode == assets.StateProbeOff {
		t.Skip("probe off")
	}
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // eye at (0.5, 65.6, 2.5), facing south: the block is 0 65 3 or further
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		run := func() *cmd.Output {
			var o cmd.Output
			debugLayers{adminBase: adminBase{g: s.g}, Mode: "look"}.Run(p, &o, tx)
			return &o
		}
		if o := run(); len(o.Errors()) != 1 {
			t.Fatalf("looking at air: %v %v", o.Errors(), o.Messages())
		}
		pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{0, 1, 1})
		last := stateProbeLayerValues(stateProbeMode, 1) - 1
		tx.SetBlockLayer(pos, StateProbe{Layer: 0, V: 1}, 0)
		tx.SetBlockLayer(pos, StateProbe{Layer: 1, V: last}, 1)
		o := run()
		if len(o.Errors()) != 0 || len(o.Messages()) != 3 {
			t.Fatalf("look: %v %v", o.Errors(), o.Messages())
		}
		info, _ := stateProbeInfoOf(StateProbe{Layer: 1, V: last})
		blk, state := stateProbeLocate(stateProbeMode, 1, last)
		if info.state != state || info.index != state || info.name != stateProbeNames[1][blk] {
			t.Errorf("info %+v", info)
		}
		var lines []string
		for _, m := range o.Messages() {
			lines = append(lines, m.String())
		}
		text := strings.Join(lines, "\n")
		for _, want := range []string{strconv.Itoa(info.index), info.name, strconv.Itoa(info.count)} {
			if !strings.Contains(text, want) {
				t.Errorf("look output misses %q:\n%s", want, text)
			}
		}
		// the index decides the wording: past the cap only where a name has more states
		beyond := info.index >= stateProbeNameCap
		if past := strings.Contains(text, "PAST") || strings.Contains(text, "한도를 넘음"); past != beyond {
			t.Errorf("index %d, output:\n%s", info.index, text)
		}
		clearLayers(tx, pos)
	})
}

// TestStateProbeInfoOf: the index of a probe block among the states of its
// name is its state; other blocks and values out of range have none.
func TestStateProbeInfoOf(t *testing.T) {
	if stateProbeMode == assets.StateProbeOff {
		if _, ok := stateProbeInfoOf(StateProbe{Layer: 0, V: 0}); ok {
			t.Error("info with the probe off")
		}
		return
	}
	if _, ok := stateProbeInfoOf(block.Air{}); ok {
		t.Error("info of air")
	}
	n := stateProbeLayerValues(stateProbeMode, 0)
	if _, ok := stateProbeInfoOf(StateProbe{Layer: 0, V: n}); ok {
		t.Error("info of a value past the last")
	}
	info, ok := stateProbeInfoOf(StateProbe{Layer: 0, V: n - 1})
	if !ok {
		t.Fatal("no info")
	}
	_, size := stateProbeBlockSpan(stateProbeMode, 0, stateProbeBlocks(stateProbeMode, 0)-1)
	if info.index != size-1 || info.count != size {
		t.Errorf("last value: %+v, block of %d states", info, size)
	}
	if stateProbeMode == assets.StateProbeAll4 && info.index < stateProbeNameCap {
		t.Errorf("all4: the last value is at index %d, expected past the cap", info.index)
	}
	if stateProbeMode == assets.StateProbeAll4Split && info.index >= stateProbeNameCap {
		t.Errorf("all4split: the last value is at index %d", info.index)
	}
}
