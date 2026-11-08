package skygear

import (
	"fmt"
	"log/slog"

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
//     geometry (state_probe_shape_<faces>, 64 of them), texture and lane.
//   - bones (C): one geometry with a bone for the core and each arm, and one
//     permutation for each kind and tier (9 a block) that sets the geometry,
//     texture and lane and switches the arm bones by bone_visibility, a
//     Molang expression of the state (the Dragonfly patch
//     customblock.Properties.BoneVisibility).
//
// Two blocks, skygear:state_probe_0 and _1 (stand-ins for the two drawing
// layers), each have one int state skygear:shape (one state a block, see
// TestCustomBlockStatesMatchTheDefinition), 0 to 576. 0 is empty; v is
// 1 + kind*192 + (tier-1)*64 + faces: kind 0 to 2 (energy, item, fluid), tier
// 1 to 3 and faces the arms as a 6-bit set in the order of conduit.Faces
// (down, up, north, south, west, east). The look is a conduit of the kind and
// tier, in the lane of its kind, with the textures of the conduit items.

const (
	// stateProbeValues is the number of values of the state of a probe block.
	stateProbeValues = 577
	// stateProbeLayers is the number of probe blocks.
	stateProbeLayers = 2

	stateProbeState = "skygear:shape"
	stateProbeKinds = 3
	stateProbeTiers = 3
	// a (kind, tier) is 64 values: the sets of 6 faces
	stateProbeFaces = 64
)

// stateProbeMode is the mode read once at init (assets.StateProbeOff,
// StateProbePerm or StateProbeBones).
var stateProbeMode string

var stateProbeHash uint64

func init() {
	stateProbeMode = assets.StateProbeMode()
	if stateProbeMode == assets.StateProbeOff {
		return
	}
	stateProbeHash = block.NextHash()
	for l := 0; l < stateProbeLayers; l++ {
		for v := 0; v < stateProbeValues; v++ {
			world.RegisterBlock(StateProbe{Layer: l, V: v})
		}
	}
	states, perms := stateProbeCounts(stateProbeMode)
	slog.Info("state probe on", "mode", stateProbeMode, "states", states, "permutations", perms)
}

// stateProbeCounts are the block states and the permutations the mode adds.
func stateProbeCounts(mode string) (states, permutations int) {
	switch mode {
	case assets.StateProbePerm:
		return stateProbeLayers * stateProbeValues, stateProbeLayers * (stateProbeValues - 1)
	case assets.StateProbeBones:
		return stateProbeLayers * stateProbeValues, stateProbeLayers * stateProbeKinds * stateProbeTiers
	}
	return 0, 0
}

// stateProbeBlock is the probe block of a layer (0 or 1) with the value v of
// its state (0 to stateProbeValues-1); false when the probe is off or the
// layer or value is out of range.
func stateProbeBlock(layer int, v int) (world.Block, bool) {
	if stateProbeMode == assets.StateProbeOff || layer < 0 || layer >= stateProbeLayers || v < 0 || v >= stateProbeValues {
		return nil, false
	}
	return StateProbe{Layer: layer, V: v}, true
}

// StateProbe is a block of the state count probe: Layer 0 or 1, V the value of
// skygear:shape. It has no item and nothing in the game places it but the
// debug command that fills with it.
type StateProbe struct{ Layer, V int }

func (b StateProbe) EncodeBlock() (string, map[string]any) {
	return fmt.Sprintf("skygear:state_probe_%d", b.Layer), map[string]any{stateProbeState: int32(b.V)}
}

func (b StateProbe) Hash() (uint64, uint64) {
	return stateProbeHash, uint64(b.Layer*stateProbeValues + b.V)
}

func (StateProbe) Model() world.BlockModel { return model.Empty{} }

// stateProbeParts splits a value 1 to 576 into kind (0 to 2), tier (1 to 3)
// and the set of arms.
func stateProbeParts(v int) (kind, tier, faces int) {
	n := v - 1
	return n / (stateProbeTiers * stateProbeFaces), n%(stateProbeTiers*stateProbeFaces)/stateProbeFaces + 1, n % stateProbeFaces
}

// stateProbeLane is the shift of a kind in blocks, the same on every axis as
// the lanes of the conduit visuals (energy -3, item 0, fluid +3 pixels).
func stateProbeLane(kind int) mgl64.Vec3 {
	s := float64(kind-1) * 3 / 16
	return mgl64.Vec3{s, s, s}
}

// stateProbeCore is the selection box of every look: the core.
var stateProbeCore = cube.Box(6.5/16, 6.5/16, 6.5/16, 9.5/16, 9.5/16, 9.5/16)

// stateProbeLook is the look of a kind and tier: its texture, lane and core.
func stateProbeLook(geometry string, kind, tier int) customblock.Properties {
	return customblock.Properties{
		Geometry:     geometry,
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial(fmt.Sprintf("skygear_conduit_%s_%d", conduitKindNames[kind], tier))},
		Translation:  stateProbeLane(kind),
		SelectionBox: stateProbeCore,
	}
}

