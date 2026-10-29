package skygear

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
)

// The look of a machine tier: a machine that
// takes kits shows its tier with its own textures
// "<name>_<face>_t<tier>" (and "_on_t<tier>" for the working look), written
// to the pack by tools/tierblocks. The sieves keep one look for every tier.

// tierState is the one block state of a machine with tier looks (a block
// keeps the order of its state values certain only with a single state): its
// value counts the looks, tier*2 resting and tier*2+1 working for an animated
// machine, the tier alone for the others.
const tierState = "skygear:tier"

// tierLook reports whether the blocks of a kind show their tier.
func tierLook(kind string) bool {
	return tierable(kind) && !isSieve(kind) && (cubeMachines[kind] || frontMachines[kind])
}

// tierLookIndex is the state number of a tier and working look.
func tierLookIndex(kind string, tier int, on bool) int {
	if !animatedMachines[kind] {
		return tier
	}
	if on {
		return tier*2 + 1
	}
	return tier * 2
}

func tierValue(kind string, tier int, on bool) string {
	return strconv.Itoa(tierLookIndex(kind, tier, on))
}

func tierStates(kind string) map[string][]any {
	var vals []any
	for t := 0; t <= maxTier; t++ {
		vals = append(vals, tierValue(kind, t, false))
		if animatedMachines[kind] {
			vals = append(vals, tierValue(kind, t, true))
		}
	}
	return map[string][]any{tierState: vals}
}

// lookProperties returns the textures of a machine kind at a tier, resting or
// working: tier 0 uses the plain names, the others the "_t<tier>" ones.
func lookProperties(kind string, tier int, on bool) customblock.Properties {
	suffix := ""
	if tier > 0 {
		suffix = fmt.Sprintf("_t%d", tier)
	}
	tex := func(face string) customblock.Material {
		return blockmodel.AlphaMaterial("skygear_" + kind + "_" + face + suffix)
	}
	if cubeMachines[kind] {
		side := "side"
		if on {
			side = "side_on"
		}
		return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
			"*": tex(side), "up": tex("up"), "down": tex("down"),
		}}
	}
	face := "front"
	if on {
		face = "front_on"
	}
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*": tex("side"), "south": tex(face), "north": tex("back"), "up": tex("up"), "down": tex("down"),
	}}
}

// tierPermutations are the looks of a machine with tier looks: every state
// value but the resting tier 0 one (the base properties).
func tierPermutations(kind string) []customblock.Permutation {
	var perms []customblock.Permutation
	for t := 0; t <= maxTier; t++ {
		for _, on := range []bool{false, true} {
			if (t == 0 && !on) || (on && !animatedMachines[kind]) {
				continue
			}
			perms = append(perms, customblock.Permutation{
				Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", tierState, tierValue(kind, t, on)),
				Properties: lookProperties(kind, t, on),
			})
		}
	}
	return perms
}

// showTier sets the block of a machine to the look of its tier when that
// changed.
func showTier(tx *world.Tx, pos cube.Pos, st *machineState) {
	m, ok := tx.Block(pos).(Machine)
	if !ok || m.st != st || !tierLook(m.Kind) || m.tier == st.tier {
		return
	}
	m.tier = st.tier
	tx.SetBlock(pos, m, nil)
}
