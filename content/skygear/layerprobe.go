package skygear

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// LayerProbe is a block for a manual test of the storage layers (/sgdebug
// layers): it draws a conduit icon, so the layer it stands on and its state
// show on screen. Kind (0 to 2) picks the icon (energy, item, fluid) and moves
// it to its lane, Alt (0 or 1) the tier 1 or tier 3 texture. Kind 3 draws
// nothing visible (the look of a conduit bundle, which is a non-air block with
// no geometry of its own); alt changes nothing there. It has no item
// and nothing in the game places it. The two make one state, Kind*2+Alt:
// TestCustomBlockStatesMatchTheDefinition allows one state property a block.
type LayerProbe struct{ Kind, Alt int }

const (
	probeState = "skygear:probe"
	probeKinds = 4
	probeAlts  = 2
)

var layerProbeHash = block.NextHash()

func init() {
	for k := 0; k < probeKinds; k++ {
		for a := 0; a < probeAlts; a++ {
			world.RegisterBlock(LayerProbe{Kind: k, Alt: a})
		}
	}
}

func (b LayerProbe) EncodeBlock() (string, map[string]any) {
	return "skygear:layer_probe", map[string]any{probeState: int32(b.Kind*probeAlts + b.Alt)}
}

func (b LayerProbe) Hash() (uint64, uint64) { return layerProbeHash, uint64(b.Kind*probeAlts + b.Alt) }

func (LayerProbe) Model() world.BlockModel { return model.Empty{} }

// probeHidden is the kind with the invisible look.
const probeHidden = 3

// probeKindNames are the conduit icons the kinds borrow ("hidden": none).
var probeKindNames = [probeKinds]string{"energy", "item", "fluid", "hidden"}

// probeLane is how far the icon of a kind moves sideways, in blocks (the lanes
// of a conduit bundle: energy left, item in the middle, fluid right).
var probeLane = [probeKinds]float64{-3.0 / 16, 0, 3.0 / 16, 0}

// probeProperties is the look of a kind and alt: the conduit icon of the kind
// with its tier 1 (alt 0) or tier 3 (alt 1) texture.
func probeProperties(kind, alt int) customblock.Properties {
	if kind == probeHidden {
		// What a conduit bundle draws: a transparent texture on the node geometry.
		return customblock.Properties{
			Geometry:     "geometry.skygear_conduit_node",
			Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_conduit_node")},
			SelectionBox: cube.Box(7.0/16, 7.0/16, 7.0/16, 9.0/16, 9.0/16, 9.0/16),
		}
	}
	tier := 1
	if alt == 1 {
		tier = 3
	}
	return customblock.Properties{
		Geometry:     "geometry.skygear.conduit_icon_" + probeKindNames[kind],
		Textures:     map[string]customblock.Material{"*": blockmodel.AlphaMaterial(fmt.Sprintf("skygear_conduit_%s_%d", probeKindNames[kind], tier))},
		Translation:  mgl64.Vec3{probeLane[kind]},
		SelectionBox: cube.Box(7.0/16, 7.0/16, 7.0/16, 9.0/16, 9.0/16, 9.0/16),
	}
}

// Properties is the look of kind 0, alt 0; Permutations has the others.
func (LayerProbe) Properties() customblock.Properties { return probeProperties(0, 0) }

// States implements block.Permutable.
func (LayerProbe) States() map[string][]any {
	var values []any
	for i := 0; i < probeKinds*probeAlts; i++ {
		values = append(values, int32(i))
	}
	return map[string][]any{probeState: values}
}

// Permutations implements block.Permutable: one look per kind and alt.
func (LayerProbe) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for k := 0; k < probeKinds; k++ {
		for a := 0; a < probeAlts; a++ {
			out = append(out, customblock.Permutation{
				Condition:  fmt.Sprintf("q.block_state('%s') == %d", probeState, k*probeAlts+a),
				Properties: probeProperties(k, a),
			})
		}
	}
	return out
}

func (LayerProbe) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0, BlastResistance: 0,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(item.Tool) bool { return false },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return nil },
	}
}
