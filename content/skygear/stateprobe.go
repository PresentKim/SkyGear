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
	"github.com/presentkim/skygear/assets"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// The state count probe is for a manual client test of conduits drawn as
// blocks: how much longer does a client take to join and load with the states
// of two blocks that draw two conduit kinds each, and how fast does it draw
// them? It is off unless the environment variable SKYGEAR_STATE_PROBE
// (assets.StateProbeEnv) is pair2, and off registers nothing and puts no probe
// file in the pack. (Measured: client_load_ms 4,005 off, 9,019 with pair2;
// 12,083 filled cells at 184 FPS, against 7 to 20 as entities.)
//
// pair2 has two fixed blocks of two kinds each: block 0 energy + item (kinds 0
// and 1), block 1 fluid + network (kinds 2 and 3), both with the bones
// technique: a geometry with the bones of its two kinds (<kind>_core and
// <kind>_down ... <kind>_east, state_probe_two_energy_item and
// state_probe_two_fluid_network, written by tools/stateprobegeo; the lane of a
// kind, energy -3, item 0, fluid +3 and network (+6, -6, +6) pixels, is built
// into it) whose faces name the material instance of their kind, which the
// permutation maps to the texture of the kind's tier (the conduit items'
// textures; the network conduit has none yet, so skygear_conduit_node, grey,
// stands in). Each block has one int state, skygear:shape (one state a block,
// see TestCustomBlockStatesMatchTheDefinition), the value a*193 + b, 0 to
// 37,248 (37,249 states, under the 65,536 of a name), a for its first kind and
// b for its second, each 0 (absent) or 1 + (tier-1)*64 + faces: tier 1 to 3
// and faces the arms as a 6-bit set in the order of conduit.Faces (down, up,
// north, south, west, east). Value 0 (both absent) is never placed.
//
// The 15 permutations of a block are (tier of a or absent) × (tier of b or
// absent) less (absent, absent); a = floor(v/193) and b = mod(v, 193) are
// decoded in Molang, and the arms are switched by bone_visibility (the
// Dragonfly patch customblock.Properties.BoneVisibility). The block is placed
// from storage layer 0 (see fillShape.cell).
//
// The client may know only the first 65,536 states of a block name (a state
// past it draws nothing and has no collision), so a block name must stay under
// stateProbeNameCap states; LogCustomBlockCounts warns of any that does not.

const (
	// stateProbeLayers is the number of probe blocks (the field Layer of a
	// StateProbe is the block, not the storage layer).
	stateProbeLayers = 2

	stateProbeState = "skygear:shape"
	stateProbeTiers = 3
	// a (kind, tier) is 64 values: the sets of 6 faces
	stateProbeFaces = 64

	// stateProbeKinds4 are the kinds: energy, item, fluid, network.
	stateProbeKinds4 = 4

	// stateProbeNameCap is the number of states of a block name the client is
	// thought to know at most.
	stateProbeNameCap = 65536

	// stateProbePair2Kind is the number of values of one kind of a pair2 block:
	// absent and 3 × 64.
	stateProbePair2Kind = 1 + stateProbeTiers*stateProbeFaces
	// stateProbePair2Values is the number of values (states) of a pair2 block.
	stateProbePair2Values = stateProbePair2Kind * stateProbePair2Kind
)

// stateProbePair2Salt is in the names of the two blocks. Dragonfly panics at
// start when two block states share a network block hash (FNV-1a 32 of the
// state's NBT), and a name is chosen so that its states meet none of the other
// blocks'. TestStateProbeNetworkHashes checks the names against the blocks
// registered: when a new block collides, take the next salt it accepts.
const stateProbePair2Salt = 0

// probeLook is what a value draws: n (0 to 2) conduits of different kinds,
// in the order of their kinds.
type probeLook struct {
	n int
	k [2]fillKind
}

// stateProbeOn reports whether the mode is pair2.
func stateProbeOn(mode string) bool { return mode == assets.StateProbePair2 }

// stateProbeLayerValues is the number of values of the state of a pair2 block
// (0 when the mode is off).
func stateProbeLayerValues(mode string) int {
	if stateProbeOn(mode) {
		return stateProbePair2Values
	}
	return 0
}

// stateProbePair2Part is the value of one kind in a pair2 block: 0 when
// absent, else 1 + (tier-1)*64 + faces.
func stateProbePair2Part(k fillKind) int { return 1 + (k.tier-1)*stateProbeFaces + k.faces }

