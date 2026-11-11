package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestHoppersFeedComposterAndBarrel: a hopper above puts plant items in a
// composter and a barrel; a hopper under a full composter takes the dirt
// into a chest (itemio, no chance, fixed counts).
func TestHoppersFeedComposterAndBarrel(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	compPos, barrelPos := cube.Pos{2, island.IslandY + 1, 2}, cube.Pos{6, island.IslandY + 1, 6}
	hopperAt := func(tx *world.Tx, pos cube.Pos, it item.Stack) {
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		_ = h.Inventory(tx, pos).SetItem(0, it)
		tx.SetBlock(pos, h, nil)
	}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(compPos, block.Composter{}, nil)
		tx.SetBlock(barrelPos, farm.Barrel{Wood: "oak"}, nil)
		hopperAt(tx, compPos.Side(cube.FaceUp), item.NewStack(block.Leaves{Type: block.OakLeaves()}, 3))
		hopperAt(tx, barrelPos.Side(cube.FaceUp), item.NewStack(sgitem.BarkFiber{}, 2))
	})
	tick(s, 20*3)
	s.World.Do(func(tx *world.Tx) {
		if n := farm.ComposterCount(s.Session, compPos, tx.Block(compPos).(block.Composter)); n != 3 {
			t.Errorf("the composter holds %d, want 3", n)
		}
		if rec := farm.BarrelAt(s.Session, barrelPos); rec == nil || rec.Count != 2 {
			t.Errorf("the barrel holds %v, want 2", rec)
		}
	})
	// A full composter with a hopper under it and a chest under that.
	full, chestPos := cube.Pos{10, island.IslandY + 3, 2}, cube.Pos{10, island.IslandY + 1, 2}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(full, block.Composter{Level: farm.ComposterLevels}, nil)
		farm.SetComposterCount(s.Session, full, farm.MatterCapacity)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(full.Side(cube.FaceDown), h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
	})
	tick(s, 20*3)
	s.World.Do(func(tx *world.Tx) {
		if !tx.Block(chestPos).(block.Chest).Inventory(tx, chestPos).ContainsItem(item.NewStack(block.Dirt{}, 1)) {
			t.Error("the hopper did not take the dirt out of the full composter")
		}
		if c := tx.Block(full).(block.Composter); c.Level != 0 || farm.ComposterCount(s.Session, full, c) != 0 {
			t.Errorf("the composter was not emptied: %+v", c)
		}
	})
}

// TestHopperFeedsComposter: a vanilla hopper holding wheat seeds above a
// composter empties into it (through the composter adapter of itemio).
func TestHopperFeedsComposter(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	compPos := cube.Pos{4, island.IslandY + 1, 4}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(compPos, block.Composter{}, nil)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		hopPos := compPos.Side(cube.FaceUp)
		_ = h.Inventory(tx, hopPos).SetItem(0, item.NewStack(block.WheatSeeds{}, 2))
		tx.SetBlock(hopPos, h, nil)
	})
	tick(s, 240)
	s.World.Do(func(tx *world.Tx) {
		hopPos := compPos.Side(cube.FaceUp)
		if left := tx.Block(hopPos).(block.Hopper).Inventory(tx, hopPos).Slots()[0]; !left.Empty() {
			t.Errorf("the hopper still holds %v", left)
		}
		if n := farm.ComposterCount(s.Session, compPos, tx.Block(compPos).(block.Composter)); n != 2 {
			t.Errorf("the composter holds %d, want 2", n)
		}
	})
}
