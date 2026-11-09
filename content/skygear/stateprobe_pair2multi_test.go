package skygear

import (
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/assets"
)

// The several state variants of pair2 (pair2ab, pair2bits). As with pair2 the
// functions take the mode; withProbeMode sets stateProbeMode for those that
// read it.

var multiModes = []string{assets.StateProbePair2AB, assets.StateProbePair2Bits}

// withProbeMode sets the probe mode for a test.
func withProbeMode(t *testing.T, mode string) {
	t.Helper()
	old := stateProbeMode
	stateProbeMode = mode
	t.Cleanup(func() { stateProbeMode = old })
}

// TestStateProbeMultiCounts: 2 names each, 37,249 (pair2ab) or 65,536
// (pair2bits, the whole of the cap and not past it) states a name, which is
// the product of its States, 15 permutations a name, the 2 pair geometries;
// the log lists both names and warns of none.
func TestStateProbeMultiCounts(t *testing.T) {
	for _, c := range []struct {
		mode              string
		perName, props    int
		states, totalPerm int
	}{
		{assets.StateProbePair2AB, 37249, 2, 74498, 30},
		{assets.StateProbePair2Bits, 65536, 14, 131072, 30},
	} {
		counts := stateProbeNameCounts(c.mode)
		if len(counts) != 2 || counts[0].name == counts[1].name {
			t.Fatalf("%s: names %v", c.mode, counts)
		}
		for l, n := range counts {
			if n.states != c.perName || n.permutations != 15 || n.states > stateProbeNameCap || !strings.HasPrefix(n.name, "skygear:state_probe") {
				t.Errorf("%s: name %d %+v", c.mode, l, n)
			}
			states := stateProbeStatesOf(c.mode, l, 0)
			product := 1
			for _, values := range states {
				product *= len(values)
			}
			if len(states) != c.props || product != c.perName || stateProbeLayerValues(c.mode, l) != c.perName {
				t.Errorf("%s/%d: %d states of %d values, %d values", c.mode, l, len(states), product, stateProbeLayerValues(c.mode, l))
			}
			if stateProbeBlocks(c.mode, l) != 1 {
				t.Errorf("%s/%d: %d names", c.mode, l, stateProbeBlocks(c.mode, l))
			}
		}
		if s, p := stateProbeCounts(c.mode); s != c.states || p != c.totalPerm {
			t.Errorf("%s: %d states, %d permutations", c.mode, s, p)
		}
		if g := stateProbeGeometries(c.mode); g != 2 {
			t.Errorf("%s: %d geometries", c.mode, g)
		}
		out := logRecords(func(log *slog.Logger) { logBlockNameCounts(log, counts) })
		if strings.Count(out, "level=WARN") != 0 || strings.Count(out, "level=INFO") != 2 || !strings.Contains(out, fmt.Sprintf("states=%d", c.perName)) {
			t.Errorf("%s: log\n%s", c.mode, out)
		}
	}
	// the last state of pair2bits is index 65,535, the last the client knows
	if _, ok := stateProbeDecode(assets.StateProbePair2Bits, 0, stateProbeNameCap-1); !ok {
		t.Error("pair2bits: 65,535 does not decode")
	}
	if _, ok := stateProbeDecode(assets.StateProbePair2Bits, 0, stateProbeNameCap); ok {
		t.Error("pair2bits: 65,536 decodes")
	}
	// the other modes keep their one state
	for _, mode := range append(append([]string{assets.StateProbeOff}, probeModes...), assets.StateProbePair2) {
		if stateProbeMultiState(mode) || len(stateProbeStatesOf(mode, 0, 0)) != 1 || len(stateProbeStateProps(mode, 0, 3)) != 1 {
			t.Errorf("%q has several states", mode)
		}
	}
}

// clientStateIndex is the index the client gives a combination of states:
// the names sorted, the first changing slowest, each value counted by its
// place in the enum (States); -1 when a value is not in it.
func clientStateIndex(states map[string][]any, props map[string]any) int {
	if len(props) != len(states) {
		return -1
	}
	idx := 0
	for _, name := range slices.Sorted(maps.Keys(states)) {
		at := slices.Index(states[name], props[name])
		if at < 0 {
			return -1
		}
		idx = idx*len(states[name]) + at
	}
	return idx
}

