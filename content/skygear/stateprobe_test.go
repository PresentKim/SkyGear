package skygear

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/assets"
)

// probeNames are the names of the probe blocks.
func probeNames() []string {
	var out []string
	for l := 0; l < stateProbeLayers; l++ {
		out = append(out, fmt.Sprintf("skygear:state_probe_%d", l))
	}
	return out
}

// TestStateProbeOff: baseline A. With SKYGEAR_STATE_PROBE unset or off, no
// probe block is registered, the helper says so, and no state is added.
func TestStateProbeOff(t *testing.T) {
	if stateProbeMode != assets.StateProbeOff {
		t.Skipf("probe mode %q", stateProbeMode)
	}
	if b, ok := stateProbeBlock(0, 1); ok || b != nil {
		t.Errorf("stateProbeBlock = %v, %v with the probe off", b, ok)
	}
	for _, b := range world.Blocks() {
		if _, ok := b.(StateProbe); ok {
			t.Fatalf("%v is registered with the probe off", b)
		}
		if name, _ := b.EncodeBlock(); strings.Contains(name, "state_probe") {
			t.Fatalf("%s is registered with the probe off", name)
		}
	}
	for name := range world.DefaultBlockRegistry.CustomBlocks() {
		if strings.Contains(name, "state_probe") {
			t.Fatalf("custom block %s with the probe off", name)
		}
	}
	if s, p := stateProbeCounts(stateProbeMode); s != 0 || p != 0 {
		t.Errorf("counts %d, %d with the probe off", s, p)
	}
}

// TestStateProbeRegistered: with a mode on (go test with SKYGEAR_STATE_PROBE
// set) both blocks have all 577 states, in the order of their values, and the
// helper gives them.
func TestStateProbeRegistered(t *testing.T) {
	if stateProbeMode == assets.StateProbeOff {
		t.Skip("probe off")
	}
	states := map[string][]any{}
	for _, b := range world.Blocks() {
		if sp, ok := b.(StateProbe); ok {
			name, props := b.EncodeBlock()
			states[name] = append(states[name], props[stateProbeState])
			if got, ok := stateProbeBlock(sp.Layer, sp.V); !ok || got != b {
				t.Errorf("stateProbeBlock(%d, %d) = %v, %v", sp.Layer, sp.V, got, ok)
			}
		}
	}
	for _, name := range probeNames() {
		if len(states[name]) != stateProbeValues {
			t.Errorf("%s: %d states, want %d", name, len(states[name]), stateProbeValues)
		}
	}
	if _, ok := stateProbeBlock(0, stateProbeValues); ok {
		t.Error("value 577 is a probe block")
	}
	if _, ok := stateProbeBlock(stateProbeLayers, 0); ok {
		t.Error("layer 2 is a probe block")
	}
	wantStates, wantPerms := stateProbeCounts(stateProbeMode)
	if wantStates != len(probeNames())*stateProbeValues {
		t.Errorf("%d states", wantStates)
	}
	var perms int
	for _, name := range probeNames() {
		cb, ok := world.DefaultBlockRegistry.CustomBlocks()[name]
		if !ok {
			t.Fatalf("%s is not a custom block", name)
		}
		perms += len(cb.(block.Permutable).Permutations())
	}
	if perms != wantPerms {
		t.Errorf("%d permutations, want %d", perms, wantPerms)
	}
}

func TestStateProbeCounts(t *testing.T) {
	for mode, want := range map[string][2]int{
		assets.StateProbeOff:   {0, 0},
		assets.StateProbePerm:  {1154, 1152},
		assets.StateProbeBones: {1154, 18},
	} {
		if s, p := stateProbeCounts(mode); s != want[0] || p != want[1] {
			t.Errorf("mode %q: %d states, %d permutations, want %v", mode, s, p, want)
		}
		if got := len(stateProbePermutations(mode)); got*stateProbeLayers != want[1] {
			t.Errorf("mode %q: %d permutations a block, want %d", mode, got, want[1]/stateProbeLayers)
		}
	}
}

