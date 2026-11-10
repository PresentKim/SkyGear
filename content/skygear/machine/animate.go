package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/world"
)

// Machines that show whether they work: the
// block has a state (activeState) and its working look uses the animated
// front texture (a flipbook strip in the pack, textures/flipbook_textures.json).
// The server only changes the state when the machine starts or stops working.

func init() {
	for kind := range AnimatedKinds {
		Kinds[kind].Animated = true
	}
}

// AnimatedKinds are the machines with a working look.
var AnimatedKinds = map[string]bool{
	KindFurnace: true, KindCrusher: true, KindWasher: true, KindAlloy: true, KindSimulator: true,
	KindCrucible: true, KindCrafter: true, KindLab: true,
	KindMiner: true, KindHarvester: true, KindGenerator: true, KindLavaGenerator: true, KindSolarGenerator: true, KindHydroGenerator: true,
	KindVacuum: true, KindFeller: true,
}

// activeState is the block state of an animated machine.
const activeState = "skygear:active"

// ActiveHold is how many ticks a machine keeps showing as working after its
// last work, so that a machine that waits a moment for its input does not
// flicker.
const ActiveHold = 40

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
	return []customblock.Permutation{{
		Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", activeState, activeValue(true)),
		Properties: lookProperties(kind, 0, true),
	}}
}

// showActiveOf is showActive for a caller that already holds the block at
// pos (a ticked machine), saving the lookup. A zero m (no machine there) does
// nothing but track the work.
func showActiveOf(tx *world.Tx, pos cube.Pos, m Machine, st *State, tick int64) {
	if st.worked != st.seenWorked {
		st.seenWorked, st.lastWorkTick = st.worked, tick
	}
	want := st.worked > 0 && tick-st.lastWorkTick < ActiveHold
	if m.State != st || m.Active == want {
		return
	}
	m.Active = want
	tx.SetBlock(pos, m, nil)
}