// TestStateProbeMultiStateOrder: the Nth state registered (the server's
// order: V) has the states the client puts at index N, for every state of both
// blocks of both modes, with values of the types of the enums.
func TestStateProbeMultiStateOrder(t *testing.T) {
	for _, mode := range multiModes {
		for l := range stateProbeLayers {
			states := stateProbeStatesOf(mode, l, 0)
			for v := range stateProbeLayerValues(mode, l) {
				props := stateProbeStateProps(mode, l, v)
				if got := clientStateIndex(states, props); got != v {
					t.Fatalf("%s/%d: state %d %v is index %d for the client", mode, l, v, props, got)
				}
			}
		}
	}
	// the orders written out: pair2ab a then b; pair2bits the names sorted
	if p := stateProbeStateProps(assets.StateProbePair2AB, 0, 70*193+192); !reflect.DeepEqual(p, map[string]any{"skygear:a": int32(70), "skygear:b": int32(192)}) {
		t.Errorf("pair2ab props %v", p)
	}
	var names []string
	for _, p := range stateProbeBitsProps[1] {
		names = append(names, strings.TrimPrefix(p.name, "skygear:"))
	}
	if got := strings.Join(names, ","); got != "fluid_down,fluid_east,fluid_north,fluid_south,fluid_tier,fluid_up,fluid_west,network_down,network_east,network_north,network_south,network_tier,network_up,network_west" {
		t.Errorf("pair2bits order %s", got)
	}
	// west of the second kind changes fastest, down of the first slowest
	if p := stateProbeStateProps(assets.StateProbePair2Bits, 0, 1); p["skygear:item_west"] != true || p["skygear:energy_down"] != false {
		t.Errorf("state 1: %v", p)
	}
	if p := stateProbeStateProps(assets.StateProbePair2Bits, 0, 32768); p["skygear:energy_down"] != true || p["skygear:item_west"] != false {
		t.Errorf("state 32768: %v", p)
	}
}

// TestStateProbeMultiRoundTrip: the looks of both modes are pair2's: every
// look of pair2 encodes in the mode to a state that decodes back to it (in
// pair2ab to the same value); pair2bits has 37,249 placed states, and its other
// states (a face of an absent kind true) decode to the look without the kind.
func TestStateProbeMultiRoundTrip(t *testing.T) {
	for _, mode := range multiModes {
		for l := range stateProbeLayers {
			used := map[int]bool{}
			for v := range stateProbePair2Values {
				look, _ := stateProbeDecode(pair2, l, v)
				s, ok := stateProbeEncode(mode, l, look)
				if !ok {
					t.Fatalf("%s/%d: %+v does not encode", mode, l, look)
				}
				if mode == assets.StateProbePair2AB && s != v {
					t.Fatalf("%s/%d: %+v is %d, pair2 %d", mode, l, look, s, v)
				}
				back, ok := stateProbeDecode(mode, l, s)
				if !ok || back != look {
					t.Fatalf("%s/%d: %d decodes to %+v, want %+v", mode, l, s, back, look)
				}
				used[s] = true
			}
			if len(used) != stateProbePair2Values {
				t.Errorf("%s/%d: %d states placed", mode, l, len(used))
			}
			n := stateProbeLayerValues(mode, l)
			for s := range n {
				look, ok := stateProbeDecode(mode, l, s)
				if !ok {
					t.Fatalf("%s/%d: %d does not decode", mode, l, s)
				}
				if back, _ := stateProbeEncode(mode, l, look); (back == s) != used[s] {
					t.Fatalf("%s/%d: %d decodes to %+v, which encodes to %d", mode, l, s, look, back)
				}
			}
			for _, s := range []int{-1, n} {
				if _, ok := stateProbeDecode(mode, l, s); ok {
					t.Errorf("%s/%d: %d decodes", mode, l, s)
				}
			}
			for _, look := range []probeLook{
				probeLookOf(fillKind{2 - 2*l, 1, 0}), // a kind of the other block
				probeLookOf(fillKind{2 * l, 1, 0}, fillKind{2 * l, 2, 0}),
				probeLookOf(fillKind{2 * l, 4, 0}),
				probeLookOf(fillKind{2 * l, 1, 64}),
			} {
				if s, ok := stateProbeEncode(mode, l, look); ok {
					t.Errorf("%s/%d: %+v encodes to %d", mode, l, look, s)
				}
			}
		}
	}
}

var (
	multiState = regexp.MustCompile(`^q\.block_state\('([a-z_:]+)'\)$`)
	multiBit   = regexp.MustCompile(`^math\.mod\(math\.floor\(\(q\.block_state\('([a-z_:]+)'\) - (\d+)\) / (\d+)\), 2\) == 1$`)
)

// multiStateValue is q.block_state of a state: the int, or 1 and 0 for a bool.
func multiStateValue(t *testing.T, props map[string]any, name string) int {
	t.Helper()
	switch v := props[name].(type) {
	case int32:
		return int(v)
	case bool:
		if v {
			return 1
		}
		return 0
	}
	t.Fatalf("no state %q in %v", name, props)
	return 0
}