// stateProbeDecode is the look of the value v of pair2 block layer: its first
// kind (a) then its second (b), those present; false when v is not a value.
func stateProbeDecode(layer, v int) (probeLook, bool) {
	if layer < 0 || layer >= stateProbeLayers || v < 0 || v >= stateProbePair2Values {
		return probeLook{}, false
	}
	var l probeLook
	for i, x := range [2]int{v / stateProbePair2Kind, v % stateProbePair2Kind} {
		if x == 0 {
			continue
		}
		x--
		l.k[l.n] = fillKind{kind: 2*layer + i, tier: x/stateProbeFaces + 1, faces: x % stateProbeFaces}
		l.n++
	}
	return l, true
}

// stateProbeEncode is the value of a look (0 to 2 conduits, in any order) on
// pair2 block layer; false when a conduit is not of the block's kinds, the
// block has one kind twice or a tier or faces is out of range.
func stateProbeEncode(layer int, l probeLook) (int, bool) {
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

// stateProbeMode is the mode read once at init (assets.StateProbeOff or
// StateProbePair2).
var stateProbeMode string

var stateProbeHash uint64

func init() {
	stateProbeMode = assets.StateProbeMode()
	if !stateProbeOn(stateProbeMode) {
		return
	}
	start := time.Now()
	for l := range stateProbeNames {
		stateProbeNames[l] = stateProbeBlockName(l)
	}
	stateProbeHash = block.NextHash()
	for l := 0; l < stateProbeLayers; l++ {
		for v := 0; v < stateProbePair2Values; v++ {
			world.RegisterBlock(StateProbe{Layer: l, V: v})
		}
	}
	took := time.Since(start)
	states, perms := stateProbeCounts(stateProbeMode)
	// the time and memory show what 74,498 states cost (the palette is built
	// later, when the server starts).
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	slog.Info("state probe on", "mode", stateProbeMode, "block_names", stateProbeLayers, "max_states_per_name", stateProbePair2Values,
		"states", states, "permutations", perms, "geometries", stateProbeGeometries(stateProbeMode),
		"register_ms", took.Milliseconds(), "heap_mb", mem.HeapAlloc>>20, "sys_mb", mem.Sys>>20)
}

// stateProbeCounts are the block states and the permutations the mode adds.
func stateProbeCounts(mode string) (states, permutations int) {
	if !stateProbeOn(mode) {
		return 0, 0
	}
	for l := 0; l < stateProbeLayers; l++ {
		states += stateProbePair2Values
		permutations += len(stateProbePermutations(l))
	}
	return states, permutations
}

// stateProbeNameCounts are the block names of the mode with their states and
// permutations, in the order of the blocks (nothing when off): what the
// registry holds for them once registered.
func stateProbeNameCounts(mode string) []blockNameCount {
	var out []blockNameCount
	if !stateProbeOn(mode) {
		return out
	}
	for l := 0; l < stateProbeLayers; l++ {
		out = append(out, blockNameCount{
			name: stateProbeBlockName(l), states: stateProbePair2Values,
			permutations: len(stateProbePermutations(l)),
		})
	}
	return out
}

// stateProbeGeometries is the number of geometries the probe blocks of the
// mode use.
func stateProbeGeometries(mode string) int {
	if !stateProbeOn(mode) {
		return 0
	}
	used := map[string]bool{}
	for l := 0; l < stateProbeLayers; l++ {
		used[stateProbeBase(l).Geometry] = true
		for _, p := range stateProbePermutations(l) {
			used[p.Geometry] = true
		}
	}
	return len(used)
}

// stateProbeBlock is pair2 block layer (0 or 1) with the value v of its state
// (0 to 37,248); false when the probe is off or the block or value is out of
// range.
func stateProbeBlock(layer int, v int) (world.Block, bool) {
	if !stateProbeOn(stateProbeMode) || layer < 0 || layer >= stateProbeLayers || v < 0 || v >= stateProbePair2Values {
		return nil, false
	}
	return StateProbe{Layer: layer, V: v}, true
}

// StateProbe is a block of the state count probe: Layer the block, 0 (energy
// + item) or 1 (fluid + network), V the value of its state skygear:shape. It
// has no item and nothing in the game places it but the debug command that
// fills with it.
type StateProbe struct{ Layer, V int }

// stateProbeNames are the names of the probe blocks, set at init.
var stateProbeNames [stateProbeLayers]string

// stateProbeBlockName is the name of pair2 block layer:
// skygear:state_probe2_<salt>_<block>.
func stateProbeBlockName(layer int) string {
	return fmt.Sprintf("skygear:state_probe2_%d_%d", stateProbePair2Salt, layer)
}

func (b StateProbe) EncodeBlock() (string, map[string]any) {
	return stateProbeNames[b.Layer], map[string]any{stateProbeState: int32(b.V)}
}

func (b StateProbe) Hash() (uint64, uint64) {
	return stateProbeHash, uint64(b.V)<<1 | uint64(b.Layer)
}

func (StateProbe) Model() world.BlockModel { return model.Empty{} }

// stateProbeCore is the selection box of every look: the core.
var stateProbeCore = cube.Box(6.5/16, 6.5/16, 6.5/16, 9.5/16, 9.5/16, 9.5/16)

var conduitKindNames = [3]string{"energy", "item", "fluid"}

// stateProbeKindNames4 are the kinds, also the names of their bones' prefix
// and material instances.
var stateProbeKindNames4 = [stateProbeKinds4]string{"energy", "item", "fluid", "network"}

// stateProbeTexture is the texture of a kind and tier. The network conduit has
// none yet: the grey conduit node stands in for every tier.
func stateProbeTexture(kind, tier int) string {
	if kind == 3 {
		return "skygear_conduit_node"
	}
	return fmt.Sprintf("skygear_conduit_%s_%d", conduitKindNames[kind], tier)
}

// stateProbeArms are the arm bones, bit i of the faces being the arm i (the
// order of conduit.Faces).
var stateProbeArms = [6]string{"down", "up", "north", "south", "west", "east"}

// stateProbeGeometry is the geometry of pair2 block layer, with the bones of
// its two kinds.
func stateProbeGeometry(layer int) string {
	return "geometry.skygear.state_probe_two_" + stateProbeKindNames4[2*layer] + "_" + stateProbeKindNames4[2*layer+1]
}

// stateProbeKindBones are the bones of a kind in the geometries: the core,
// then the arms in the order of the faces.
func stateProbeKindBones(kind int) []string {
	out := []string{stateProbeKindNames4[kind] + "_core"}
	for _, arm := range stateProbeArms {
		out = append(out, stateProbeKindNames4[kind]+"_"+arm)
	}
	return out
}

// stateProbeBase is the look of the value 0 of pair2 block layer: every bone
// hidden.
func stateProbeBase(layer int) customblock.Properties {
	p := stateProbeLook(layer, fillKind{kind: 2 * layer, tier: 1}, fillKind{kind: 2*layer + 1, tier: 1})
	p.BoneVisibility = map[string]string{}
	for i := range 2 {
		for _, bone := range stateProbeKindBones(2*layer + i) {
			p.BoneVisibility[bone] = "false"
		}
	}
	return p
}

// stateProbeLook is the look of pair2 block layer with the textures of its two
// kinds (kinds a and b; the tier of an absent one is a placeholder: the
// geometry has the material instances of both kinds). "*" takes the first, for
// faces without one.
func stateProbeLook(layer int, a, b fillKind) customblock.Properties {
	p := customblock.Properties{
		Geometry:     stateProbeGeometry(layer),
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial(stateProbeTexture(a.kind, a.tier))},
		SelectionBox: stateProbeCore,
	}
	for _, k := range []fillKind{a, b} {
		p.Textures[stateProbeKindNames4[k.kind]] = blockmodel.AlphaMaterial(stateProbeTexture(k.kind, k.tier))
	}
	return p
}

