package skygear

import (
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/assets"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// The state count probe is for a manual client test: how much longer does a
// client take to join and load with about 1,150 more block states, drawn in
// two ways? It is off unless the environment variable SKYGEAR_STATE_PROBE
// (assets.StateProbeEnv) says otherwise, and off registers nothing and puts
// no probe file in the pack:
//
//   - unset or off (A): the game as it is.
//   - perm (B): one permutation for each state, each with its own block
//     geometry (state_probe_shape_<kind>_<faces>, 3 × 64 of them, the lane of
//     the kind built into it) and texture.
//   - bones (C): a geometry for each kind (state_probe_bones_<kind>, the lane
//     built into it) with a bone for the core and each arm, and one
//     permutation for each kind and tier (9 a block) that sets the geometry
//     and texture and switches the arm bones by bone_visibility, a
//     Molang expression of the state (the Dragonfly patch
//     customblock.Properties.BoneVisibility).
//
// Two blocks, skygear:state_probe_0 and _1 (stand-ins for the two drawing
// layers; skygear:state_probe4_2_0 and _1 in the 4-kind modes, see
// stateProbeSalt), each have one int state skygear:shape (one state a block, see
// TestCustomBlockStatesMatchTheDefinition), 0 to 576. 0 is empty; v is
// 1 + kind*192 + (tier-1)*64 + faces: kind 0 to 2 (energy, item, fluid), tier
// 1 to 3 and faces the arms as a 6-bit set in the order of conduit.Faces
// (down, up, north, south, west, east). The look is a conduit of the kind and
// tier, in the lane of its kind (energy -3, item 0, fluid +3 pixels on every
// axis, as the conduit visuals; tools/stateprobegeo builds it into the
// geometry so the arms reach the edge of the block as the entity's do), with
// the textures of the conduit items.

// The 4-kind modes (single4, same4, all4) model a fourth conduit kind, the
// network conduit (kinds 0 to 3: energy, item, fluid, network), all with the
// bones technique: geometries with the bones of one kind or of two
// (state_probe_four.geo.json, geometry.skygear.state_probe_one_<kind> and
// _two_<a>_<b>, bones <kind>_core, <kind>_down ...; the network lane is
// (+6, -6, +6) pixels, see tools/stateprobegeo), one int state a block and a
// permutation for each kind and tier and for each pair of kinds and tiers.
// Each face of the geometry names the material instance of its kind, which
// the permutation maps to the texture of the kind's tier, so two kinds with
// their own textures share one block. The network conduit has no texture yet:
// its looks use skygear_conduit_node (grey) as a placeholder, for every tier.
//
// Values of a block in these modes, in contiguous ranges:
//
//   - 0: empty.
//   - singles, 1 to 768: 1 + kind*192 + (tier-1)*64 + faces (the layout of
//     perm and bones with kind 3 added).
//   - same-faces pairs (same4, layer 1), 769 to 4224:
//     769 + (pair*9 + (ta-1)*3 + (tb-1))*64 + faces, both kinds with faces.
//   - any pairs (all4, both layers), 769 to 221952:
//     769 + ((pair*9 + (ta-1)*3 + (tb-1))*64 + fa)*64 + fb.
//
// pair is the index of the kinds a < b in 01 02 03 12 13 23 (stateProbeKindPairs);
// stateProbeDecode and stateProbeEncode turn values into looks and back.
//
//	mode     layer 0 values         layer 1 values
//	single4  769                    769
//	same4    769                    769 + 3,456 = 4,225
//	all4     769 + 221,184 = 221,953 (443,906 states for both blocks)

const (
	// stateProbeValues is the number of values of the state of a probe block
	// in modes perm and bones.
	stateProbeValues = 577
	// stateProbeLayers is the number of probe blocks.
	stateProbeLayers = 2

	stateProbeState = "skygear:shape"
	stateProbeKinds = 3
	stateProbeTiers = 3
	// a (kind, tier) is 64 values: the sets of 6 faces
	stateProbeFaces = 64

	// stateProbeKinds4 are the kinds of the 4-kind modes.
	stateProbeKinds4 = 4
	// stateProbePairStart is the first pair value of the 4-kind modes, after
	// 0 and the 4 × 3 × 64 singles.
	stateProbePairStart = 1 + stateProbeKinds4*stateProbeTiers*stateProbeFaces
	// stateProbeTierPairs are the tiers of two kinds.
	stateProbeTierPairs = stateProbeTiers * stateProbeTiers
	// stateProbeSamePairs and stateProbeAnyPairs are the pair values of a
	// block with same-faces pairs and with any pairs.
	stateProbeSamePairs = len(stateProbeKindPairs) * stateProbeTierPairs * stateProbeFaces
	stateProbeAnyPairs  = stateProbeSamePairs * stateProbeFaces
)

// stateProbeKindPairs are the pairs of kinds of the 4-kind modes, in the order
// of their values (and of the pair geometries).
var stateProbeKindPairs = [6][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}

// probePairs is what pairs of kinds a probe block draws.
type probePairs int

const (
	probeNoPairs probePairs = iota
	// probeSamePairs: two kinds with the same faces (a value per faces)
	probeSamePairs
	// probeAnyPairs: two kinds with any faces each
	probeAnyPairs
)

// stateProbeFourKinds reports whether the mode is one of single4, same4, all4.
func stateProbeFourKinds(mode string) bool {
	switch mode {
	case assets.StateProbeSingle4, assets.StateProbeSame4, assets.StateProbeAll4:
		return true
	}
	return false
}

// stateProbeLayerPairs are the pairs the block of a layer draws in the mode.
func stateProbeLayerPairs(mode string, layer int) probePairs {
	switch mode {
	case assets.StateProbeSame4:
		if layer == 1 {
			return probeSamePairs
		}
	case assets.StateProbeAll4:
		return probeAnyPairs
	}
	return probeNoPairs
}

// stateProbePairSpan is the number of values of a (pair, tiers): 64 or 4,096.
func stateProbePairSpan(p probePairs) int {
	if p == probeAnyPairs {
		return stateProbeFaces * stateProbeFaces
	}
	return stateProbeFaces
}

// stateProbeLayerValues is the number of values of the state of the block of
// a layer in the mode (0 when off).
func stateProbeLayerValues(mode string, layer int) int {
	switch {
	case mode == assets.StateProbePerm || mode == assets.StateProbeBones:
		return stateProbeValues
	case stateProbeFourKinds(mode):
		switch stateProbeLayerPairs(mode, layer) {
		case probeSamePairs:
			return stateProbePairStart + stateProbeSamePairs
		case probeAnyPairs:
			return stateProbePairStart + stateProbeAnyPairs
		}
		return stateProbePairStart
	}
	return 0
}

// stateProbeModeKinds are the kinds of the mode: 3 or 4.
func stateProbeModeKinds(mode string) int {
	if stateProbeFourKinds(mode) {
		return stateProbeKinds4
	}
	return stateProbeKinds
}

// probeLook is what a value draws: n (0 to 2) conduits of different kinds,
// in the order of their kinds.
type probeLook struct {
	n int
	k [2]fillKind
}

// stateProbeDecode is the look of the value v of the block of a layer in the
// mode; false when v is not a value of it.
func stateProbeDecode(mode string, layer, v int) (probeLook, bool) {
	if v < 0 || v >= stateProbeLayerValues(mode, layer) {
		return probeLook{}, false
	}
	if v == 0 {
		return probeLook{}, true
	}
	if v < stateProbePairStart {
		kind, tier, faces := stateProbeParts(v)
		return probeLook{n: 1, k: [2]fillKind{{kind, tier, faces}}}, true
	}
	n := v - stateProbePairStart
	var fa, fb int
	switch stateProbeLayerPairs(mode, layer) {
	case probeSamePairs:
		fa, fb, n = n%stateProbeFaces, n%stateProbeFaces, n/stateProbeFaces
	case probeAnyPairs:
		fa, fb, n = n/stateProbeFaces%stateProbeFaces, n%stateProbeFaces, n/(stateProbeFaces*stateProbeFaces)
	}
	pair, tiers := stateProbeKindPairs[n/stateProbeTierPairs], n%stateProbeTierPairs
	return probeLook{n: 2, k: [2]fillKind{
		{pair[0], tiers/stateProbeTiers + 1, fa},
		{pair[1], tiers%stateProbeTiers + 1, fb},
	}}, true
}

// stateProbeEncode is the value of a look on the block of a layer in the mode
// (the conduits in any order); false when that block cannot draw it.
func stateProbeEncode(mode string, layer int, l probeLook) (int, bool) {
	kinds := stateProbeModeKinds(mode)
	for _, k := range l.k[:l.n] {
		if k.kind < 0 || k.kind >= kinds || k.tier < 1 || k.tier > stateProbeTiers || k.faces < 0 || k.faces >= stateProbeFaces {
			return 0, false
		}
	}
	var v int
	switch l.n {
	case 0:
		v = 0
	case 1:
		v = l.k[0].probeValue()
	case 2:
		a, b := l.k[0], l.k[1]
		if a.kind > b.kind {
			a, b = b, a
		}
		pair := -1
		for i, p := range stateProbeKindPairs {
			if p == [2]int{a.kind, b.kind} {
				pair = i
			}
		}
		if pair < 0 {
			return 0, false
		}
		n := pair*stateProbeTierPairs + (a.tier-1)*stateProbeTiers + (b.tier - 1)
		switch stateProbeLayerPairs(mode, layer) {
		case probeSamePairs:
			if a.faces != b.faces {
				return 0, false
			}
			v = stateProbePairStart + n*stateProbeFaces + a.faces
		case probeAnyPairs:
			v = stateProbePairStart + (n*stateProbeFaces+a.faces)*stateProbeFaces + b.faces
		default:
			return 0, false
		}
	default:
		return 0, false
	}
	if v >= stateProbeLayerValues(mode, layer) {
		return 0, false
	}
	return v, true
}

// stateProbeMode is the mode read once at init (assets.StateProbeOff,
// StateProbePerm, StateProbeBones or a 4-kind mode).
var stateProbeMode string

var stateProbeHash uint64

func init() {
	stateProbeMode = assets.StateProbeMode()
	if stateProbeMode == assets.StateProbeOff {
		return
	}
	start := time.Now()
	for l := range stateProbeNames {
		stateProbeNames[l] = stateProbeName(stateProbeMode, l)
	}
	stateProbeHash = block.NextHash()
	for l := 0; l < stateProbeLayers; l++ {
		for v := 0; v < stateProbeLayerValues(stateProbeMode, l); v++ {
			world.RegisterBlock(StateProbe{Layer: l, V: v})
		}
	}
	took := time.Since(start)
	states, perms := stateProbeCounts(stateProbeMode)
	// all4 registers 443,906 states: the time and memory show what that
	// costs (the palette is built later, when the server starts).
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	slog.Info("state probe on", "mode", stateProbeMode,
		"states_layer0", stateProbeLayerValues(stateProbeMode, 0), "states_layer1", stateProbeLayerValues(stateProbeMode, 1),
		"states", states, "permutations", perms, "geometries", stateProbeGeometries(stateProbeMode),
		"register_ms", took.Milliseconds(), "heap_mb", mem.HeapAlloc>>20, "sys_mb", mem.Sys>>20)
}

// stateProbeCounts are the block states and the permutations the mode adds.
func stateProbeCounts(mode string) (states, permutations int) {
	if mode == assets.StateProbeOff {
		return 0, 0
	}
	for l := 0; l < stateProbeLayers; l++ {
		states += stateProbeLayerValues(mode, l)
		permutations += len(stateProbePermutations(mode, l))
	}
	return states, permutations
}

// stateProbeGeometries is the number of geometries the probe blocks of the
// mode use.
func stateProbeGeometries(mode string) int {
	if mode == assets.StateProbeOff {
		return 0
	}
	used := map[string]bool{stateProbeBase(mode).Geometry: true}
	for l := 0; l < stateProbeLayers; l++ {
		for _, p := range stateProbePermutations(mode, l) {
			used[p.Geometry] = true
		}
	}
	return len(used)
}

// stateProbeBlock is the probe block of a layer (0 or 1) with the value v of
// its state (0 to stateProbeLayerValues-1); false when the probe is off or the
// layer or value is out of range.
func stateProbeBlock(layer int, v int) (world.Block, bool) {
	if stateProbeMode == assets.StateProbeOff || layer < 0 || layer >= stateProbeLayers || v < 0 || v >= stateProbeLayerValues(stateProbeMode, layer) {
		return nil, false
	}
	return StateProbe{Layer: layer, V: v}, true
}

// StateProbe is a block of the state count probe: Layer 0 or 1, V the value of
// skygear:shape. It has no item and nothing in the game places it but the
// debug command that fills with it.
type StateProbe struct{ Layer, V int }

// stateProbeNames are the names of the probe blocks of the mode, set at init.
var stateProbeNames [stateProbeLayers]string

// stateProbeSalt is in the names of the 4-kind modes' blocks. Dragonfly
// panics at start when two block states share a network block hash (FNV-1a
// 32 of the state's NBT), and all4's 443,906 states meet about 2.7 of the
// other 26,000 states' hashes whatever the name; with this salt they meet
// none. TestStateProbeNetworkHashes checks every mode against the blocks
// registered: when a new block collides, take the next salt it accepts.
const stateProbeSalt = 2

// stateProbeName is the name of the probe block of a layer in the mode:
// skygear:state_probe_<layer> (perm, bones) or
// skygear:state_probe4_<salt>_<layer> (4-kind modes).
func stateProbeName(mode string, layer int) string {
	if stateProbeFourKinds(mode) {
		return fmt.Sprintf("skygear:state_probe4_%d_%d", stateProbeSalt, layer)
	}
	return fmt.Sprintf("skygear:state_probe_%d", layer)
}

func (b StateProbe) EncodeBlock() (string, map[string]any) {
	return stateProbeNames[b.Layer], map[string]any{stateProbeState: int32(b.V)}
}

func (b StateProbe) Hash() (uint64, uint64) {
	return stateProbeHash, uint64(b.V)<<1 | uint64(b.Layer)
}

func (StateProbe) Model() world.BlockModel { return model.Empty{} }

// stateProbeParts splits a single value (1 to 576, or to 768 with 4 kinds)
// into kind, tier (1 to 3) and the set of arms.
func stateProbeParts(v int) (kind, tier, faces int) {
	n := v - 1
	return n / (stateProbeTiers * stateProbeFaces), n%(stateProbeTiers*stateProbeFaces)/stateProbeFaces + 1, n % stateProbeFaces
}

// stateProbeCore is the selection box of every look: the core.
var stateProbeCore = cube.Box(6.5/16, 6.5/16, 6.5/16, 9.5/16, 9.5/16, 9.5/16)

// stateProbeLook is the look of a kind and tier: its texture and core.
func stateProbeLook(geometry string, kind, tier int) customblock.Properties {
	return customblock.Properties{
		Geometry:     geometry,
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial(stateProbeTexture(kind, tier))},
		SelectionBox: stateProbeCore,
	}
}