func TestStateProbeParts(t *testing.T) {
	seen := map[[3]int]bool{}
	for v := 1; v < stateProbeValues; v++ {
		kind, tier, faces := stateProbeParts(v)
		if kind < 0 || kind > 2 || tier < 1 || tier > 3 || faces < 0 || faces > 63 {
			t.Fatalf("v=%d: kind %d tier %d faces %d", v, kind, tier, faces)
		}
		if k := [3]int{kind, tier, faces}; seen[k] {
			t.Fatalf("v=%d repeats %v", v, k)
		} else {
			seen[k] = true
		}
		if 1+kind*192+(tier-1)*64+faces != v {
			t.Fatalf("v=%d does not round-trip", v)
		}
	}
	if k, tr, f := stateProbeParts(1); k != 0 || tr != 1 || f != 0 {
		t.Errorf("v=1 is %d %d %d", k, tr, f)
	}
	if k, tr, f := stateProbeParts(576); k != 2 || tr != 3 || f != 63 {
		t.Errorf("v=576 is %d %d %d", k, tr, f)
	}
	if got := stateProbeLane(0); got != [3]float64{-3.0 / 16, -3.0 / 16, -3.0 / 16} {
		t.Errorf("energy lane %v", got)
	}
	if got := stateProbeLane(1); got != [3]float64{} {
		t.Errorf("item lane %v", got)
	}
	if got := stateProbeLane(2); got != [3]float64{3.0 / 16, 3.0 / 16, 3.0 / 16} {
		t.Errorf("fluid lane %v", got)
	}
}

