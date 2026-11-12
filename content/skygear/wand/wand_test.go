package wand_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/preview"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
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
	s := coretest.Session(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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

// sentViews records what preview.Send sends while f runs.
func sentViews(t *testing.T, f func()) []preview.View {
	t.Helper()
	var got []preview.View
	old := preview.Send
	preview.Send = func(_ *player.Player, v preview.View) { got = append(got, v) }
	defer func() { preview.Send = old }()
	f()
	return got
}

// TestWandPreview: on a floor of slabs, the preview sends each cell on top
// of the floor the preview glass on layer 0 (the crosshair goes through it)
// and the slab it would get on layer 1, and hiding it sends air on both.
func TestWandPreview(t *testing.T) {
	s := coretest.Session(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
		// looking straight down at the middle of the floor
		p.Teleport(mid.Vec3Centre().Add(mgl64.Vec3{0, 2, 0}))
		p.Move(mgl64.Vec3{}, 0, 90-p.Rotation().Pitch())
		views := sentViews(t, func() { wand.ShowPreview(s.Session, p, tx) })
		if len(views) != 18 {
			t.Fatalf("%d views, want 2 for each of 9 cells: %v", len(views), views)
		}
		for _, v := range views {
			if v.Pos[1] != floor[1]+1 || v.Pos[0] < floor[0] || v.Pos[0] > floor[0]+2 || v.Pos[2] < floor[2] || v.Pos[2] > floor[2]+2 {
				t.Fatalf("view at %v, want on top of the floor", v.Pos)
			}
			var want world.Block = slab
			if v.Layer == 0 {
				want = preview.Glass{Colour: preview.WandColour}
			}
			if world.BlockRuntimeID(v.Block) != world.BlockRuntimeID(want) {
				t.Fatalf("layer %d at %v shows %v, want %v", v.Layer, v.Pos, v.Block, want)
			}
		}
		// the same look sends nothing again
		if again := sentViews(t, func() { wand.ShowPreview(s.Session, p, tx) }); len(again) != 0 {
			t.Fatalf("the same preview sent %d views again", len(again))
		}
		// hiding clears the liquid layer of a cell before its layer 0: the
		// client would move the block on the liquid layer up to an empty
		// layer 0 and keep it there
		hidden := sentViews(t, func() { wand.HidePreview(s.Session, p) })
		cleared := map[cube.Pos]bool{}
		for _, v := range hidden {
			if _, ok := v.Block.(block.Air); !ok {
				t.Fatalf("hiding sent layer %d %v at %v, want air", v.Layer, v.Block, v.Pos)
			}
			switch v.Layer {
			case 1:
				cleared[v.Pos] = true
			case 0:
				if !cleared[v.Pos] {
					t.Fatalf("hiding cleared layer 0 at %v before its liquid layer", v.Pos)
				}
			}
		}
		if len(hidden) != 18 {
			t.Fatalf("hiding sent %d views, want 2 for each of 9 cells", len(hidden))
		}
	})
}

// TestPreviewOverlap: where two previews show the same cell, hiding one
// shows the other's look again, and hiding both shows the real blocks; the
// vein marks a block on layer 1 only.
func TestPreviewOverlap(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		stone := cube.Pos{2, island.IslandY + 3, 2}
		tx.SetBlock(stone, block.Stone{}, nil)
		cell := stone.Side(cube.FaceUp)
		vein := tinker.VeinLooks([]cube.Pos{stone})
		if l := vein[stone]; l.Block != nil || l.Liquid != (preview.Glass{Colour: preview.VeinColour}) {
			t.Fatalf("vein look %+v, want glass on the liquid layer only", l)
		}
		got := sentViews(t, func() { preview.Set(s.Session, p, tx, "vein", vein) })
		if len(got) != 1 || got[0].Layer != 1 || got[0].Block != (preview.Glass{Colour: preview.VeinColour}) {
			t.Fatalf("vein sent %v, want the glass on layer 1", got)
		}

		plank := block.Planks{Wood: block.OakWood()}
		preview.Set(s.Session, p, tx, "a", map[cube.Pos]preview.Look{cell: preview.Fill(preview.WandColour, block.Stone{})})
		preview.Set(s.Session, p, tx, "b", map[cube.Pos]preview.Look{cell: preview.Fill(preview.WandColour, plank)})
		got = sentViews(t, func() { preview.Clear(s.Session, p, tx, "b") })
		if len(got) != 2 || got[1].Block != (block.Stone{}) || got[0].Block != (preview.Glass{Colour: preview.WandColour}) {
			t.Fatalf("hiding b sent %v, want a's look again", got)
		}
		got = sentViews(t, func() { preview.Clear(s.Session, p, tx, "a") })
		if len(got) != 2 {
			t.Fatalf("hiding a sent %v", got)
		}
		for _, v := range got {
			if _, ok := v.Block.(block.Air); !ok {
				t.Fatalf("hiding a sent %v, want air", v)
			}
		}
		got = sentViews(t, func() { preview.Clear(s.Session, p, tx, "vein") })
		if len(got) != 1 || got[0].Layer != 1 {
			t.Fatalf("hiding the vein sent %v, want only layer 1 back", got)
		}
		if preview.Shown(s.Session, p, "vein") || preview.Shown(s.Session, p, "a") {
			t.Fatal("a cleared preview still shown")
		}
	})
}

// TestWandNextOption: a press on the lock or the match goes to the next
// value, around to the first.
func TestWandNextOption(t *testing.T) {
	l := wand.LockNone
	for range wand.Locks {
		l = wand.NextLock(l)
	}
	if l != wand.LockNone || wand.NextLock(wand.LockNone) != wand.LockHorizontal {
		t.Fatalf("locks do not go round: %d", l)
	}
	if wand.NextMatch(wand.MatchExact) != wand.MatchSimilar || wand.NextMatch(wand.MatchAny) != wand.MatchExact {
		t.Fatal("matches do not go in the listed order")
	}
}