var conduitKindNames = [stateProbeKinds]string{"energy", "item", "fluid"}

// stateProbeKindNames4 are the kinds of the 4-kind modes, also the names of
// their bones' prefix and material instances.
var stateProbeKindNames4 = [stateProbeKinds4]string{"energy", "item", "fluid", "network"}

// stateProbeTexture is the texture of a kind and tier. The network conduit has
// none yet: the grey conduit node stands in for every tier.
func stateProbeTexture(kind, tier int) string {
	if kind == 3 {
		return "skygear_conduit_node"
	}
	return fmt.Sprintf("skygear_conduit_%s_%d", conduitKindNames[kind], tier)
}

// stateProbeArms are the arm bones of the geometry of mode bones, bit i of the
// faces being the arm i (the order of conduit.Faces).
var stateProbeArms = [6]string{"down", "up", "north", "south", "west", "east"}

// stateProbeShapeGeometry is the geometry of mode perm for a kind and a set of
// arms.
func stateProbeShapeGeometry(kind, faces int) string {
	return fmt.Sprintf("geometry.skygear.state_probe_shape_%s_%d", conduitKindNames[kind], faces)
}

// stateProbeBonesGeometry is the geometry of mode bones for a kind.
func stateProbeBonesGeometry(kind int) string {
	return "geometry.skygear.state_probe_bones_" + conduitKindNames[kind]
}

