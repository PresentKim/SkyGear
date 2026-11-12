package machine

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// How the client draws a kind. A kind has one of three shapes, and its
// states, permutations, base properties and extra components all follow from
// that shape (shapeOf); the four methods below are what Dragonfly and
// clientdef ask a custom block.

// shape is how the client draws the blocks of a kind.
type shape int

const (
	// shapeCube is a full cube with textures by face; some kinds have a look
	// per tier or a working look as block states (tierlook.go, animate.go).
	shapeCube shape = iota
	// shapeSieve is the sieve model with a state for what lies on its mesh
	// (sieve.go).
	shapeSieve
	// shapeNet is a thin network block with a state for its arms and facing
	// (netshape.go).
	shapeNet
)

// shapeOf is the shape of a kind.
func shapeOf(kind string) shape {
	switch {
	case IsSieve(kind):
		return shapeSieve
	case NetShaped(kind):
		return shapeNet
	}
	return shapeCube
}

// The textures of the cubes. cubeMachines are drawn with one side texture
// and frontMachines with a front (south), back and side textures, both named
// after the kind; plainCubes have the one named texture on every face.
var (
	cubeMachines  = map[string]bool{KindMiner: true, KindHarvester: true, KindGenerator: true, KindEnergyCell: true, KindVacuum: true, KindFeller: true, KindTank: true, KindLavaGenerator: true, KindSolarGenerator: true, KindHydroGenerator: true, KindDrawerController: true, KindDrawerInterface: true, KindTinker: true, KindCollector: true, KindRelicTable: true, KindNetController: true, KindDiskDrive: true, KindNetTerminal: true, KindPatternEncoder: true}
	frontMachines = map[string]bool{KindCrusher: true, KindWasher: true, KindAlloy: true, KindSimulator: true, KindCrucible: true, KindFurnace: true, KindCrafter: true, KindLab: true, KindNetAutocrafter: true}
	plainCubes    = map[string]string{KindDrawerTrim: "skygear_drawer_side"}
)

// cubeProperties are the base properties of a cube kind.
func cubeProperties(kind string) customblock.Properties {
	if tex, ok := plainCubes[kind]; ok {
		return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.AlphaMaterial(tex)}}
	}
	if cubeMachines[kind] {
		return blockmodel.CubeProperties(kind)
	}
	return blockmodel.FrontCubeProperties(kind)
}

// States implements block.Permutable.
func (m Machine) States() map[string][]any {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return SieveStates()
	case shapeNet:
		return netStates(m.Kind)
	}
	if TierLook(m.Kind) {
		return tierStates(m.Kind)
	}
	if AnimatedKinds[m.Kind] {
		return activeStates()
	}
	return nil
}

// Permutations implements block.Permutable.
func (m Machine) Permutations() []customblock.Permutation {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sievePermutations(m.Kind)
	case shapeNet:
		return netPermutations(m.Kind)
	}
	if TierLook(m.Kind) {
		return tierPermutations(m.Kind)
	}
	if AnimatedKinds[m.Kind] {
		return activePermutations(m.Kind)
	}
	return nil
}

// Properties are the base properties of the kind's block.
func (m Machine) Properties() customblock.Properties {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sieveProperties(m.Kind)
	case shapeNet:
		return netProperties(m.Kind, 0)
	}
	return cubeProperties(m.Kind)
}

// ClientExtras implements clientdef.ExtraProperties for the kinds whose
// geometry has bones: a sieve shows its content bones, a network block its
// shape's bones and collision boxes. A cube has none.
func (m Machine) ClientExtras() (clientdef.Extras, map[string]clientdef.Extras) {
	switch shapeOf(m.Kind) {
	case shapeSieve:
		return sieveExtras()
	case shapeNet:
		return netAllExtras(m.Kind)
	}
	return clientdef.Extras{}, nil
}
