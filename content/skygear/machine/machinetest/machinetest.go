// Package machinetest places machines for tests. Only test files import it;
// a test file of the machine package itself must be in the package
// machine_test to do so.
package machinetest

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// Pos is where the tests place their machine: on the island's level, inside
// the chunks coretest.LoadAround loads.
var Pos = cube.Pos{3, island.IslandY, 3}

// Place places a machine of kind at pos in s's world and returns its state.
// The machine needs no energy, and a sieve has the string mesh fitted.
func Place(s core.Lifecycle, pos cube.Pos, kind string) *machine.State {
	m := machine.New(kind)
	m.State.Unlimited = true
	if machine.IsSieve(kind) {
		m.State.Mesh = "string"
	}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
	return m.State
}