// stateProbeOneGeometry and stateProbeTwoGeometry are the geometries of the
// 4-kind modes for one kind and for two (a < b).
func stateProbeOneGeometry(kind int) string {
	return "geometry.skygear.state_probe_one_" + stateProbeKindNames4[kind]
}

func stateProbeTwoGeometry(a, b int) string {
	return "geometry.skygear.state_probe_two_" + stateProbeKindNames4[a] + "_" + stateProbeKindNames4[b]
}

// stateProbeBase is the look of the value 0, empty. Mode perm has no geometry
// without a core to hide, so it takes the plain core and shrinks it out of
// sight; the bones modes hide every bone.
func stateProbeBase(mode string) customblock.Properties {
	switch {
	case mode == assets.StateProbeBones:
		p := stateProbeLook(stateProbeBonesGeometry(1), 1, 1)
		p.BoneVisibility = map[string]string{"core": "false"}
		for _, arm := range stateProbeArms {
			p.BoneVisibility[arm] = "false"
		}
		return p
	case stateProbeFourKinds(mode):
		p := stateProbeFourLook(stateProbeOneGeometry(1), fillKind{kind: 1, tier: 1})
		p.BoneVisibility = map[string]string{}
		for _, bone := range stateProbeKindBones(1) {
			p.BoneVisibility[bone] = "false"
		}
		return p
	}
	// No scale here: permutations inherit the base, so a scale on it would
	// shrink every one of them. The value 0 has its own permutation.
	return stateProbeLook(stateProbeShapeGeometry(1, 0), 1, 1)
}

