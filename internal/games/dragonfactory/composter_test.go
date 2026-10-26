package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// TestComposterLevels: the levels are at fixed counts (no chance): the
// seventh at the 32nd item.
func TestComposterLevels(t *testing.T) {
	want := map[int]int{0: 0, 4: 0, 5: 1, 9: 1, 10: 2, 14: 3, 19: 4, 23: 5, 28: 6, 31: 6, 32: 7}
	for n, level := range want {
		if got := composterLevel(n); got != level {
			t.Errorf("%d items: level %d, want %d", n, got, level)
		}
	}
}

// TestComposterFillsToDirt: 32 plant items fill a composter step by step,
// the full one makes dirt in composterDelay, and a click then gives one dirt
// and empties it; other items are not taken.
func TestComposterFillsToDirt(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{3, IslandY, 3}
		tx.SetBlock(pos, block.Composter{}, nil)
		sapling := item.NewStack(block.Sapling{Type: block.OakSapling()}, 64)
		if s.useComposter(p, tx, pos, item.NewStack(item.Stick{}, 1)) {
			t.Fatal("a stick was put in a composter")
		}
		_ = p.Inventory().SetItem(0, sapling)
		p.SetHeldItems(sapling, item.Stack{})
		for i := 1; i <= matterCapacity; i++ {
			held, _ := p.HeldItems()
			if !s.useComposter(p, tx, pos, held) {
				t.Fatalf("item %d was not taken", i)
			}
			if c := tx.Block(pos).(block.Composter); c.Level != composterLevel(i) {
				t.Fatalf("after %d items the level is %d, want %d", i, c.Level, composterLevel(i))
			}
		}
		if held, _ := p.HeldItems(); held.Count() != 64-matterCapacity {
			t.Fatalf("%d saplings left, want %d", held.Count(), 64-matterCapacity)
		}
		held, _ := p.HeldItems()
		// full: the dirt takes composterDelay
		if !s.useComposter(p, tx, pos, held) {
			t.Fatal("the full composter did not answer")
		}
		if p.Inventory().ContainsItem(item.NewStack(block.Dirt{}, 1)) || s.composterCount(pos, tx.Block(pos).(block.Composter)) != matterCapacity {
			t.Fatal("a full composter gave dirt at once")
		}
		s.state.ComposterReady[cropKey(pos)] = -1 // the time is up
		s.tickComposters(tx)
		if c := tx.Block(pos).(block.Composter); c.Level != composterReadyLevel {
			t.Fatalf("ready composter shows level %d", c.Level)
		}
		if !s.useComposter(p, tx, pos, held) {
			t.Fatal("the ready composter did not answer")
		}
		if c := tx.Block(pos).(block.Composter); c.Level != 0 || s.composterCount(pos, c) != 0 {
			t.Errorf("the composter is not empty: %+v", c)
		}
		if !p.Inventory().ContainsItem(item.NewStack(block.Dirt{}, 1)) {
			t.Error("no dirt was given")
		}
	})
}

// TestBarkFiberIsPlantMatter: the bark fiber goes into the composter.
func TestBarkFiberIsPlantMatter(t *testing.T) {
	if !plantMatter(item.NewStack(dfitem.BarkFiber{}, 1)) || !plantMatter(item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1)) {
		t.Error("fibers or leaves are not plant matter")
	}
	if plantMatter(item.NewStack(item.Stick{}, 1)) || plantMatter(item.Stack{}) {
		t.Error("a stick or nothing is plant matter")
	}
}

// TestHoppersFeedComposterAndBarrel: a hopper above puts plant items in a
// composter and a barrel; a hopper under a full composter takes the dirt
// into a chest (itemio, no chance, fixed counts).
func TestHoppersFeedComposterAndBarrel(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	compPos, barrelPos := cube.Pos{2, IslandY + 1, 2}, cube.Pos{6, IslandY + 1, 6}
	hopperAt := func(tx *world.Tx, pos cube.Pos, it item.Stack) {
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		_ = h.Inventory(tx, pos).SetItem(0, it)
		tx.SetBlock(pos, h, nil)
	}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(compPos, block.Composter{}, nil)
		tx.SetBlock(barrelPos, Barrel{Wood: "oak"}, nil)
		hopperAt(tx, compPos.Side(cube.FaceUp), item.NewStack(block.Leaves{Type: block.OakLeaves()}, 3))
		hopperAt(tx, barrelPos.Side(cube.FaceUp), item.NewStack(dfitem.BarkFiber{}, 2))
	})
	tick(s, 20*3)
	s.world.Do(func(tx *world.Tx) {
		if n := s.composterCount(compPos, tx.Block(compPos).(block.Composter)); n != 3 {
			t.Errorf("the composter holds %d, want 3", n)
		}
		if rec := s.barrelAt(barrelPos); rec == nil || rec.Count != 2 {
			t.Errorf("the barrel holds %v, want 2", rec)
		}
	})
	// A full composter with a hopper under it and a chest under that.
	full, chestPos := cube.Pos{10, IslandY + 3, 2}, cube.Pos{10, IslandY + 1, 2}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(full, block.Composter{Level: composterLevels}, nil)
		s.setComposterCount(full, matterCapacity)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		tx.SetBlock(full.Side(cube.FaceDown), h, nil)
		tx.SetBlock(chestPos, block.NewChest(), nil)
	})
	tick(s, 20*3)
	s.world.Do(func(tx *world.Tx) {
		if !tx.Block(chestPos).(block.Chest).Inventory(tx, chestPos).ContainsItem(item.NewStack(block.Dirt{}, 1)) {
			t.Error("the hopper did not take the dirt out of the full composter")
		}
		if c := tx.Block(full).(block.Composter); c.Level != 0 || s.composterCount(full, c) != 0 {
			t.Errorf("the composter was not emptied: %+v", c)
		}
	})
}
