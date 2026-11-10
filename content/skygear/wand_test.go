package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/wand"
	"github.com/presentkim/skygear/platform/account"
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
	for _, b := range []world.Block{block.Stone{}, block.Planks{Wood: block.OakWood()}, block.Slab{Block: block.Stone{}}, block.Stairs{Block: block.Stone{}}, block.WoodFence{Wood: block.OakWood()}, block.Glass{}, sgitem.CrushedBlock{Kind: sgitem.CrushedKinds[0]}} {
		if !wand.Placeable(b) {
			t.Errorf("%T is not placeable", b)
		}
	}
}

// TestAutofill: a hand left empty gets the same item from the inventory
// (the main inventory first), unless the player turned autofill off; a
// wand that breaks is replaced by another.
func TestAutofill(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		stone := item.NewStack(block.Stone{}, 1)
		_ = p.Inventory().SetItem(3, item.NewStack(block.Stone{}, 4))
		_ = p.Inventory().SetItem(20, item.NewStack(block.Stone{}, 7))
		if !core.RefillHand(p, stone) {
			t.Fatal("empty hand not refilled")
		}
		if held, _ := p.HeldItems(); held.Count() != 7 {
			t.Fatalf("hand holds %d, want the 7 of the main inventory", held.Count())
		}
		if core.RefillHand(p, stone) {
			t.Fatal("refilled a hand that is not empty")
		}

		// the wand takes from the main inventory first, so the hotbar
		// stack stays; a wand that breaks is replaced by another
		p.Inventory().Clear()
		mid := wandWall(tx)
		p.SetHeldItems(item.NewStack(sgitem.NewWand(sgitem.WandStone), 1).Damage(130), item.Stack{})
		_ = p.Inventory().SetItem(1, item.NewStack(block.Stone{}, 2))
		_ = p.Inventory().SetItem(20, item.NewStack(block.Stone{}, 10))
		_ = p.Inventory().SetItem(25, item.NewStack(sgitem.NewWand(sgitem.WandStone), 1))
		held, _ := p.HeldItems()
		wand.Use(s.Session, p, tx, mid, cube.FaceNorth, held)
		if st, _ := p.Inventory().Item(1); st.Count() != 2 {
			t.Fatalf("hotbar stack holds %d, want 2", st.Count())
		}
		if held, _ := p.HeldItems(); !sgitem.IsWand(held) || held.Durability() != held.MaxDurability() {
			t.Fatalf("hand after the wand broke: %v", held)
		}
		// turned off: nothing comes
		if _, err := account.Toggle(p, core.AutofillSetting); err != nil {
			t.Fatal(err)
		}
		p.SetHeldItems(item.Stack{}, item.Stack{})
		if core.RefillHand(p, stone) {
			t.Fatal("refilled with autofill off")
		}
	})
}

// TestAutofillPlace: placing the last block of the hand fills it again once
// the game has taken the block.
func TestAutofillPlace(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	hd := &handler{s: s}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Stone{}, 1), item.Stack{})
		_ = p.Inventory().SetItem(20, item.NewStack(block.Stone{}, 5))
		hd.HandleBlockPlace(player.NewEventContext(tx, p), cube.Pos{4, island.IslandY + 1, 4}, block.Stone{})
		p.SetHeldItems(item.Stack{}, item.Stack{}) // the game takes the placed block
	})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if held, _ := p.HeldItems(); held.Count() != 5 {
			t.Fatalf("hand holds %d after placing its last block, want 5", held.Count())
		}
	})
}

// TestWandPreview: on a floor of slabs, the preview shows the slab each
// spot on top of the floor gets on layer 0 and the green glass on layer 1,
// and restores the air of both layers; a click on a previewed spot (the
// client shows it as a block) is a click on the face the preview is of.
func TestWandPreview(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		can := func(cube.Pos) bool { return true }
		slab := block.Slab{Block: block.Planks{Wood: block.OakWood()}}
		floor := cube.Pos{3, island.IslandY + 5, 3}
		for x := 0; x < 3; x++ {
			for z := 0; z < 3; z++ {
				tx.SetBlock(floor.Add(cube.Pos{x, 0, z}), slab, nil)
			}
		}
		mid := floor.Add(cube.Pos{1, 0, 1})
		p.SetHeldItems(item.NewStack(sgitem.NewWand(sgitem.WandStone), 1), item.Stack{})
		_, _ = p.Inventory().AddItem(item.NewStack(slab, 20))
		got := wand.Plan(tx, p, mid, cube.FaceUp, wand.Options{}, 27, can, nil)
		if len(got) != 9 {
			t.Fatalf("planned %d slabs, want 9", len(got))
		}
		spots := map[cube.Pos]world.Block{}
		for _, pl := range got {
			if pl.Pos[1] != floor[1]+1 || pl.Pos[0] < floor[0] || pl.Pos[0] > floor[0]+2 || pl.Pos[2] < floor[2] || pl.Pos[2] > floor[2]+2 {
				t.Fatalf("slab planned at %v, want on top of the floor", pl.Pos)
			}
			spots[pl.Pos] = pl.Block
		}
		views := wand.PreviewViews(spots)
		if len(views) != 18 {
			t.Fatalf("%d views, want 2 per spot", len(views))
		}
		for _, v := range views {
			want := spots[v.Pos]
			if v.Layer == 1 {
				want = wand.PreviewGlass
			}
			if world.BlockRuntimeID(v.Block) != world.BlockRuntimeID(want) {
				t.Fatalf("layer %d at %v shows %v, want %v", v.Layer, v.Pos, v.Block, want)
			}
		}
		for _, v := range wand.RestoreViews(tx, []cube.Pos{got[0].Pos}) {
			if _, ok := v.Block.(block.Air); !ok {
				t.Fatalf("restore of layer %d shows %v, want air", v.Layer, v.Block)
			}
		}

		// looking straight down at the middle of the floor
		p.Teleport(mid.Vec3Centre().Add(mgl64.Vec3{0, 2, 0}))
		p.Move(mgl64.Vec3{}, 0, 90-p.Rotation().Pitch())
		wand.ShowPreview(s.Session, p, tx)
		spot := mid.Side(cube.FaceUp).Add(cube.Pos{1, 0, 0})
		if at, face := wand.Aimed(s.Session, p, spot, cube.FaceUp); at != mid || face != cube.FaceUp {
			t.Fatalf("a click on the previewed %v is for %v %v, want %v up", spot, at, face, mid)
		}
		if at, _ := wand.Aimed(s.Session, p, floor, cube.FaceNorth); at != floor {
			t.Fatalf("a click off the preview moved to %v", at)
		}
		wand.HidePreview(s.Session, p)
		if at, _ := wand.Aimed(s.Session, p, spot, cube.FaceUp); at != spot {
			t.Fatal("the hidden preview still takes clicks")
		}
	})
}