// stateProbeKindBones are the bones of a kind in the 4-kind geometries: the
// core, then the arms in the order of the faces.
func stateProbeKindBones(kind int) []string {
	out := []string{stateProbeKindNames4[kind] + "_core"}
	for _, arm := range stateProbeArms {
		out = append(out, stateProbeKindNames4[kind]+"_"+arm)
	}
	return out
}

// stateProbeFourLook is the look of a geometry of the 4-kind modes drawing
// the kinds given, each with the texture of its tier on its material
// instance ("*" takes the first, for faces without one).
func stateProbeFourLook(geometry string, kinds ...fillKind) customblock.Properties {
	p := customblock.Properties{
		Geometry:     geometry,
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial(stateProbeTexture(kinds[0].kind, kinds[0].tier))},
		SelectionBox: stateProbeCore,
	}
	for _, k := range kinds {
		p.Textures[stateProbeKindNames4[k.kind]] = blockmodel.AlphaMaterial(stateProbeTexture(k.kind, k.tier))
	}
	return p
}

// stateProbeHidden is the scale of the permutation of the value 0.
var stateProbeHidden = mgl64.Vec3{0.001, 0.001, 0.001}

// stateProbeShapeExpr is the Molang expression of the state value.
const stateProbeShapeExpr = "q.block_state('" + stateProbeState + "')"

