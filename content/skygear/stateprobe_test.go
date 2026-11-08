package skygear

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/assets"
)

// probeNames are the names of the probe blocks.
func probeNames() []string {
	var out []string
	for l := 0; l < stateProbeLayers; l++ {
		out = append(out, stateProbeName(stateProbeMode, l))
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
// set) both blocks have all their states (577 in perm and bones), in the
// order of their values, and the helper gives them.
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
	sum := 0
	for l, name := range probeNames() {
		n := stateProbeLayerValues(stateProbeMode, l)
		sum += n
		if len(states[name]) != n {
			t.Errorf("%s: %d states, want %d", name, len(states[name]), n)
		}
		if _, ok := stateProbeBlock(l, n); ok {
			t.Errorf("value %d is a probe block of layer %d", n, l)
		}
	}
	if _, ok := stateProbeBlock(stateProbeLayers, 0); ok {
		t.Error("layer 2 is a probe block")
	}
	wantStates, wantPerms := stateProbeCounts(stateProbeMode)
	if wantStates != sum {
		t.Errorf("%d states, want %d", wantStates, sum)
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

// TestStateProbeCounts: states (layer 0, layer 1), permutations (layer 0,
// layer 1) and geometries of each mode.
func TestStateProbeCounts(t *testing.T) {
	for mode, want := range map[string][5]int{
		assets.StateProbeOff:     {0, 0, 0, 0, 0},
		assets.StateProbePerm:    {577, 577, 577, 577, 192},
		assets.StateProbeBones:   {577, 577, 9, 9, 3},
		assets.StateProbeSingle4: {769, 769, 12, 12, 4},
		assets.StateProbeSame4:   {769, 769 + 3456, 12, 12 + 54, 10},
		assets.StateProbeAll4:    {769 + 221184, 769 + 221184, 12 + 54, 12 + 54, 10},
	} {
		if s, p := stateProbeCounts(mode); s != want[0]+want[1] || p != want[2]+want[3] {
			t.Errorf("mode %q: %d states, %d permutations, want %v", mode, s, p, want)
		}
		for l := range stateProbeLayers {
			if got := stateProbeLayerValues(mode, l); got != want[l] {
				t.Errorf("mode %q layer %d: %d values, want %d", mode, l, got, want[l])
			}
			if mode == assets.StateProbeOff {
				continue
			}
			if got := len(stateProbePermutations(mode, l)); got != want[2+l] {
				t.Errorf("mode %q layer %d: %d permutations, want %d", mode, l, got, want[2+l])
			}
		}
		if got := stateProbeGeometries(mode); got != want[4] {
			t.Errorf("mode %q: %d geometries, want %d", mode, got, want[4])
		}
	}
	if s, _ := stateProbeCounts(assets.StateProbeAll4); s != 443906 {
		t.Errorf("all4: %d states", s)
	}
}

// probeModes are the modes with the probe on.
var probeModes = []string{assets.StateProbePerm, assets.StateProbeBones, assets.StateProbeSingle4, assets.StateProbeSame4, assets.StateProbeAll4}

// probeSample are the values of a block to check: all of them, or for a large
// block every 97th and the ends of every range of 64.
func probeSample(n int) []int {
	if n <= 5000 {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	var out []int
	for v := 0; v < n; v++ {
		if v < stateProbePairStart || v%97 == 0 || (v-stateProbePairStart)%64 == 0 || (v-stateProbePairStart)%64 == 63 {
			out = append(out, v)
		}
	}
	return out
}

// TestStateProbeRoundTrip: every value (sampled for all4) decodes to a look
// that encodes back to it, with kinds in order and of the mode, pairs only
// where the layer has them (same faces in same4), and values out of range do
// not decode.
func TestStateProbeRoundTrip(t *testing.T) {
	for _, mode := range probeModes {
		for l := range stateProbeLayers {
			n := stateProbeLayerValues(mode, l)
			pairs := stateProbeLayerPairs(mode, l)
			seen := map[probeLook]bool{}
			for _, v := range probeSample(n) {
				look, ok := stateProbeDecode(mode, l, v)
				if !ok {
					t.Fatalf("%s/%d: %d does not decode", mode, l, v)
				}
				if seen[look] {
					t.Fatalf("%s/%d: %d repeats %+v", mode, l, v, look)
				}
				seen[look] = true
				if got, ok := stateProbeEncode(mode, l, look); !ok || got != v {
					t.Fatalf("%s/%d: %d decodes to %+v, encodes to %d %v", mode, l, v, look, got, ok)
				}
				wantN := 1
				switch {
				case v == 0:
					wantN = 0
				case v >= stateProbePairStart:
					wantN = 2
				}
				if look.n != wantN {
					t.Fatalf("%s/%d: %d has %d kinds", mode, l, v, look.n)
				}
				for _, k := range look.k[:look.n] {
					if k.kind < 0 || k.kind >= stateProbeModeKinds(mode) || k.tier < 1 || k.tier > 3 || k.faces < 0 || k.faces > 63 {
						t.Fatalf("%s/%d: %d is %+v", mode, l, v, look)
					}
				}
				if look.n == 2 {
					if pairs == probeNoPairs || look.k[0].kind >= look.k[1].kind {
						t.Fatalf("%s/%d: %d is %+v", mode, l, v, look)
					}
					if pairs == probeSamePairs && look.k[0].faces != look.k[1].faces {
						t.Fatalf("%s/%d: %d has different faces", mode, l, v)
					}
				}
			}
			if n <= 5000 && len(seen) != n {
				t.Errorf("%s/%d: %d looks of %d values", mode, l, len(seen), n)
			}
			for _, v := range []int{-1, n} {
				if _, ok := stateProbeDecode(mode, l, v); ok {
					t.Errorf("%s/%d: %d decodes", mode, l, v)
				}
			}
		}
	}
	// the documented formulas
	ab := probeLook{n: 2, k: [2]fillKind{{3, 2, 5}, {1, 3, 40}}} // item t3 f40 + network t2 f5: pair 13 is 4
	if v, ok := stateProbeEncode(assets.StateProbeAll4, 0, ab); !ok || v != 769+((4*9+2*3+1)*64+40)*64+5 {
		t.Errorf("all4 pair = %d %v", v, ok)
	}
	same := probeLook{n: 2, k: [2]fillKind{{0, 1, 9}, {2, 3, 9}}} // pair 02 is 1
	if v, ok := stateProbeEncode(assets.StateProbeSame4, 1, same); !ok || v != 769+(1*9+0*3+2)*64+9 {
		t.Errorf("same4 pair = %d %v", v, ok)
	}
	for _, c := range []struct {
		mode  string
		layer int
		look  probeLook
	}{
		{assets.StateProbeSame4, 0, same},                                                 // no pairs on layer 0
		{assets.StateProbeSame4, 1, ab},                                                   // faces differ
		{assets.StateProbeSingle4, 0, same},                                               // no pairs
		{assets.StateProbeBones, 0, probeLookOf(ab.k[0])},                                 // no network
		{assets.StateProbeAll4, 0, probeLook{n: 2, k: [2]fillKind{{1, 1, 0}, {1, 2, 0}}}}, // one kind twice
	} {
		if v, ok := stateProbeEncode(c.mode, c.layer, c.look); ok {
			t.Errorf("%s/%d: %+v encodes to %d", c.mode, c.layer, c.look, v)
		}
	}
}

// probeFourGeometry reads state_probe_four.geo.json: the bones of each
// geometry and the material instance of each bone's faces.
func probeFourGeometry(t *testing.T) (bones map[string][]string, instance map[string]string) {
	t.Helper()
	var geo struct {
		Geometry []struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
			Bones []struct {
				Name  string `json:"name"`
				Cubes []struct {
					UV map[string]struct {
						MaterialInstance string `json:"material_instance"`
					} `json:"uv"`
				} `json:"cubes"`
			} `json:"bones"`
		} `json:"minecraft:geometry"`
	}
	readPackJSON(t, "models/blocks/state_probe_four.geo.json", &geo)
	bones, instance = map[string][]string{}, map[string]string{}
	for _, g := range geo.Geometry {
		for _, b := range g.Bones {
			bones[g.Description.Identifier] = append(bones[g.Description.Identifier], b.Name)
			for _, c := range b.Cubes {
				for _, f := range c.UV {
					if prev, ok := instance[b.Name]; ok && prev != f.MaterialInstance {
						t.Fatalf("bone %s has faces of %s and %s", b.Name, prev, f.MaterialInstance)
					}
					instance[b.Name] = f.MaterialInstance
				}
			}
		}
	}
	return bones, instance
}

// TestStateProbeFourKindsDraw: in each 4-kind mode and layer, every value
// (sampled for all4) matches exactly one permutation, whose geometry has the
// bones of the value's kinds, whose material instances give each kind the
// texture of its tier, and whose bone_visibility (evaluated here) shows the
// cores and the arms of each kind's faces and nothing else.
func TestStateProbeFourKindsDraw(t *testing.T) {
	bones, instance := probeFourGeometry(t)
	if len(bones) != 10 {
		t.Fatalf("%d geometries", len(bones))
	}
	for _, mode := range []string{assets.StateProbeSingle4, assets.StateProbeSame4, assets.StateProbeAll4} {
		base := stateProbeBase(mode)
		for _, b := range bones[base.Geometry] {
			if base.BoneVisibility[b] != "false" {
				t.Errorf("%s base: bone %s is %q", mode, b, base.BoneVisibility[b])
			}
		}
		for l := range stateProbeLayers {
			perms := stateProbePermutations(mode, l)
			type span struct{ lo, hi int }
			ranges := make([]span, len(perms))
			for i, p := range perms {
				m := probeRange.FindStringSubmatch(p.Condition)
				if m == nil {
					t.Fatalf("condition %q", p.Condition)
				}
				ranges[i].lo, _ = strconv.Atoi(m[1])
				ranges[i].hi, _ = strconv.Atoi(m[2])
			}
			for _, v := range probeSample(stateProbeLayerValues(mode, l)) {
				if v == 0 {
					continue
				}
				look, _ := stateProbeDecode(mode, l, v)
				var matches []int
				for i, r := range ranges {
					if v >= r.lo && v <= r.hi {
						matches = append(matches, i)
					}
				}
				if len(matches) != 1 {
					t.Fatalf("%s/%d: %d matches permutations %v", mode, l, v, matches)
				}
				p := perms[matches[0]].Properties
				geoBones, ok := bones[p.Geometry]
				if !ok {
					t.Fatalf("%s/%d: %d geometry %s not in the pack", mode, l, v, p.Geometry)
				}
				var want []string
				for _, k := range look.k[:look.n] {
					name := stateProbeKindNames4[k.kind]
					if got := p.Textures[name].Encode()["texture"]; got != stateProbeTexture(k.kind, k.tier) {
						t.Fatalf("%s/%d: %d kind %s texture %v", mode, l, v, name, got)
					}
					want = append(want, name+"_core")
					for i, arm := range stateProbeArms {
						if k.faces&(1<<i) != 0 {
							want = append(want, name+"_"+arm)
						}
					}
				}
				if len(p.Textures) != look.n+1 {
					t.Fatalf("%s/%d: %d textures %v", mode, l, v, p.Textures)
				}
				var shown []string
				for _, b := range geoBones {
					if _, ok := p.Textures[instance[b]]; !ok {
						t.Fatalf("%s/%d: %d bone %s instance %q has no texture", mode, l, v, b, instance[b])
					}
					switch e := p.BoneVisibility[b]; e {
					case "true":
						shown = append(shown, b)
					default:
						m := probeBit.FindStringSubmatch(e)
						if m == nil {
							t.Fatalf("%s/%d: %d bone %s visibility %q", mode, l, v, b, e)
						}
						lo, _ := strconv.Atoi(m[1])
						div, _ := strconv.Atoi(m[2])
						if (v-lo)/div%2 == 1 {
							shown = append(shown, b)
						}
					}
				}
				if strings.Join(shown, ",") != strings.Join(want, ",") {
					t.Fatalf("%s/%d: %d (%+v) shows %v, want %v", mode, l, v, look, shown, want)
				}
			}
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
	if len(shapes) != 3*64 || len(bones) != 3 {
		t.Fatalf("%d shape geometries, %d bones geometries", len(shapes), len(bones))
	}
	for kind := range conduitKindNames {
		if got := strings.Join(bones[stateProbeBonesGeometry(kind)], ","); got != "core,down,up,north,south,west,east" {
			t.Fatalf("bones %s", got)
		}
	}
	perm := stateProbePermutations(assets.StateProbePerm, 0)
	bonesPerms := stateProbePermutations(assets.StateProbeBones, 0)
	if len(perm) != 577 || len(bonesPerms) != 9 {
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
		p := perm[v].Properties
		if cond := fmt.Sprintf("q.block_state('skygear:shape') == %d", v); perm[v].Condition != cond {
			t.Fatalf("permutation %d: condition %s", v, perm[v].Condition)
		}
		if got := p.Textures["*"].Encode()["texture"]; got != wantTexture || p.Geometry != stateProbeShapeGeometry(kind, faces) {
			t.Fatalf("v=%d perm: texture %v geometry %s", v, got, p.Geometry)
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
		if got := bp.Textures["*"].Encode()["texture"]; got != wantTexture || bp.Geometry != stateProbeBonesGeometry(kind) {
			t.Fatalf("v=%d bones: texture %v geometry %s", v, got, bp.Geometry)
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
	if p := stateProbeBase(assets.StateProbePerm); p.Geometry != stateProbeShapeGeometry(1, 0) || p.Scale != (mgl64.Vec3{}) {
		t.Errorf("perm base %+v", p)
	}
}

// TestStateProbePermScale: the server emits minecraft:transformation only for
// a non-zero scale, and permutations inherit the base. So in mode perm the base
// has no scale and only the permutation of the value 0 shrinks.
func TestStateProbePermScale(t *testing.T) {
	if s := stateProbeBase(assets.StateProbePerm).Scale; s != (mgl64.Vec3{}) {
		t.Errorf("base scale %v", s)
	}
	for i, p := range stateProbePermutations(assets.StateProbePerm, 0) {
		hidden := p.Properties.Scale != (mgl64.Vec3{}) && p.Properties.Scale != (mgl64.Vec3{1, 1, 1})
		if want := i == 0; hidden != want {
			t.Errorf("permutation %d (%s): scale %v", i, p.Condition, p.Properties.Scale)
		}
	}
	if c := stateProbePermutations(assets.StateProbePerm, 0)[0].Condition; c != "q.block_state('skygear:shape') == 0" {
		t.Errorf("first condition %s", c)
	}
}

// probeNetworkHash is Dragonfly's network block hash of a state
// (world.networkBlockHash): FNV-1a 32 of its NBT, the states sorted by name.
func probeNetworkHash(name string, props map[string]any) uint32 {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var d []byte
	str := func(s string) { d = binary.LittleEndian.AppendUint16(d, uint16(len(s))); d = append(d, s...) }
	d = append(d, 10, 0, 0, 8)
	str("name")
	str(name)
	d = append(d, 10)
	str("states")
	for _, k := range keys {
		switch v := props[k].(type) {
		case string:
			d = append(d, 8)
			str(k)
			str(v)
		case uint8, int8, bool:
			b := byte(0)
			switch v := v.(type) {
			case uint8:
				b = v
			case int8:
				b = byte(v)
			case bool:
				if v {
					b = 1
				}
			}
			d = append(d, 1)
			str(k)
			d = append(d, b)
		case int16:
			d = append(d, 2)
			str(k)
			d = binary.LittleEndian.AppendUint16(d, uint16(v))
		case uint16:
			d = append(d, 2)
			str(k)
			d = binary.LittleEndian.AppendUint16(d, v)
		case int32:
			d = append(d, 3)
			str(k)
			d = binary.LittleEndian.AppendUint32(d, uint32(v))
		case uint32:
			d = append(d, 3)
			str(k)
			d = binary.LittleEndian.AppendUint32(d, v)
		default:
			panic(fmt.Sprintf("state %s of %s: %T", k, name, v))
		}
	}
	d = append(d, 0, 0)
	h := uint32(2166136261)
	for _, b := range d {
		h ^= uint32(b)
		h *= 16777619
	}
	return h
}

// TestStateProbeNetworkHashes: Dragonfly panics at start when two states share
// a network block hash. In every mode the probe's states (all of all4's) share
// none with each other or with the other blocks registered; if a new block
// meets one, change stateProbeSalt (or, for perm and bones, the names).
func TestStateProbeNetworkHashes(t *testing.T) {
	others := map[uint32]string{}
	for _, b := range world.Blocks() {
		if _, ok := b.(StateProbe); ok {
			continue
		}
		name, props := b.EncodeBlock()
		others[probeNetworkHash(name, props)] = name
	}
	if len(others) < 10000 {
		t.Fatalf("%d other states", len(others))
	}
	for _, mode := range probeModes {
		seen := map[uint32]bool{}
		for l := range stateProbeLayers {
			name := stateProbeName(mode, l)
			for v := range stateProbeLayerValues(mode, l) {
				h := probeNetworkHash(name, map[string]any{stateProbeState: int32(v)})
				if other, ok := others[h]; ok || seen[h] {
					t.Fatalf("%s: %s %d meets the network hash of %q (stateProbeSalt %d)", mode, name, v, other, stateProbeSalt)
				}
				seen[h] = true
			}
		}
	}
}
