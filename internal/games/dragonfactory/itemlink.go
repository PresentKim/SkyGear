package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/hopper"
	"github.com/presentkim/dragonfactory/internal/itemio"
)

// How the game's blocks move items: everything goes
// through internal/itemio. Machines, drawers and barrels implement its
// interfaces; the composter (a vanilla block) is served by an adapter; hoppers
// of every kind (the vanilla one through a Dragonfly patch hook, ours in
// hopper.go) move one item per step with hopper.StepHopper.

// registerItemIO connects the game's blocks to itemio and the vanilla hopper
// to the game's item transfer.
func registerItemIO() {
	hopper.RegisterItemIOOnce.Do(func() {
		itemio.RegisterAdapter(func(tx *world.Tx, _ cube.Pos, b world.Block) (itemio.Inserter, itemio.Extractor) {
			if _, ok := b.(block.Composter); ok {
				if s := activeGameSession(tx); s != nil {
					return composterIO{s}, composterIO{s}
				}
			}
			return nil, nil
		})
		block.HopperTransfer = func(h block.Hopper, pos cube.Pos, tx *world.Tx) (bool, bool) {
			return hopper.StepHopper(tx, pos, h.Inventory(tx, pos), h.Facing, cube.FaceUp)
		}
	})
}
