package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// droppedItems counts the item entities in the world by item key.
func droppedItems(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if it, ok := e.(*entity.Ent); ok {
			if b, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
				out[itemKey(b.Item().Item())] += b.Item().Count()
			}
		}
	}
	return out
}

// TestStickKnocksPebbles: a stick used on grass drops pebbles now and then;
// on stone it does nothing.
func TestStickKnocksPebbles(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		grass, stone := cube.Pos{1, IslandY - 1, 1}, cube.Pos{2, IslandY - 1, 1}
		tx.SetBlock(grass, block.Grass{}, nil)
		tx.SetBlock(stone, block.Stone{}, nil)
		stickStack := item.NewStack(item.Stick{}, 1)
		if s.useStick(p, tx, stone, cube.FaceUp, stickStack) {
			t.Fatal("a stick on stone was handled")
		}
		for range 100 {
			delete(s.lastOpen, p.H())
			if !s.useStick(p, tx, grass, cube.FaceUp, stickStack) {
				t.Fatal("a stick on grass was not handled")
			}
		}
		got := droppedItems(tx)
		if n := got[itemKey(Pebble{Kind: PebbleStone})]; n < 10 || n > 50 {
			t.Fatalf("%d stone pebbles from 100 uses: %v", n, got)
		}
	})
}

// TestAxeStripsFibre: stripping a log with a stone axe drops fibres now and
// then; a wooden axe drops none.
func TestAxeStripsFibre(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		pos := cube.Pos{1, IslandY, 1}
		tx.SetBlock(pos, oakLog, nil)
		for range 50 {
			s.stripFibre(tx, pos, cube.FaceUp, item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1))
		}
		if droppedItems(tx)[itemKey(BarkFibre{})] != 0 {
			t.Fatal("a wooden axe stripped fibres")
		}
		for range 50 {
			s.stripFibre(tx, pos, cube.FaceUp, item.NewStack(item.Axe{Tier: item.ToolTierStone}, 1))
		}
		if n := droppedItems(tx)[itemKey(BarkFibre{})]; n < 20 {
			t.Fatalf("%d fibres from 50 stone axe strips", n)
		}
	})
}