// stateProbeBitExpr is true when bit i of floor((v - lo) / unit) is set.
func stateProbeBitExpr(lo, unit, i int) string {
	return fmt.Sprintf("math.mod(math.floor((%s - %d) / %d), 2) == 1", stateProbeShapeExpr, lo, unit<<i)
}

// stateProbeRangeCond is the condition of the values lo to lo+n-1.
func stateProbeRangeCond(lo, n int) string {
	return fmt.Sprintf("%s >= %d && %s <= %d", stateProbeShapeExpr, lo, stateProbeShapeExpr, lo+n-1)
}

// stateProbePermutations are the permutations of the probe block of a layer
// in the mode (the same for both layers but in same4).
func stateProbePermutations(mode string, layer int) []customblock.Permutation {
	var out []customblock.Permutation
	switch {
	case mode == assets.StateProbePerm:
		empty := stateProbeBase(mode)
		empty.Scale = stateProbeHidden
		out = append(out, customblock.Permutation{
			Condition:  stateProbeShapeExpr + " == 0",
			Properties: empty,
		})
		for v := 1; v < stateProbeValues; v++ {
			kind, tier, faces := stateProbeParts(v)
			out = append(out, customblock.Permutation{
				Condition:  fmt.Sprintf("%s == %d", stateProbeShapeExpr, v),
				Properties: stateProbeLook(stateProbeShapeGeometry(kind, faces), kind, tier),
			})
		}
	case mode == assets.StateProbeBones:
		for kind := 0; kind < stateProbeKinds; kind++ {
			for tier := 1; tier <= stateProbeTiers; tier++ {
				lo := 1 + kind*stateProbeTiers*stateProbeFaces + (tier-1)*stateProbeFaces
				p := stateProbeLook(stateProbeBonesGeometry(kind), kind, tier)
				p.BoneVisibility = map[string]string{"core": "true"}
				for i, arm := range stateProbeArms {
					// bit i of (v - lo): v is in lo..lo+63, so the faces are v - lo
					p.BoneVisibility[arm] = stateProbeBitExpr(lo, 1, i)
				}
				out = append(out, customblock.Permutation{
					Condition:  stateProbeRangeCond(lo, stateProbeFaces),
					Properties: p,
				})
			}
		}
	case stateProbeFourKinds(mode):
		// singles: the faces are v - lo
		for kind := 0; kind < stateProbeKinds4; kind++ {
			for tier := 1; tier <= stateProbeTiers; tier++ {
				lo := 1 + kind*stateProbeTiers*stateProbeFaces + (tier-1)*stateProbeFaces
				p := stateProbeFourLook(stateProbeOneGeometry(kind), fillKind{kind: kind, tier: tier})
				p.BoneVisibility = stateProbeKindVisibility(kind, lo, 1)
				out = append(out, customblock.Permutation{Condition: stateProbeRangeCond(lo, stateProbeFaces), Properties: p})
			}
		}
		// pairs: n = v - lo; same faces: both kinds n; any: a floor(n/64), b mod(n, 64)
		pairs := stateProbeLayerPairs(mode, layer)
		if pairs == probeNoPairs {
			break
		}
		span := stateProbePairSpan(pairs)
		unitA := 1
		if pairs == probeAnyPairs {
			unitA = stateProbeFaces
		}
		for pi, pair := range stateProbeKindPairs {
			for ta := 1; ta <= stateProbeTiers; ta++ {
				for tb := 1; tb <= stateProbeTiers; tb++ {
					lo := stateProbePairStart + (pi*stateProbeTierPairs+(ta-1)*stateProbeTiers+(tb-1))*span
					p := stateProbeFourLook(stateProbeTwoGeometry(pair[0], pair[1]), fillKind{kind: pair[0], tier: ta}, fillKind{kind: pair[1], tier: tb})
					p.BoneVisibility = stateProbeKindVisibility(pair[0], lo, unitA)
					maps.Copy(p.BoneVisibility, stateProbeKindVisibility(pair[1], lo, 1))
					out = append(out, customblock.Permutation{Condition: stateProbeRangeCond(lo, span), Properties: p})
				}
			}
		}
	}
	return out
}

