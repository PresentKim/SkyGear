package skygear

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/assets"
)

// pair2 is the mode these tests check; they need no environment variable, as
// the functions take the mode (the few that read stateProbeMode get it set by
// withPair2).
const pair2 = assets.StateProbePair2

// withPair2 sets the probe mode to pair2 for a test.
func withPair2(t *testing.T) {
	t.Helper()
	old := stateProbeMode
	stateProbeMode = pair2
	t.Cleanup(func() { stateProbeMode = old })
}

// TestStateProbePair2Counts: 2 names of 37,249 states (under the cap), 15
// permutations each, the 2 pair geometries, 74,498 states.
func TestStateProbePair2Counts(t *testing.T) {
	if stateProbePair2Values != 37249 {
		t.Fatalf("%d values", stateProbePair2Values)
	}
	counts := stateProbeNameCounts(pair2)
	if len(counts) != 2 {
		t.Fatalf("%d names: %v", len(counts), counts)
	}
	for i, c := range counts {
		if c.states != 37249 || c.permutations != 15 || c.states > stateProbeNameCap {
			t.Errorf("name %d: %+v", i, c)
		}
		if !strings.HasPrefix(c.name, "skygear:state_probe") {
			t.Errorf("name %q is not counted as a probe block", c.name)
		}
	}
	if counts[0].name == counts[1].name {
		t.Error("the two blocks share a name")
	}
	if s, p := stateProbeCounts(pair2); s != 74498 || p != 30 {
		t.Errorf("%d states, %d permutations", s, p)
	}
	if g := stateProbeGeometries(pair2); g != 2 {
		t.Errorf("%d geometries", g)
	}
	for l := range stateProbeLayers {
		if stateProbeBlocks(pair2, l) != 1 {
			t.Errorf("block %d: %d names", l, stateProbeBlocks(pair2, l))
		}
		if blk, state := stateProbeLocate(pair2, l, 12345); blk != 0 || state != 12345 {
			t.Errorf("locate = %d %d", blk, state)
		}
	}
}

// TestStateProbePair2RoundTrip: every value of both blocks decodes to a look of
// the block's two kinds that encodes back, no two values share a look, the
// formula is a*193 + b, and what the block cannot draw does not encode.
func TestStateProbePair2RoundTrip(t *testing.T) {
	for l := range stateProbeLayers {
		seen := map[probeLook]bool{}
		for v := range stateProbePair2Values {
			look, ok := stateProbeDecode(pair2, l, v)
			if !ok {
				t.Fatalf("block %d: %d does not decode", l, v)
			}
			if seen[look] {
				t.Fatalf("block %d: %d repeats %+v", l, v, look)
			}
			seen[look] = true
			if got, ok := stateProbeEncode(pair2, l, look); !ok || got != v {
				t.Fatalf("block %d: %d decodes to %+v, encodes to %d %v", l, v, look, got, ok)
			}
			a, b := v/193, v%193
			wantN := 0
			if a > 0 {
				wantN++
			}
			if b > 0 {
				wantN++
			}
			if look.n != wantN {
				t.Fatalf("block %d: %d has %d kinds", l, v, look.n)
			}
			for i, k := range look.k[:look.n] {
				if k.kind/2 != l || k.tier < 1 || k.tier > 3 || k.faces < 0 || k.faces > 63 || (i > 0 && look.k[i-1].kind >= k.kind) {
					t.Fatalf("block %d: %d is %+v", l, v, look)
				}
			}
		}
		if len(seen) != 37249 {
			t.Errorf("block %d: %d looks", l, len(seen))
		}
		for _, v := range []int{-1, 37249} {
			if _, ok := stateProbeDecode(pair2, l, v); ok {
				t.Errorf("block %d: %d decodes", l, v)
			}
		}
	}
	// the formula: energy tier 2 faces 5 is a = 1+64+5, item tier 3 faces 63 is b = 1+128+63
	both := probeLook{n: 2, k: [2]fillKind{{1, 3, 63}, {0, 2, 5}}} // in any order
	if v, ok := stateProbeEncode(pair2, 0, both); !ok || v != 70*193+192 {
		t.Errorf("block 0 pair = %d %v", v, ok)
	}
	if v, ok := stateProbeEncode(pair2, 1, probeLookOf(fillKind{3, 1, 0})); !ok || v != 1 {
		t.Errorf("network alone = %d %v", v, ok)
	}
	if v, ok := stateProbeEncode(pair2, 1, probeLookOf(fillKind{2, 1, 0})); !ok || v != 193 {
		t.Errorf("fluid alone = %d %v", v, ok)
	}
	if v, ok := stateProbeEncode(pair2, 0, probeLook{}); !ok || v != 0 {
		t.Errorf("empty = %d %v", v, ok)
	}
	for _, c := range []struct {
		layer int
		look  probeLook
	}{
		{0, probeLookOf(fillKind{2, 1, 0})},                    // fluid is on block 1
		{1, probeLookOf(fillKind{1, 1, 0})},                    // item is on block 0
		{0, probeLookOf(fillKind{0, 1, 0}, fillKind{0, 2, 0})}, // one kind twice
		{0, probeLookOf(fillKind{0, 4, 0})},                    // tier 4
		{0, probeLookOf(fillKind{0, 1, 64})},                   // faces 64
		{2, probeLookOf(fillKind{0, 1, 0})},                    // no third block
	} {
		if v, ok := stateProbeEncode(pair2, c.layer, c.look); ok {
			t.Errorf("block %d: %+v encodes to %d", c.layer, c.look, v)
		}
	}
	if s := stateProbeLookString(probeLookOf(fillKind{2, 3, 0b000110})); s != "fluid tier 3 up+north" {
		t.Errorf("look string %q", s)
	}
	if s := stateProbeLookString(probeLook{}); s != "nothing" {
		t.Errorf("look string %q", s)
	}
}

