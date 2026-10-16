package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/world"
)

// Machines that show whether they work (docs/tech/block-animation.md): the
// block has a state (activeState) and its working look uses the animated
// front texture (a flipbook strip in the pack, textures/flipbook_textures.json).
// The server only changes the state when the machine starts or stops working.

// animatedMachines are the machines with a working look.
var animatedMachines = map[string]bool{
	KindFurnace: true, KindCrusher: true, KindWasher: true, KindAlloy: true, KindSimulator: true,
	KindCrucible: true, KindCrafter: true, KindLab: true,
	KindMiner: true, KindHarvester: true, KindGenerator: true, KindLavaGenerator: true,
	KindVacuum: true, KindFeller: true,
}

// activeState is the block state of an animated machine.
const activeState = "dragonfactory:active"

// activeHold is how many ticks a machine keeps showing as working after its
// last work, so that a machine that waits a moment for its input does not
// flicker.
const activeHold = 40

func activeValue(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func activeStates() map[string][]any {
	return map[string][]any{activeState: {activeValue(false), activeValue(true)}}
}

// activePermutations is the working look of a machine: the animated texture
// "<name>_front_on" on the front of a machine with a front, "<name>_side_on" on
// the sides of a plain cube.
func activePermutations(kind string) []customblock.Permutation {
	var props customblock.Properties
	if cubeMachines[kind] {
		props = cubeProperties(kind)
		props.Textures["*"] = alphaMaterial("dragonfactory_" + kind + "_side_on")
	} else {
		props = frontCubeProperties(kind)
		props.Textures["south"] = alphaMaterial("dragonfactory_" + kind + "_front_on")
	}
	return []customblock.Permutation{{
		Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", activeState, activeValue(true)),
		Properties: props,
	}}
}

// showActive sets the block of an animated machine to its working or resting
// look when that changed.
func showActive(tx *world.Tx, pos cube.Pos, st *machineState, tick int64) {
	if st.worked != st.seenWorked {
		st.seenWorked, st.lastWorkTick = st.worked, tick
	}
	want := st.worked > 0 && tick-st.lastWorkTick < activeHold
	m, ok := tx.Block(pos).(Machine)
	if !ok || m.st != st || m.active == want {
		return
	}
	m.active = want
	tx.SetBlock(pos, m, nil)
}