// probeGeometryBones reads the pack file of a probe geometry: its bones by
// identifier.
func probeGeometryBones(t *testing.T, file string) map[string][]string {
	t.Helper()
	var geo struct {
		Geometry []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name string `json:"name"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/blocks/"+file, &geo)
	out := map[string][]string{}
	for _, g := range geo.Geometry {
		for _, b := range g.Bones {
			out[g.Description.Identifier] = append(out[g.Description.Identifier], b.Name)
		}
		if out[g.Description.Identifier] == nil {
			out[g.Description.Identifier] = []string{}
		}
	}
	return out
}

var (
	probeRange = regexp.MustCompile(`^q\.block_state\('skygear:shape'\) >= (\d+) && q\.block_state\('skygear:shape'\) <= (\d+)$`)
	probeBit   = regexp.MustCompile(`^math\.mod\(math\.floor\(\(q\.block_state\('skygear:shape'\) - (\d+)\) / (\d+)\), 2\) == 1$`)
)

// TestStateProbeModesDrawTheSame: every value 1 to 576 comes out the same in
// both modes: the permutation matching it has the texture and lane of its kind
// and tier, and the arms drawn are the set of its faces, whether the geometry
// is the one of the set (perm) or the bones geometry with the arms its
// bone_visibility expressions switch on (bones, expressions evaluated here).
func TestStateProbeModesDrawTheSame(t *testing.T) {
	shapes := probeGeometryBones(t, "state_probe_shape.geo.json")
	bones := probeGeometryBones(t, "state_probe_bones.geo.json")
	if len(shapes) != 64 || len(bones) != 1 {
		t.Fatalf("%d shape geometries, %d bones geometries", len(shapes), len(bones))
	}
	if got := strings.Join(bones[stateProbeBonesGeometry], ","); got != "core,down,up,north,south,west,east" {
		t.Fatalf("bones %s", got)
	}
	perm := stateProbePermutations(assets.StateProbePerm)
	bonesPerms := stateProbePermutations(assets.StateProbeBones)
	if len(perm) != 576 || len(bonesPerms) != 9 {
		t.Fatalf("%d and %d permutations", len(perm), len(bonesPerms))
	}
	conditions := map[string]bool{}
	for _, p := range perm {
		if conditions[p.Condition] {
			t.Fatalf("condition %s repeats", p.Condition)
		}
		conditions[p.Condition] = true
	}
	for v := 1; v < stateProbeValues; v++ {
		kind, tier, faces := stateProbeParts(v)
		wantTexture := fmt.Sprintf("skygear_conduit_%s_%d", conduitKindNames[kind], tier)
		var want []string
		for i, arm := range stateProbeArms {
			if faces&(1<<i) != 0 {
				want = append(want, arm)
			}
		}
		// B: the permutation of the value
		p := perm[v-1].Properties
		if cond := fmt.Sprintf("q.block_state('skygear:shape') == %d", v); perm[v-1].Condition != cond {
			t.Fatalf("permutation %d: condition %s", v-1, perm[v-1].Condition)
		}
		if got := p.Textures["*"].Encode()["texture"]; got != wantTexture || p.Translation != stateProbeLane(kind) {
			t.Fatalf("v=%d perm: texture %v lane %v", v, got, p.Translation)
		}
		var drawn []string
		for _, b := range shapes[p.Geometry] {
			if b != "core" {
				drawn = append(drawn, b)
			}
		}
		if shapes[p.Geometry] == nil || shapes[p.Geometry][0] != "core" || strings.Join(drawn, ",") != strings.Join(want, ",") {
			t.Fatalf("v=%d perm: geometry %s has %v, want core and %v", v, p.Geometry, shapes[p.Geometry], want)
		}
		// C: the one permutation whose range holds v
		var matches []int
		for i, bp := range bonesPerms {
			m := probeRange.FindStringSubmatch(bp.Condition)
			if m == nil {
				t.Fatalf("condition %q", bp.Condition)
			}
			lo, _ := strconv.Atoi(m[1])
			hi, _ := strconv.Atoi(m[2])
			if v >= lo && v <= hi {
				matches = append(matches, i)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("v=%d matches permutations %v", v, matches)
		}
		bp := bonesPerms[matches[0]].Properties
		if got := bp.Textures["*"].Encode()["texture"]; got != wantTexture || bp.Translation != stateProbeLane(kind) || bp.Geometry != stateProbeBonesGeometry {
			t.Fatalf("v=%d bones: texture %v lane %v geometry %s", v, got, bp.Translation, bp.Geometry)
		}
		if bp.BoneVisibility["core"] != "true" {
			t.Fatalf("v=%d bones: core is %q", v, bp.BoneVisibility["core"])
		}
		var shown []string
		for _, arm := range stateProbeArms {
			m := probeBit.FindStringSubmatch(bp.BoneVisibility[arm])
			if m == nil {
				t.Fatalf("bone_visibility %q", bp.BoneVisibility[arm])
			}
			lo, _ := strconv.Atoi(m[1])
			div, _ := strconv.Atoi(m[2])
			if (v-lo)/div%2 == 1 {
				shown = append(shown, arm)
			}
		}
		if strings.Join(shown, ",") != strings.Join(want, ",") {
			t.Fatalf("v=%d bones: arms %v, want %v", v, shown, want)
		}
	}
}

// TestStateProbeEmptyLook: the value 0 draws nothing in mode bones (every bone
// hidden) and a shrunk core in mode perm.
func TestStateProbeEmptyLook(t *testing.T) {
	b := stateProbeBase(assets.StateProbeBones)
	if len(b.BoneVisibility) != 7 {
		t.Errorf("bones: %v", b.BoneVisibility)
	}
	for bone, v := range b.BoneVisibility {
		if v != "false" {
			t.Errorf("bone %s is %q", bone, v)
		}
	}
	if p := stateProbeBase(assets.StateProbePerm); p.Geometry != stateProbeShapeGeometry(0) || p.Scale[0] >= 0.01 {
		t.Errorf("perm base %+v", p)
	}
}