var (
	pair2Clause = regexp.MustCompile(`^(.+) (>=|<=|==) (\d+)$`)
	pair2Bit    = regexp.MustCompile(`^math\.mod\(math\.floor\(\((.+) - (\d+)\) / (\d+)\), 2\) == 1$`)
)

// pair2Eval evaluates the Molang the pair2 permutations use on the value v:
// the state, floor(state / 193) and mod(state, 193).
func pair2Eval(t *testing.T, expr string, v int) int {
	t.Helper()
	switch expr {
	case stateProbeShapeExpr:
		return v
	case stateProbePair2A:
		return v / 193
	case stateProbePair2B:
		return v % 193
	}
	t.Fatalf("unknown expression %q", expr)
	return 0
}

// pair2Matches evaluates a condition (clauses joined by &&) on v.
func pair2Matches(t *testing.T, cond string, v int) bool {
	t.Helper()
	for _, c := range strings.Split(cond, " && ") {
		m := pair2Clause.FindStringSubmatch(c)
		if m == nil {
			t.Fatalf("clause %q of %q", c, cond)
		}
		x, n := pair2Eval(t, m[1], v), 0
		n, _ = strconv.Atoi(m[3])
		ok := false
		switch m[2] {
		case ">=":
			ok = x >= n
		case "<=":
			ok = x <= n
		case "==":
			ok = x == n
		}
		if !ok {
			return false
		}
	}
	return true
}

// TestStateProbePair2Draw: in each block, every value but 0 matches exactly one
// of the 15 permutations (and 0 none), whose geometry is the block's pair
// geometry of the pack, whose material instances give each kind the texture of
// its tier, and whose bone_visibility (evaluated here) shows the core and arms
// of each kind present with its faces and nothing else.
func TestStateProbePair2Draw(t *testing.T) {
	bones, instance := probeFourGeometry(t)
	for l := range stateProbeLayers {
		base := stateProbeBaseOf(pair2, l)
		if base.Geometry != stateProbeTwoGeometry(2*l, 2*l+1) || len(bones[base.Geometry]) != 14 {
			t.Fatalf("block %d: base geometry %s (%d bones)", l, base.Geometry, len(bones[base.Geometry]))
		}
		for _, b := range bones[base.Geometry] {
			if base.BoneVisibility[b] != "false" {
				t.Errorf("block %d base: bone %s is %q", l, b, base.BoneVisibility[b])
			}
		}
		perms := stateProbePermutations(pair2, l)
		if len(perms) != 15 {
			t.Fatalf("block %d: %d permutations", l, len(perms))
		}
		for v := range stateProbePair2Values {
			var matches []int
			for i, p := range perms {
				if pair2Matches(t, p.Condition, v) {
					matches = append(matches, i)
				}
			}
			if v == 0 {
				if len(matches) != 0 {
					t.Fatalf("block %d: the value 0 matches %v", l, matches)
				}
				continue
			}
			if len(matches) != 1 {
				t.Fatalf("block %d: %d matches permutations %v", l, v, matches)
			}
			p := perms[matches[0]].Properties
			look, _ := stateProbeDecode(pair2, l, v)
			if p.Geometry != base.Geometry {
				t.Fatalf("block %d: %d geometry %s", l, v, p.Geometry)
			}
			var want []string
			for _, k := range look.k[:look.n] {
				name := stateProbeKindNames4[k.kind]
				if got := p.Textures[name].Encode()["texture"]; got != stateProbeTexture(k.kind, k.tier) {
					t.Fatalf("block %d: %d kind %s texture %v", l, v, name, got)
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
					t.Fatalf("block %d: %d bone %s instance %q has no texture", l, v, b, instance[b])
				}
				switch e := p.BoneVisibility[b]; e {
				case "true":
					shown = append(shown, b)
				case "false":
				default:
					m := pair2Bit.FindStringSubmatch(e)
					if m == nil {
						t.Fatalf("block %d: %d bone %s visibility %q", l, v, b, e)
					}
					lo, _ := strconv.Atoi(m[2])
					div, _ := strconv.Atoi(m[3])
					if (pair2Eval(t, m[1], v)-lo)/div%2 == 1 {
						shown = append(shown, b)
					}
				}
			}
			if strings.Join(shown, ",") != strings.Join(want, ",") {
				t.Fatalf("block %d: %d (%+v) shows %v, want %v", l, v, look, shown, want)
			}
		}
	}
}