// stateProbeShapeExpr is the Molang expression of the state value.
const stateProbeShapeExpr = "q.block_state('" + stateProbeState + "')"

// stateProbeRangeCond is the condition of the values lo to lo+n-1.
func stateProbeRangeCond(lo, n int) string {
	return fmt.Sprintf("%s >= %d && %s <= %d", stateProbeShapeExpr, lo, stateProbeShapeExpr, lo+n-1)
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

// stateProbePermutations are the 15 permutations of pair2 block layer: for the
// tier of each kind (0 absent) but both absent.
func stateProbePermutations(layer int) []customblock.Permutation {
	ka, kb := 2*layer, 2*layer+1
	var out []customblock.Permutation
	for ta := 0; ta <= stateProbeTiers; ta++ {
		for tb := 0; tb <= stateProbeTiers; tb++ {
			if ta == 0 && tb == 0 {
				continue
			}
			p := stateProbeLook(layer, fillKind{kind: ka, tier: max(ta, 1)}, fillKind{kind: kb, tier: max(tb, 1)})
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

// Properties is the look of the value 0; Permutations has the others.
func (b StateProbe) Properties() customblock.Properties { return stateProbeBase(b.Layer) }

// stateProbeStateValues are the values of skygear:shape of a block, made once.
var stateProbeStateValues = sync.OnceValue(func() []any {
	values := make([]any, stateProbePair2Values)
	for i := range values {
		values[i] = int32(i)
	}
	return values
})

// States implements block.Permutable.
func (b StateProbe) States() map[string][]any {
	return map[string][]any{stateProbeState: stateProbeStateValues()}
}

// Permutations implements block.Permutable.
func (b StateProbe) Permutations() []customblock.Permutation {
	return stateProbePermutations(b.Layer)
}

func (StateProbe) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return nil },
	}
}