// multiCond compiles a condition of the several state modes (clauses of
// q.block_state('x') >=, <= or == a number, joined by &&).
func multiCond(t *testing.T, cond string) func(map[string]any) bool {
	t.Helper()
	type clause struct {
		name, op string
		n        int
	}
	var cs []clause
	for _, c := range strings.Split(cond, " && ") {
		m := pair2Clause.FindStringSubmatch(c)
		if m == nil {
			t.Fatalf("clause %q of %q", c, cond)
		}
		s := multiState.FindStringSubmatch(m[1])
		if s == nil {
			t.Fatalf("not a plain state in %q", c)
		}
		n, _ := strconv.Atoi(m[3])
		cs = append(cs, clause{s[1], m[2], n})
	}
	return func(props map[string]any) bool {
		for _, c := range cs {
			x := multiStateValue(t, props, c.name)
			if !map[string]bool{">=": x >= c.n, "<=": x <= c.n, "==": x == c.n}[c.op] {
				return false
			}
		}
		return true
	}
}

// multiShown evaluates a bone_visibility of the several state modes: true,
// false, a bool state (pair2bits) or a bit of an int state (pair2ab).
func multiShown(t *testing.T, mode, e string, props map[string]any) bool {
	t.Helper()
	switch e {
	case "true":
		return true
	case "false":
		return false
	}
	if m := multiState.FindStringSubmatch(e); m != nil && mode == assets.StateProbePair2Bits {
		if _, ok := props[m[1]].(bool); !ok {
			t.Fatalf("%s: %q is not a bool state", mode, e)
		}
		return multiStateValue(t, props, m[1]) == 1
	}
	if m := multiBit.FindStringSubmatch(e); m != nil && mode == assets.StateProbePair2AB {
		lo, _ := strconv.Atoi(m[2])
		div, _ := strconv.Atoi(m[3])
		return (multiStateValue(t, props, m[1])-lo)/div%2 == 1
	}
	t.Fatalf("%s: visibility %q", mode, e)
	return false
}

// TestStateProbeMultiDraw: in both modes, every state of each block (its
// states evaluated in the Molang of the conditions and bone_visibility)
// matches one of the 15 permutations but those of no kind (none), with the
// block's pair geometry, the texture of each kind's tier and the core and arms
// of each kind present shown and nothing else: what pair2 draws. The
// conditions have no arithmetic, nor (pair2bits) does bone_visibility.
func TestStateProbeMultiDraw(t *testing.T) {
	bones, instance := probeFourGeometry(t)
	for _, mode := range multiModes {
		for l := range stateProbeLayers {
			base := stateProbeBaseOf(mode, l)
			if !reflect.DeepEqual(base, stateProbeBaseOf(pair2, l)) {
				t.Fatalf("%s/%d: base differs from pair2", mode, l)
			}
			perms := stateProbePermutations(mode, l)
			if len(perms) != 15 {
				t.Fatalf("%s/%d: %d permutations", mode, l, len(perms))
			}
			conds := make([]func(map[string]any) bool, len(perms))
			for i, p := range perms {
				if strings.Contains(p.Condition, "math.") {
					t.Errorf("%s/%d: condition %q has arithmetic", mode, l, p.Condition)
				}
				for b, e := range p.Properties.BoneVisibility {
					if mode == assets.StateProbePair2Bits && strings.Contains(e, "math.") {
						t.Errorf("%s/%d: bone %s %q has arithmetic", mode, l, b, e)
					}
				}
				conds[i] = multiCond(t, p.Condition)
			}
			for s := range stateProbeLayerValues(mode, l) {
				props := stateProbeStateProps(mode, l, s)
				look, _ := stateProbeDecode(mode, l, s)
				var matches []int
				for i, c := range conds {
					if c(props) {
						matches = append(matches, i)
					}
				}
				if look.n == 0 {
					if len(matches) != 0 {
						t.Fatalf("%s/%d: %d (no kind) matches %v", mode, l, s, matches)
					}
					continue
				}
				if len(matches) != 1 {
					t.Fatalf("%s/%d: %d %v matches %v", mode, l, s, props, matches)
				}
				p := perms[matches[0]].Properties
				if p.Geometry != base.Geometry {
					t.Fatalf("%s/%d: %d geometry %s", mode, l, s, p.Geometry)
				}
				var want []string
				for _, k := range look.k[:look.n] {
					name := stateProbeKindNames4[k.kind]
					if got := p.Textures[name].Encode()["texture"]; got != stateProbeTexture(k.kind, k.tier) {
						t.Fatalf("%s/%d: %d kind %s texture %v", mode, l, s, name, got)
					}
					want = append(want, name+"_core")
					for i, arm := range stateProbeArms {
						if k.faces&(1<<i) != 0 {
							want = append(want, name+"_"+arm)
						}
					}
				}
				var shown []string
				for _, b := range bones[p.Geometry] {
					if _, ok := p.Textures[instance[b]]; !ok {
						t.Fatalf("%s/%d: bone %s has no texture", mode, l, b)
					}
					if multiShown(t, mode, p.BoneVisibility[b], props) {
						shown = append(shown, b)
					}
				}
				if strings.Join(shown, ",") != strings.Join(want, ",") {
					t.Fatalf("%s/%d: %d (%+v) shows %v, want %v", mode, l, s, look, shown, want)
				}
			}
		}
	}
}