// stateProbeKindVisibility is the bone_visibility of a kind whose faces are
// floor((v - lo) / unit) mod 64: the core and the arm of each bit.
func stateProbeKindVisibility(kind, lo, unit int) map[string]string {
	bones := stateProbeKindBones(kind)
	out := map[string]string{bones[0]: "true"}
	for i := range stateProbeArms {
		out[bones[1+i]] = stateProbeBitExpr(lo, unit, i)
	}
	return out
}

// Properties is the look of the value 0; Permutations has the others.
func (StateProbe) Properties() customblock.Properties { return stateProbeBase(stateProbeMode) }

// stateProbeStateValues are the values of skygear:shape of each layer, made
// once (all4 has 221,953 of them).
var stateProbeStateValues = [stateProbeLayers]func() []any{stateProbeValuesOf(0), stateProbeValuesOf(1)}

func stateProbeValuesOf(layer int) func() []any {
	return sync.OnceValue(func() []any {
		values := make([]any, stateProbeLayerValues(stateProbeMode, layer))
		for i := range values {
			values[i] = int32(i)
		}
		return values
	})
}

// States implements block.Permutable.
func (b StateProbe) States() map[string][]any {
	return map[string][]any{stateProbeState: stateProbeStateValues[b.Layer]()}
}

// Permutations implements block.Permutable.
func (b StateProbe) Permutations() []customblock.Permutation {
	return stateProbePermutations(stateProbeMode, b.Layer)
}

func (StateProbe) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return nil },
	}
}
