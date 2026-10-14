package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
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

// TestComposterFillsToDirt: 32 plant items fill a composter step by step and
// a click on the full one gives one dirt and empties it; other items are
// not taken.
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
		if !s.useComposter(p, tx, pos, held) {
			t.Fatal("the full composter did not answer")
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
	if !plantMatter(item.NewStack(BarkFiber{}, 1)) || !plantMatter(item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1)) {
		t.Error("fibers or leaves are not plant matter")
	}
	if plantMatter(item.NewStack(item.Stick{}, 1)) || plantMatter(item.Stack{}) {
		t.Error("a stick or nothing is plant matter")
	}
}