// cellLooks is what a cell puts: for layers 0 and 1 the block (-1 none) and
// the look it draws.
func cellLooks(c fillCell, mode string) (out [2]struct {
	blk  int
	look probeLook
}) {
	for i, b := range []world.Block{c.b0, c.b1} {
		out[i].blk = -1
		if b != nil {
			sp := b.(StateProbe)
			out[i].blk = sp.Layer
			out[i].look, _ = stateProbeDecode(mode, sp.Layer, sp.V)
		}
	}
	return out
}

// TestStateProbeMultiPlacement: fill (block mode) puts in every cell of a
// region the same blocks on the same layers, drawing the same looks, as pair2,
// with no conduit lost, and the three placements of pair2 all occur.
func TestStateProbeMultiPlacement(t *testing.T) {
	type cell struct {
		pos  cube.Pos
		want [2]struct {
			blk  int
			look probeLook
		}
	}
	var cells []cell
	withProbeMode(t, pair2)
	for x := -16; x < 16; x++ {
		for y := 60; y < 64; y++ {
			for z := -16; z < 16; z++ {
				pos := cube.Pos{x, y, z}
				c, lost, ok := fillLook(pos).cell(pos)
				if !ok || lost != 0 {
					t.Fatalf("pair2 %v: %v %d", pos, ok, lost)
				}
				cells = append(cells, cell{pos, cellLooks(c, pair2)})
			}
		}
	}
	for _, mode := range multiModes {
		withProbeMode(t, mode)
		placed := map[[2]int]int{}
		for _, c := range cells {
			got, lost, ok := fillLook(c.pos).cell(c.pos)
			if !ok || lost != 0 {
				t.Fatalf("%s %v: %v lost %d", mode, c.pos, ok, lost)
			}
			if looks := cellLooks(got, mode); looks != c.want {
				t.Fatalf("%s %v: %+v, pair2 %+v", mode, c.pos, looks, c.want)
			}
			placed[[2]int{c.want[0].blk, c.want[1].blk}]++
		}
		if len(placed) != 3 || placed[[2]int{0, 1}] == 0 || placed[[2]int{1, -1}] == 0 || placed[[2]int{0, -1}] == 0 {
			t.Errorf("%s: placements %v", mode, placed)
		}
	}
}

// TestStateProbeMultiRegistered: with the mode on (go test with
// SKYGEAR_STATE_PROBE=pair2ab or pair2bits) the registry has every state of
// both names in the order the client numbers them (the definition the client
// gets, States, sorted by name), each a StateProbe that round trips through
// its name and states.
func TestStateProbeMultiRegistered(t *testing.T) {
	if !stateProbeMultiState(stateProbeMode) {
		t.Skipf("probe mode %q", stateProbeMode)
	}
	byName := map[string][]world.Block{}
	for _, b := range world.Blocks() {
		if _, ok := b.(StateProbe); ok {
			name, _ := b.EncodeBlock()
			byName[name] = append(byName[name], b)
		}
	}
	custom := world.DefaultBlockRegistry.CustomBlocks()
	for l, c := range stateProbeNameCounts(stateProbeMode) {
		states := custom[c.name].(block.Permutable).States()
		if got := byName[c.name]; len(got) != c.states {
			t.Fatalf("%s: %d states registered, want %d", c.name, len(got), c.states)
		}
		for i, b := range byName[c.name] {
			name, props := b.EncodeBlock()
			if idx := clientStateIndex(states, props); idx != i || b.(StateProbe).V != i || b.(StateProbe).Layer != l {
				t.Fatalf("%s: state %d registered is %#v %v, client index %d", c.name, i, b, props, idx)
			}
			if i%997 == 0 || i == c.states-1 {
				if got, ok := world.DefaultBlockRegistry.BlockByName(name, props); !ok || got != b {
					t.Fatalf("%s %v: registered as %#v", name, props, got)
				}
			}
		}
	}
}
