package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
)

// droppedItems counts the item entities in the world by item key.
func droppedItems(tx *world.Tx) map[string]int {
	out := map[string]int{}
	for e := range tx.Entities() {
		if it, ok := e.(*entity.Ent); ok {
			if b, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
				out[naming.ItemKey(b.Item().Item())] += b.Item().Count()
			}
		}
	}
	return out
}

// TestStickKnocksPebbles: every stick use on grass drops a pebble, pushed
// in some direction (never dead still); on stone it does nothing.
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
		for range 20 {
			delete(s.lastStick, p.H())
			if !s.useStick(p, tx, grass, cube.FaceUp, stickStack) {
				t.Fatal("a stick on grass was not handled")
			}
		}
		total := 0
		for _, n := range droppedItems(tx) {
			total += n
		}
		if total != 20 {
			t.Fatalf("%d pebbles from 20 uses", total)
		}
		// A use right after the last one is the client's double click: ignored.
		s.useStick(p, tx, grass, cube.FaceUp, stickStack)
		s.useStick(p, tx, grass, cube.FaceUp, stickStack)
		total2 := 0
		for _, n := range droppedItems(tx) {
			total2 += n
		}
		if total2 != 20 {
			t.Errorf("double click gave %d pebbles, want 20 in all", total2)
		}
	})
}

// TestAxeStripsFiber: stripping a log with a stone axe drops fibers now and
// then; a wooden axe drops none.
func TestAxeStripsFiber(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		pos := cube.Pos{1, IslandY, 1}
		tx.SetBlock(pos, oakLog, nil)
		for range 50 {
			s.stripFiber(nil, tx, pos, cube.FaceUp, item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1))
		}
		if droppedItems(tx)[naming.ItemKey(sgitem.BarkFiber{})] != 0 {
			t.Fatal("a wooden axe stripped fibers")
		}
		for range 50 {
			s.stripFiber(nil, tx, pos, cube.FaceUp, item.NewStack(item.Axe{Tier: item.ToolTierStone}, 1))
		}
		if n := droppedItems(tx)[naming.ItemKey(sgitem.BarkFiber{})]; n < 20 {
			t.Fatalf("%d fibers from 50 stone axe strips", n)
		}
	})
}

// TestDropsHaveMotion: dropAt pushes every drop; none starts dead still.
func TestDropsHaveMotion(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		for range 20 {
			s.dropAt(tx, cube.Pos{1, IslandY + 2, 1}.Vec3Centre(), item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 1))
		}
		n := 0
		for e := range tx.Entities() {
			if e.H().Type() == entity.ItemType {
				n++
				if e.(*entity.Ent).Velocity().Len() == 0 {
					t.Error("a dropped item has no velocity")
				}
			}
		}
		if n != 20 {
			t.Errorf("%d items", n)
		}
	})
}

// TestStringRecipe: three fibers in a row make three string, and the
// recipe is three wide, so the 2x2 inventory grid cannot make it.
func TestStringRecipe(t *testing.T) {
	for _, r := range recipe.Recipes() {
		for _, out := range r.Output() {
			if _, ok := out.Item().(block.String); !ok {
				continue
			}
			sh, ok := r.(recipe.Shaped)
			if !ok || sh.Shape() != recipe.NewShape(3, 1) || out.Count() != 3 || len(sh.Input()) != 3 {
				t.Fatalf("string recipe %#v", r)
			}
			return
		}
	}
	t.Fatal("no string recipe")
}

// TestGrassSeeds: tilling dirt with a hoe sometimes drops grass seeds, and the
// seeds turn dirt into grass.
func TestGrassSeeds(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, IslandY + 2, 4}
		hoe := item.NewStack(item.Hoe{Tier: item.ToolTierWood}, 1)
		for range 200 {
			tx.SetBlock(pos, block.Dirt{}, nil)
			s.tillSeeds(p, tx, pos, cube.FaceUp, hoe)
		}
		if n := droppedKeys(tx)[naming.ItemKey(sgitem.GrassSeeds{})]; n < 5 || n > 40 {
			t.Fatalf("200 tillings dropped %d seeds, want about 20", n)
		}
		ctx := &item.UseContext{}
		if !(sgitem.GrassSeeds{}).UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) || ctx.CountSub != 1 {
			t.Fatal("seeds were not used on dirt")
		}
		if _, ok := tx.Block(pos).(block.Grass); !ok {
			t.Fatalf("seeds made %T", tx.Block(pos))
		}
	})
}
