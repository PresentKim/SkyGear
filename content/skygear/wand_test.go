package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// wandWall is a 3×3 wall of stone at z = 8, its north face free.
func wandWall(tx *world.Tx) cube.Pos {
	base := cube.Pos{3, island.IslandY + 1, 8}
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			tx.SetBlock(base.Add(cube.Pos{x, y, 0}), block.Stone{}, nil)
		}
	}
	return base.Add(cube.Pos{1, 1, 0}) // the middle
}

func countStone(p *player.Player) int {
	n := 0
	for _, st := range p.Inventory().Slots() {
		if _, ok := st.Item().(block.Stone); ok {
			n += st.Count()
		}
	}
	return n
}

// TestWandBuildsFace: a click places a block in front of every block of the
// face, takes them from the inventory, wears the wand, and undo takes them
// back.
func TestWandBuildsFace(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mid := wandWall(tx)
		// the wand goes in the held slot first, or it would take the stone's
		p.SetHeldItems(item.NewStack(sgitem.NewWand(sgitem.WandStone), 1), item.Stack{})
		_, _ = p.Inventory().AddItem(item.NewStack(block.Stone{}, 20))
		held, _ := p.HeldItems()
		wand.Use(s.Session, p, tx, mid, cube.FaceNorth, held)
		placed := 0
		for x := 3; x < 6; x++ {
			for y := island.IslandY + 1; y < island.IslandY+4; y++ {
				if _, ok := tx.Block(cube.Pos{x, y, 7}).(block.Stone); ok {
					placed++
				}
			}
		}
		if placed != 9 {
			t.Fatalf("placed %d blocks in front of the wall, want 9", placed)
		}
		if n := countStone(p); n != 11 {
			t.Fatalf("%d stone left, want 11", n)
		}
		held, _ = p.HeldItems()
		if got, want := held.Durability(), held.MaxDurability()-9; got != want {
			t.Fatalf("durability %d, want %d", got, want)
		}
		if n := wand.Undo(s.Session, p, tx); n != 9 {
			t.Fatalf("undo took %d blocks, want 9", n)
		}
		if _, ok := tx.Block(cube.Pos{4, island.IslandY + 2, 7}).(block.Air); !ok {
			t.Fatal("undo left a block")
		}
		if n := countStone(p); n != 20 {
			t.Fatalf("%d stone after undo, want 20", n)
		}
	})
}

// TestWandLock: the horizontal lock builds one row only.
func TestWandLock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		mid := wandWall(tx)
		_, _ = p.Inventory().AddItem(item.NewStack(block.Stone{}, 20))
		st := wand.WithOptions(item.NewStack(sgitem.NewWand(sgitem.WandIron), 1), wand.Options{Lock: wand.LockHorizontal}, "ko")
		got := wand.Plan(tx, p, mid, cube.FaceNorth, wand.OptionsOf(st), 27, func(cube.Pos) bool { return true }, nil)
		if len(got) != 3 {
			t.Fatalf("planned %d blocks, want 3", len(got))
		}
		for _, pl := range got {
			if pl.Pos[1] != mid[1] {
				t.Fatalf("planned %v off the row", pl.Pos)
			}
		}
		// the limit stops the face short
		if got := wand.Plan(tx, p, mid, cube.FaceNorth, wand.Options{}, 4, func(cube.Pos) bool { return true }, nil); len(got) != 4 {
			t.Fatalf("planned %d blocks with a limit of 4", len(got))
		}
		// no blocks in the inventory: nothing to place
		p.Inventory().Clear()
		if got := wand.Plan(tx, p, mid, cube.FaceNorth, wand.Options{}, 27, func(cube.Pos) bool { return true }, nil); len(got) != 0 {
			t.Fatalf("planned %d blocks with an empty inventory", len(got))
		}
	})
}

// TestWandPlaceable: machines and blocks without a collision box are not
// placed.
func TestWandPlaceable(t *testing.T) {
	for _, b := range []world.Block{machine.Machine{Kind: machine.KindGenerator}, block.Torch{}, block.NewChest(), block.Air{}} {
		if wand.Placeable(b) {
			t.Errorf("%T is placeable", b)
		}
	}
	for _, b := range []world.Block{block.Stone{}, block.Planks{Wood: block.OakWood()}, block.Slab{Block: block.Stone{}}, sgitem.CrushedBlock{Kind: sgitem.CrushedKinds[0]}} {
		if !wand.Placeable(b) {
			t.Errorf("%T is not placeable", b)
		}
	}
}
