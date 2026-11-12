package storage

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// registerKinds hands the machine frame the behaviour of the storage's
// machine kinds. The drawer controller and the drawer interface stand for
// their drawer set: a click deposits into it, hoppers and conduits put into
// and take from it, and the panel counts it. The drawer trim only joins
// drawers: a click does nothing and its panel has no lines. A machine of a
// set that was broken invalidates the networks.
func registerKinds() {
	broken := func(s *core.Session, _ *world.Tx, _ cube.Pos) { InvalidateNetworks(s) }
	set := machine.Behaviour{
		Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			UseController(s, p, tx, pos)
			return true
		},
		Insert: ControllerInsert, Peek: ControllerPeek, Extract: ControllerExtract,
		Lines:  Lines,
		Broken: broken,
	}
	machine.Register(machine.KindDrawerController, set)
	machine.Register(machine.KindDrawerInterface, set)
	machine.Register(machine.KindDrawerTrim, machine.Behaviour{
		Click:  func(*core.Session, *player.Player, *world.Tx, cube.Pos) bool { return true },
		Lines:  machine.NoLines,
		Broken: broken,
	})
}