// pair2Cells is the cell of a look: the blocks of its layers 0 and 1 as
// (block, value), (-1, 0) for none.
func pair2Cells(t *testing.T, l fillShape) (layers [2][2]int, lost int) {
	t.Helper()
	c, lost, ok := l.cellPair2(cube.Pos{})
	if !ok {
		t.Fatalf("%+v: no cell", l)
	}
	layers = [2][2]int{{-1, 0}, {-1, 0}}
	if c.b0 != nil {
		sp := c.b0.(StateProbe)
		layers[0] = [2]int{sp.Layer, sp.V}
	}
	if c.b1 != nil {
		sp := c.b1.(StateProbe)
		layers[1] = [2]int{sp.Layer, sp.V}
	}
	return layers, lost
}

// TestStateProbePair2Placement: the non-empty blocks fill layer 0 first, then
// layer 1, block 0 before block 1, so block 1 alone is on layer 0; every look
// of 1 to 4 kinds is drawn whole, whatever its tiers and faces.
func TestStateProbePair2Placement(t *testing.T) {
	withPair2(t)
	k := func(kind int) fillKind { return fillKind{kind: kind, tier: 1 + kind%3, faces: 7 * (kind + 1)} }
	for set := 1; set < 16; set++ {
		var l fillShape
		var a, b []fillKind
		for kind := range 4 {
			if set&(1<<kind) != 0 {
				l.k[l.n] = k(kind)
				l.n++
				if kind < 2 {
					a = append(a, k(kind))
				} else {
					b = append(b, k(kind))
				}
			}
		}
		layers, lost := pair2Cells(t, l)
		if lost != 0 {
			t.Errorf("%+v: %d lost", l, lost)
		}
		var want [2]int
		switch {
		case len(a) > 0 && len(b) > 0:
			want = [2]int{0, 1}
		case len(a) > 0:
			want = [2]int{0, -1}
		default:
			want = [2]int{1, -1} // block 1 alone is on layer 0
		}
		if layers[0][0] != want[0] || layers[1][0] != want[1] {
			t.Errorf("%+v: blocks %d and %d on layers 0 and 1, want %v", l, layers[0][0], layers[1][0], want)
		}
		var got []fillKind
		for i, la := range layers {
			if la[0] < 0 {
				continue
			}
			look, _ := stateProbeDecode(pair2, la[0], la[1])
			got = append(got, look.k[:look.n]...)
			if la[1] == 0 {
				t.Errorf("%+v: the value 0 is placed on layer %d", l, i)
			}
		}
		if len(got) != l.n {
			t.Errorf("%+v: draws %v", l, got)
		}
	}
}

// TestStateProbePair2FillCells: every cell fillLook gives over a region draws
// whole (no conduit lost), with the layer rule, in mode pair2.
func TestStateProbePair2FillCells(t *testing.T) {
	withPair2(t)
	counts := map[string]int{}
	for x := -20; x < 20; x++ {
		for y := 60; y < 64; y++ {
			for z := -20; z < 20; z++ {
				pos := cube.Pos{x, y, z}
				c, lost, ok := fillLook(pos).cell(pos)
				if !ok || lost != 0 {
					t.Fatalf("%v: cell %v lost %d", pos, ok, lost)
				}
				b0, b1 := c.b0.(StateProbe), StateProbe{Layer: -1}
				if c.b1 != nil {
					b1 = c.b1.(StateProbe)
					if b0.Layer != 0 || b1.Layer != 1 {
						t.Fatalf("%v: blocks %d and %d on layers 0 and 1", pos, b0.Layer, b1.Layer)
					}
				}
				counts[fmt.Sprint(b0.Layer, b1.Layer)]++
			}
		}
	}
	for _, key := range []string{"0 -1", "1 -1", "0 1"} {
		if counts[key] == 0 {
			t.Errorf("no cell with blocks %s: %v", key, counts)
		}
	}
	if len(counts) != 3 {
		t.Errorf("cells: %v", counts)
	}
}
