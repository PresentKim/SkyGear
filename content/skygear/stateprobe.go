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

// The 4-kind modes (single4, same4, all4, all4split) model a fourth conduit kind, the
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
//
// all4split (the 4-kind mode with the values of all4) splits each layer into
// 7 block names of at most 65,536 states, because the client may know only the
// first 65,536 states of a block name (a state past it draws nothing and has no
// collision): a singles block (value 0 and the 768 singles, 769 states, 12
// permutations) and a block for each pair of kinds (9 tier combinations × 64 ×
// 64 faces = 36,864 states, 9 permutations). The value v of the look stays the
// all4 value (StateProbe.V); the state of skygear:shape on the block is
// v - the first value of the block (stateProbeLocate), so the Nth state of a
// name (the order of its States, the order the client gets) is that state.

// pair2 (see stateProbePair2) has two fixed blocks of two kinds each: block 0
// energy + item (kinds 0 and 1), block 1 fluid + network (kinds 2 and 3), both
// with the bones technique and the geometries two_energy_item and
// two_fluid_network. Here StateProbe.Layer is the block, not the storage layer:
// the cell puts the non-empty blocks on storage layer 0, then 1 (block 0 first;
// block 1 alone goes on layer 0, as a layer 1 block draws over a visible layer
// 0 only). A block's value is a*193 + b, 0 to 37,248 (37,249 states, under the
// 65,536 of a name), a for its first kind and b for the second, each 0 (absent)
// or 1 + (tier-1)*64 + faces (1 to 192). Value 0 (both absent) is never placed.
// The 15 permutations of a block are (tier of a or absent) × (tier of b or
// absent) less (absent, absent); a = floor(v/193) and b = mod(v, 193) are
// decoded in Molang.

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

	// stateProbeNameCap is the number of states of a block name the client is
	// thought to know at most (the hypothesis behind all4split).
	stateProbeNameCap = 65536
	// stateProbePairBlock is the number of states of a pair block of all4split.
	stateProbePairBlock = stateProbeTierPairs * stateProbeFaces * stateProbeFaces
	// stateProbeMaxBlocks is the number of block names of a layer in all4split:
	// the singles and a block for each pair of kinds.
	stateProbeMaxBlocks = 1 + len(stateProbeKindPairs)

	// stateProbePair2Kind is the number of values of one kind of a pair2 block:
	// absent and 3 × 64.
	stateProbePair2Kind = 1 + stateProbeTiers*stateProbeFaces
	// stateProbePair2Values is the number of values (states) of a pair2 block.
	stateProbePair2Values = stateProbePair2Kind * stateProbePair2Kind
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

// stateProbeFourKinds reports whether the mode is one of single4, same4, all4,
// all4split, pair2.
func stateProbeFourKinds(mode string) bool {
	switch mode {
	case assets.StateProbeSingle4, assets.StateProbeSame4, assets.StateProbeAll4, assets.StateProbeAll4Split, assets.StateProbePair2:
		return true
	}
	return false
}

// stateProbePair2 reports whether the mode is pair2.
func stateProbePair2(mode string) bool { return mode == assets.StateProbePair2 }

// stateProbeSplit reports whether the mode splits a layer into several block
// names.
func stateProbeSplit(mode string) bool { return mode == assets.StateProbeAll4Split }

// stateProbeBlocks is the number of block names of a layer in the mode.
func stateProbeBlocks(mode string, layer int) int {
	switch {
	case stateProbeLayerValues(mode, layer) == 0:
		return 0
	case stateProbeSplit(mode):
		return stateProbeMaxBlocks
	}
	return 1
}

// stateProbeBlockSpan is the first value and the number of values (states) of
// block blk of a layer: the block's states are the values lo to lo+n-1.
func stateProbeBlockSpan(mode string, layer, blk int) (lo, n int) {
	if !stateProbeSplit(mode) {
		return 0, stateProbeLayerValues(mode, layer)
	}
	if blk == 0 {
		return 0, stateProbePairStart
	}
	return stateProbePairStart + (blk-1)*stateProbePairBlock, stateProbePairBlock
}

