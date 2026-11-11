package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestHandsFreeLine: an auto miner mines the block under it, an item
// conduit carries the cobblestone into a crusher beside it, and the crusher's
// output goes through a hopper into a chest.
func TestHandsFreeLine(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	minerPos := cube.Pos{0, island.IslandY + 2, 0}
	conduitPos := minerPos.Side(cube.FaceEast)
	crusherPos := conduitPos.Side(cube.FaceEast)
	hopperPos, chestPos := crusherPos.Side(cube.FaceDown), crusherPos.Add(cube.Pos{0, -2, 0})
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(minerPos.Side(cube.FaceDown), block.Cobblestone{}, nil)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(hopperPos, h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
		c := conduit.Bundle{Item: 1}
		c.Modes[cube.FaceWest] = itemnet.Input
		c.Modes[cube.FaceEast] = itemnet.Output
		tx.SetBlock(conduitPos, c, nil)
	})
	placeTestMachine(s, minerPos, machine.KindMiner)
	placeTestMachine(s, crusherPos, machine.KindCrusher)
	runSteps(s, 120)
	s.World.Do(func(tx *world.Tx) {
		inv := tx.Block(chestPos).(block.Chest).Inventory(tx, chestPos)
		if !inv.ContainsItem(item.NewStack(block.Gravel{}, 1)) {
			t.Fatalf("crusher output never reached the chest: %v", inv.Items())
		}
	})
}
