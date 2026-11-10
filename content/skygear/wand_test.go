package skygear

import (
	"strings"
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
	"github.com/presentkim/skygear/content/skygear/preview"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
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
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
				want = preview.Glass{}
			}
			if world.BlockRuntimeID(v.Block) != world.BlockRuntimeID(want) {
				t.Fatalf("layer %d at %v shows %v, want %v", v.Layer, v.Pos, v.Block, want)
			}
		}
		// the same look sends nothing again
		if again := sentViews(t, func() { wand.ShowPreview(s.Session, p, tx) }); len(again) != 0 {
			t.Fatalf("the same preview sent %d views again", len(again))
		}
		for _, v := range sentViews(t, func() { wand.HidePreview(s.Session, p) }) {
			if _, ok := v.Block.(block.Air); !ok {
				t.Fatalf("hiding sent layer %d %v at %v, want air", v.Layer, v.Block, v.Pos)
			}
		}
	})
}

// TestPreviewOverlap: where two previews show the same cell, hiding one
// shows the other's look again, and hiding both shows the real blocks; the
// vein marks a block on layer 1 only.
func TestPreviewOverlap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		stone := cube.Pos{2, island.IslandY + 3, 2}
		tx.SetBlock(stone, block.Stone{}, nil)
		cell := stone.Side(cube.FaceUp)
		vein := tinker.VeinLooks([]cube.Pos{stone})
		if l := vein[stone]; l.Block != nil || l.Liquid != (preview.Glass{}) {
			t.Fatalf("vein look %+v, want glass on the liquid layer only", l)
		}
		got := sentViews(t, func() { preview.Set(s.Session, p, tx, "vein", vein) })
		if len(got) != 1 || got[0].Layer != 1 || got[0].Block != (preview.Glass{}) {
			t.Fatalf("vein sent %v, want the glass on layer 1", got)
		}

		plank := block.Planks{Wood: block.OakWood()}
		preview.Set(s.Session, p, tx, "a", map[cube.Pos]preview.Look{cell: preview.Fill(block.Stone{})})
		preview.Set(s.Session, p, tx, "b", map[cube.Pos]preview.Look{cell: preview.Fill(plank)})
		got = sentViews(t, func() { preview.Clear(s.Session, p, tx, "b") })
		if len(got) != 2 || got[1].Block != (block.Stone{}) || got[0].Block != (preview.Glass{}) {
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

// TestPreviewGlassBlock: the preview glass is registered, has no
// collision or selection box, draws the vanilla green stained glass and is
// no item to give.
func TestPreviewGlassBlock(t *testing.T) {
	props := preview.Glass{}.Properties()
	if !props.NoCollision || !props.NoSelection || !props.Cube {
		t.Fatalf("properties %+v, want a cube with no boxes", props)
	}
	if tex := props.Textures["*"].Encode()["texture"]; tex != preview.GlassTexture {
		t.Fatalf("texture %v, want %s", tex, preview.GlassTexture)
	}
	if _, ok := world.DefaultBlockRegistry.CustomBlocks()["skygear:preview_glass"]; !ok {
		t.Fatal("preview glass not registered")
	}
	for id := range giveItems() {
		if strings.Contains(id, "preview_glass") {
			t.Fatalf("/sgadmin give lists %s", id)
		}
	}
	if _, ok := world.ItemByName("skygear:preview_glass", 0); ok {
		t.Fatal("preview glass is an item")
	}
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