// stateProbeLocate is the block of a layer that has the value v and the value
// of its skygear:shape (the Nth state of the block's name).
func stateProbeLocate(mode string, layer, v int) (blk, state int) {
	if !stateProbeSplit(mode) || v < stateProbePairStart {
		return 0, v
	}
	d := v - stateProbePairStart
	return 1 + d/stateProbePairBlock, d % stateProbePairBlock
}

// stateProbeLayerPairs are the pairs the block of a layer draws in the mode.
func stateProbeLayerPairs(mode string, layer int) probePairs {
	switch mode {
	case assets.StateProbeSame4:
		if layer == 1 {
			return probeSamePairs
		}
	case assets.StateProbeAll4, assets.StateProbeAll4Split:
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
	case stateProbePair2(mode):
		return stateProbePair2Values
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
	if stateProbePair2(mode) {
		return stateProbePair2Decode(layer, v), true
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
	if stateProbePair2(mode) {
		return stateProbePair2Encode(layer, l)
	}
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

// stateProbePair2Part is the value of one kind in a pair2 block: 0 when
// absent, else 1 + (tier-1)*64 + faces.
func stateProbePair2Part(k fillKind) int { return 1 + (k.tier-1)*stateProbeFaces + k.faces }

// stateProbePair2Decode is the look of the value v (in range) of pair2 block
// layer: its first kind (a) then its second (b), those present.
func stateProbePair2Decode(layer, v int) probeLook {
	var l probeLook
	for i, x := range [2]int{v / stateProbePair2Kind, v % stateProbePair2Kind} {
		if x == 0 {
			continue
		}
		x--
		l.k[l.n] = fillKind{kind: 2*layer + i, tier: x/stateProbeFaces + 1, faces: x % stateProbeFaces}
		l.n++
	}
	return l
}

// stateProbePair2Encode is the value of a look (0 to 2 conduits, in any order)
// on pair2 block layer; false when a conduit is not of the block's kinds, the
// block has one kind twice or a tier or faces is out of range.
func stateProbePair2Encode(layer int, l probeLook) (int, bool) {
	if layer < 0 || layer >= stateProbeLayers || l.n < 0 || l.n > 2 {
		return 0, false
	}
	var part [2]int
	for _, k := range l.k[:l.n] {
		i := k.kind - 2*layer
		if i < 0 || i > 1 || part[i] != 0 || k.tier < 1 || k.tier > stateProbeTiers || k.faces < 0 || k.faces >= stateProbeFaces {
			return 0, false
		}
		part[i] = stateProbePair2Part(k)
	}
	return part[0]*stateProbePair2Kind + part[1], true
}

// stateProbeLookString names a look for the console: "energy tier 2 down+north".
func stateProbeLookString(l probeLook) string {
	if l.n == 0 {
		return "nothing"
	}
	var out string
	for i, k := range l.k[:l.n] {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s tier %d", stateProbeKindNames4[k.kind], k.tier)
		sep := " "
		for b, arm := range stateProbeArms {
			if k.faces&(1<<b) != 0 {
				out += sep + arm
				sep = "+"
			}
		}
		if k.faces == 0 {
			out += " core only"
		}
	}
	return out
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
		for blk := range stateProbeNames[l] {
			stateProbeNames[l][blk] = stateProbeBlockName(stateProbeMode, l, blk)
		}
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
	names, maxStates := 0, 0
	for _, c := range stateProbeNameCounts(stateProbeMode) {
		names++
		maxStates = max(maxStates, c.states)
	}
	slog.Info("state probe on", "mode", stateProbeMode, "block_names", names, "max_states_per_name", maxStates,
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

// stateProbeNameCounts are the block names of the mode with their states and
// permutations, in the order of layer and block (nothing when off): what the
// registry holds for them once registered.
func stateProbeNameCounts(mode string) []blockNameCount {
	var out []blockNameCount
	for l := 0; l < stateProbeLayers; l++ {
		for blk := 0; blk < stateProbeBlocks(mode, l); blk++ {
			_, n := stateProbeBlockSpan(mode, l, blk)
			out = append(out, blockNameCount{
				name: stateProbeBlockName(mode, l, blk), states: n,
				permutations: len(stateProbeBlockPermutations(mode, l, blk)),
			})
		}
	}
	return out
}

// stateProbeGeometries is the number of geometries the probe blocks of the
// mode use.
func stateProbeGeometries(mode string) int {
	if mode == assets.StateProbeOff {
		return 0
	}
	used := map[string]bool{}
	for l := 0; l < stateProbeLayers; l++ {
		used[stateProbeBaseOf(mode, l).Geometry] = true
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
// the look (the value of skygear:shape, but in all4split the state of a block
// is the value less the first value of the block, see stateProbeLocate). It has
// no item and nothing in the game places it but the debug command that fills
// with it.
type StateProbe struct{ Layer, V int }

// stateProbeNames are the names of the probe blocks of the mode (the blocks of
// a layer), set at init.
var stateProbeNames [stateProbeLayers][stateProbeMaxBlocks]string

// stateProbeSalt is in the names of the 4-kind modes' blocks. Dragonfly
// panics at start when two block states share a network block hash (FNV-1a
// 32 of the state's NBT), and all4's 443,906 states meet about 2.7 of the
// other 26,000 states' hashes whatever the name; with this salt they meet
// none. TestStateProbeNetworkHashes checks every mode against the blocks
// registered: when a new block collides, take the next salt it accepts.
const stateProbeSalt = 2

// stateProbeSplitSalt is the same for the names of all4split (14 names; salt 0
// met the hash of skygear:auto_sieve, 1 meets none).
const stateProbeSplitSalt = 1

// stateProbePair2Salt is the same for the two names of pair2.
const stateProbePair2Salt = 0

// stateProbeName is the name of the first probe block of a layer in the mode.
func stateProbeName(mode string, layer int) string { return stateProbeBlockName(mode, layer, 0) }

// stateProbeBlockName is the name of block blk of the probe of a layer in the
// mode: skygear:state_probe_<layer> (perm, bones),
// skygear:state_probe4_<salt>_<layer> (other 4-kind modes),
// skygear:state_probe4s_<salt>_<layer>_<blk> (all4split) or
// skygear:state_probe2_<salt>_<block> (pair2).
func stateProbeBlockName(mode string, layer, blk int) string {
	switch {
	case stateProbePair2(mode):
		return fmt.Sprintf("skygear:state_probe2_%d_%d", stateProbePair2Salt, layer)
	case stateProbeSplit(mode):
		return fmt.Sprintf("skygear:state_probe4s_%d_%d_%d", stateProbeSplitSalt, layer, blk)
	case stateProbeFourKinds(mode):
		return fmt.Sprintf("skygear:state_probe4_%d_%d", stateProbeSalt, layer)
	}
	return fmt.Sprintf("skygear:state_probe_%d", layer)
}

// location is the block of the probe block in its layer and the value of its
// skygear:shape.
func (b StateProbe) location() (blk, state int) {
	return stateProbeLocate(stateProbeMode, b.Layer, b.V)
}

// stateProbeInfo is where a probe block is among the states of its name.
type stateProbeInfo struct {
	name  string
	state int // the value of skygear:shape
	index int // the Nth state of the name, 0-based, in the order of States
	count int // the number of states of the name
}

// stateProbeInfoOf is the name, state and index of a probe block among the
// states of its name (false for any other block).
func stateProbeInfoOf(b world.Block) (stateProbeInfo, bool) {
	sp, ok := b.(StateProbe)
	if !ok || sp.Layer < 0 || sp.Layer >= stateProbeLayers || sp.V < 0 || sp.V >= stateProbeLayerValues(stateProbeMode, sp.Layer) {
		return stateProbeInfo{}, false
	}
	blk, state := sp.location()
	_, n := stateProbeBlockSpan(stateProbeMode, sp.Layer, blk)
	// States lists 0 to n-1 in order, so the index of a state is the state.
	return stateProbeInfo{name: stateProbeNames[sp.Layer][blk], state: state, index: state, count: n}, true
}

func (b StateProbe) EncodeBlock() (string, map[string]any) {
	blk, state := b.location()
	return stateProbeNames[b.Layer][blk], map[string]any{stateProbeState: int32(state)}
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
func stateProbeBase(mode string) customblock.Properties { return stateProbeBaseOf(mode, 0) }

// stateProbeBaseOf is the look of the value 0 of the block of a layer: the
// same in every mode but pair2, where each block has its own geometry.
func stateProbeBaseOf(mode string, layer int) customblock.Properties {
	switch {
	case stateProbePair2(mode):
		p := stateProbePair2Look(layer, fillKind{kind: 2 * layer, tier: 1}, fillKind{kind: 2*layer + 1, tier: 1})
		p.BoneVisibility = map[string]string{}
		for i := range 2 {
			for _, bone := range stateProbeKindBones(2*layer + i) {
				p.BoneVisibility[bone] = "false"
			}
		}
		return p
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
// in the mode (the same for both layers but in same4), conditions on the values
// of the look; all4split has them by block in stateProbeBlockPermutations.
func stateProbePermutations(mode string, layer int) []customblock.Permutation {
	return stateProbePermutationsOf(mode, layer, -1)
}

// stateProbeBlockPermutations are the permutations of block blk of a layer:
// all of them but in all4split, where the block has its own, with conditions on
// the states of the block.
func stateProbeBlockPermutations(mode string, layer, blk int) []customblock.Permutation {
	if !stateProbeSplit(mode) {
		blk = -1
	}
	return stateProbePermutationsOf(mode, layer, blk)
}

// stateProbePermutationsOf are the permutations of block blk (conditions on its
// states), or with blk -1 of all the values.
func stateProbePermutationsOf(mode string, layer, blk int) []customblock.Permutation {
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
	case stateProbePair2(mode):
		out = stateProbePair2Permutations(layer)
	case stateProbeFourKinds(mode):
		shift := 0
		if blk >= 0 {
			shift, _ = stateProbeBlockSpan(mode, layer, blk)
		}
		// singles: the faces are v - lo
		for kind := 0; blk <= 0 && kind < stateProbeKinds4; kind++ {
			for tier := 1; tier <= stateProbeTiers; tier++ {
				lo := 1 + kind*stateProbeTiers*stateProbeFaces + (tier-1)*stateProbeFaces
				p := stateProbeFourLook(stateProbeOneGeometry(kind), fillKind{kind: kind, tier: tier})
				p.BoneVisibility = stateProbeKindVisibility(kind, lo, 1)
				out = append(out, customblock.Permutation{Condition: stateProbeRangeCond(lo, stateProbeFaces), Properties: p})
			}
		}
		// pairs: n = v - lo; same faces: both kinds n; any: a floor(n/64), b mod(n, 64)
		pairs := stateProbeLayerPairs(mode, layer)
		if pairs == probeNoPairs || blk == 0 {
			break
		}
		span := stateProbePairSpan(pairs)
		unitA := 1
		if pairs == probeAnyPairs {
			unitA = stateProbeFaces
		}
		for pi, pair := range stateProbeKindPairs {
			if blk > 0 && pi != blk-1 {
				continue
			}
			for ta := 1; ta <= stateProbeTiers; ta++ {
				for tb := 1; tb <= stateProbeTiers; tb++ {
					lo := stateProbePairStart + (pi*stateProbeTierPairs+(ta-1)*stateProbeTiers+(tb-1))*span - shift
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

// stateProbePair2Look is the look of pair2 block layer with the textures of
// its two kinds (kinds a and b, the tier of an absent one is a placeholder: the
// geometry has the material instances of both kinds).
func stateProbePair2Look(layer int, a, b fillKind) customblock.Properties {
	return stateProbeFourLook(stateProbeTwoGeometry(2*layer, 2*layer+1), a, b)
}

// Molang parts of a pair2 value: a = floor(v/193), b = mod(v, 193).
var (
	stateProbePair2A = fmt.Sprintf("math.floor(%s / %d)", stateProbeShapeExpr, stateProbePair2Kind)
	stateProbePair2B = fmt.Sprintf("math.mod(%s, %d)", stateProbeShapeExpr, stateProbePair2Kind)
)

// stateProbePair2Cond is the condition of one kind of a pair2 block being at
// tier (0 absent), as a condition on the whole value: a is in 1+(tier-1)*64 to
// +63 (the value in a range), b is the same on mod(v, 193).
func stateProbePair2Cond(second bool, tier int) string {
	lo, n := 0, 1
	if tier > 0 {
		lo, n = 1+(tier-1)*stateProbeFaces, stateProbeFaces
	}
	if !second {
		return stateProbeRangeCond(lo*stateProbePair2Kind, n*stateProbePair2Kind)
	}
	if tier == 0 {
		return stateProbePair2B + " == 0"
	}
	return fmt.Sprintf("%s >= %d && %s <= %d", stateProbePair2B, lo, stateProbePair2B, lo+n-1)
}

// stateProbePair2Visibility is the bone_visibility of a kind of a pair2 block:
// hidden when absent, else the core and the arm of each bit of faces =
// floor(part - lo) with part the Molang expression of a or b.
func stateProbePair2Visibility(kind int, part string, tier int) map[string]string {
	bones := stateProbeKindBones(kind)
	out := map[string]string{}
	if tier == 0 {
		for _, b := range bones {
			out[b] = "false"
		}
		return out
	}
	out[bones[0]] = "true"
	lo := 1 + (tier-1)*stateProbeFaces
	for i := range stateProbeArms {
		out[bones[1+i]] = fmt.Sprintf("math.mod(math.floor((%s - %d) / %d), 2) == 1", part, lo, 1<<i)
	}
	return out
}

// stateProbePair2Permutations are the 15 permutations of pair2 block layer:
// for the tier of each kind (0 absent) but both absent.
func stateProbePair2Permutations(layer int) []customblock.Permutation {
	ka, kb := 2*layer, 2*layer+1
	var out []customblock.Permutation
	for ta := 0; ta <= stateProbeTiers; ta++ {
		for tb := 0; tb <= stateProbeTiers; tb++ {
			if ta == 0 && tb == 0 {
				continue
			}
			p := stateProbePair2Look(layer, fillKind{kind: ka, tier: max(ta, 1)}, fillKind{kind: kb, tier: max(tb, 1)})
			p.BoneVisibility = stateProbePair2Visibility(ka, stateProbePair2A, ta)
			maps.Copy(p.BoneVisibility, stateProbePair2Visibility(kb, stateProbePair2B, tb))
			out = append(out, customblock.Permutation{
				Condition:  stateProbePair2Cond(false, ta) + " && " + stateProbePair2Cond(true, tb),
				Properties: p,
			})
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
func (b StateProbe) Properties() customblock.Properties {
	return stateProbeBaseOf(stateProbeMode, b.Layer)
}

// stateProbeStateValues are the values of skygear:shape of each block of each
// layer, made once (all4 has 221,953 of them on a block).
var stateProbeStateValues = func() (out [stateProbeLayers][stateProbeMaxBlocks]func() []any) {
	for l := range out {
		for blk := range out[l] {
			out[l][blk] = stateProbeValuesOf(l, blk)
		}
	}
	return out
}()

func stateProbeValuesOf(layer, blk int) func() []any {
	return sync.OnceValue(func() []any {
		_, n := stateProbeBlockSpan(stateProbeMode, layer, blk)
		values := make([]any, n)
		for i := range values {
			values[i] = int32(i)
		}
		return values
	})
}

// States implements block.Permutable.
func (b StateProbe) States() map[string][]any {
	blk, _ := b.location()
	return map[string][]any{stateProbeState: stateProbeStateValues[b.Layer][blk]()}
}

// Permutations implements block.Permutable.
func (b StateProbe) Permutations() []customblock.Permutation {
	blk, _ := b.location()
	return stateProbeBlockPermutations(stateProbeMode, b.Layer, blk)
}

func (StateProbe) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return nil },
	}
}