var conduitKindNames = [stateProbeKinds]string{"energy", "item", "fluid"}

// stateProbeArms are the arm bones of the geometry of mode bones, bit i of the
// faces being the arm i (the order of conduit.Faces).
var stateProbeArms = [6]string{"down", "up", "north", "south", "west", "east"}

// stateProbeShapeGeometry is the geometry of mode perm for a set of arms.
func stateProbeShapeGeometry(faces int) string {
	return fmt.Sprintf("geometry.skygear.state_probe_shape_%d", faces)
}

const stateProbeBonesGeometry = "geometry.skygear.state_probe_bones"

// stateProbeBase is the look of the value 0, empty. Mode perm has no geometry
// without a core to hide, so it takes the plain core and shrinks it out of
// sight; mode bones hides every bone.
func stateProbeBase(mode string) customblock.Properties {
	switch mode {
	case assets.StateProbeBones:
		p := stateProbeLook(stateProbeBonesGeometry, 1, 1)
		p.BoneVisibility = map[string]string{"core": "false"}
		for _, arm := range stateProbeArms {
			p.BoneVisibility[arm] = "false"
		}
		return p
	}
	p := stateProbeLook(stateProbeShapeGeometry(0), 1, 1)
	p.Scale = mgl64.Vec3{0.001, 0.001, 0.001}
	return p
}

// stateProbeShapeExpr is the Molang expression of the state value.
const stateProbeShapeExpr = "q.block_state('" + stateProbeState + "')"

// stateProbePermutations are the permutations of a probe block in the mode.
func stateProbePermutations(mode string) []customblock.Permutation {
	var out []customblock.Permutation
	switch mode {
	case assets.StateProbePerm:
		for v := 1; v < stateProbeValues; v++ {
			kind, tier, faces := stateProbeParts(v)
			out = append(out, customblock.Permutation{
				Condition:  fmt.Sprintf("%s == %d", stateProbeShapeExpr, v),
				Properties: stateProbeLook(stateProbeShapeGeometry(faces), kind, tier),
			})
		}
	case assets.StateProbeBones:
		for kind := 0; kind < stateProbeKinds; kind++ {
			for tier := 1; tier <= stateProbeTiers; tier++ {
				lo := 1 + kind*stateProbeTiers*stateProbeFaces + (tier-1)*stateProbeFaces
				p := stateProbeLook(stateProbeBonesGeometry, kind, tier)
				p.BoneVisibility = map[string]string{"core": "true"}
				for i, arm := range stateProbeArms {
					// bit i of (v - lo): v is in lo..lo+63, so the faces are v - lo
					p.BoneVisibility[arm] = fmt.Sprintf("math.mod(math.floor((%s - %d) / %d), 2) == 1", stateProbeShapeExpr, lo, 1<<i)
				}
				out = append(out, customblock.Permutation{
					Condition:  fmt.Sprintf("%s >= %d && %s <= %d", stateProbeShapeExpr, lo, stateProbeShapeExpr, lo+stateProbeFaces-1),
					Properties: p,
				})
			}
		}
	}
	return out
}

// Properties is the look of the value 0; Permutations has the others.
func (StateProbe) Properties() customblock.Properties { return stateProbeBase(stateProbeMode) }

// States implements block.Permutable.
func (StateProbe) States() map[string][]any {
	values := make([]any, stateProbeValues)
	for i := range values {
		values[i] = int32(i)
	}
	return map[string][]any{stateProbeState: values}
}

// Permutations implements block.Permutable.
func (StateProbe) Permutations() []customblock.Permutation {
	return stateProbePermutations(stateProbeMode)
}

func (StateProbe) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return nil },
	}
}
