package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/internal/skygear/machinerecipe"
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

// chestWithBundle puts a chest at pos and an item conduit bundle east of it
// whose west side is set.
func chestWithBundle(tx *world.Tx, pos cube.Pos, mode itemnet.Mode, extract, insert int) cube.Pos {
	tx.SetBlock(pos, block.NewChest(), nil)
	b := ConduitBundle{Item: 1}
	b.Modes[cube.FaceWest] = mode
	b.Extract[cube.FaceWest], b.Insert[cube.FaceWest] = extract, insert
	at := pos.Add(cube.Pos{1, 0, 0})
	tx.SetBlock(at, b, nil)
	b.refresh(at, tx)
	return at
}

// TestCopierCardOnItem: the settings live on the item and survive its NBT.
func TestCopierCardOnItem(t *testing.T) {
	c := &configCard{Kind: cardSide, Side: 4, Sides: []*conduitCard{{Mode: int(itemnet.Input), Extract: 5, Insert: 9}}}
	st := withCard(item.NewStack(sgitem.ConfigCopier{}, 1), c, "ko")
	back, ok := cardOf(st)
	if !ok || back.Kind != cardSide || back.Sides[0].Extract != 5 || back.Sides[0].Insert != 9 {
		t.Fatalf("card back: %+v %v", back, ok)
	}
	saved := item.WriteNBT(st, true)
	again, ok := cardOf(item.ReadNBT(saved, nil))
	if !ok || again.Sides[0].Mode != int(itemnet.Input) {
		t.Fatalf("card after NBT: %+v %v", again, ok)
	}
	if len(st.Lore()) == 0 {
		t.Fatal("the copier does not say what it holds")
	}
	empty := withCard(st, nil, "ko")
	if _, ok := cardOf(empty); ok {
		t.Fatal("an emptied copier still holds settings")
	}
}

// TestCopierConduitSide: one side's mode and channels go to a side of
// another bundle, and a click finds the plate or the bundle.
func TestCopierConduitSide(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		a := chestWithBundle(tx, machinePos, itemnet.Input, 5, 9)
		b := chestWithBundle(tx, machinePos.Add(cube.Pos{0, 0, 4}), itemnet.Off, 0, 0)
		// a click on the bundle's west face finds its side
		tg, ok := targetOf(tx, a, cube.FaceWest, mgl64Centre())
		if !ok || tg.kind != cardSide || tg.pos != a || tg.side != faceIndex(cube.FaceWest) {
			t.Fatalf("target %+v %v", tg, ok)
		}
		card, ok := readCard(tx, tg)
		if !ok {
			t.Fatal("nothing read")
		}
		dst, _ := targetOf(tx, b, cube.FaceWest, mgl64Centre())
		if !s.pasteCard(tx, dst, *card) {
			t.Fatal("not pasted")
		}
		got := tx.Block(b).(ConduitBundle)
		if got.Modes[cube.FaceWest] != itemnet.Input || got.Extract[cube.FaceWest] != 5 || got.Insert[cube.FaceWest] != 9 {
			t.Fatalf("pasted side: %v %v %v", got.Modes, got.Extract, got.Insert)
		}
		// a side card does not fit a drawer
		drawer := machinePos.Add(cube.Pos{0, 0, 8})
		tx.SetBlock(drawer, Drawer{}, nil)
		dt, ok := targetOf(tx, drawer, cube.FaceNorth, mgl64Centre())
		if !ok || s.pasteCard(tx, dt, *card) {
			t.Fatalf("a conduit side was pasted on a drawer (target %v)", ok)
		}
	})
}

// TestCopierWholeBundle: the body of a bundle copies the settings of all its
// sides with a plate, and pasting skips the sides the other bundle has no
// plate on.
func TestCopierWholeBundle(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.world.Do(func(tx *world.Tx) {
		a := chestWithBundle(tx, machinePos, itemnet.Both, 2, 3)
		b := chestWithBundle(tx, machinePos.Add(cube.Pos{0, 0, 4}), itemnet.Off, 0, 0)
		northBefore := tx.Block(b).(ConduitBundle).Modes[cube.FaceNorth]
		// a click on a face with no plate takes the whole bundle
		tg, ok := targetOf(tx, a, cube.FaceUp, mgl64Centre())
		if !ok || tg.kind != cardBundle {
			t.Fatalf("target %+v %v", tg, ok)
		}
		card, _ := readCard(tx, tg)
		dst, _ := targetOf(tx, b, cube.FaceUp, mgl64Centre())
		if !s.pasteCard(tx, dst, *card) {
			t.Fatal("not pasted")
		}
		got := tx.Block(b).(ConduitBundle)
		if got.Modes[cube.FaceWest] != itemnet.Both || got.Extract[cube.FaceWest] != 2 || got.Insert[cube.FaceWest] != 3 {
			t.Fatalf("pasted: %v %v %v", got.Modes, got.Extract, got.Insert)
		}
		if got.Modes[cube.FaceNorth] != northBefore {
			t.Fatal("a side without a plate changed")
		}
		// a bundle card on a single side takes the first side's settings
		side, _ := targetOf(tx, b, cube.FaceWest, mgl64Centre())
		if !s.pasteCard(tx, side, *card) {
			t.Fatal("a bundle card was not pasted on a side")
		}
	})
}

// TestCopierCrafterAndDrawer: an auto crafter's recipe and a drawer's item
// are copied; a drawer holding another item refuses.
func TestCopierCrafterAndDrawer(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.recipesFree = true
	crafterA := placeTestMachine(s, machinePos, KindCrafter, "")
	crafterB := placeTestMachine(s, machinePos.Add(cube.Pos{0, 0, 3}), KindCrafter, "")
	cr := machinerecipe.Crafters()[0]
	crafterA.setRecipe(cr)
	s.world.Do(func(tx *world.Tx) {
		src, ok := targetOf(tx, machinePos, cube.FaceNorth, mgl64Centre())
		if !ok || src.kind != cardCrafter {
			t.Fatalf("crafter target %+v %v", src, ok)
		}
		card, ok := readCard(tx, src)
		if !ok {
			t.Fatal("no recipe read")
		}
		dst, _ := targetOf(tx, machinePos.Add(cube.Pos{0, 0, 3}), cube.FaceNorth, mgl64Centre())
		if !s.pasteCard(tx, dst, *card) || crafterB.recipeID != cr.ID {
			t.Fatalf("recipe %q, want %q", crafterB.recipeID, cr.ID)
		}
		// an auto crafter with no recipe has nothing to copy
		empty := placeTestMachine(s, machinePos.Add(cube.Pos{0, 0, 6}), KindCrafter, "")
		_ = empty
		blank, _ := targetOf(tx, machinePos.Add(cube.Pos{0, 0, 6}), cube.FaceNorth, mgl64Centre())
		if _, ok := readCard(tx, blank); ok {
			t.Fatal("copied a recipe that is not there")
		}
		// drawers
		d1, d2, d3 := machinePos.Add(cube.Pos{3, 0, 0}), machinePos.Add(cube.Pos{3, 0, 2}), machinePos.Add(cube.Pos{3, 0, 4})
		tx.SetBlock(d1, Drawer{Item: item.NewStack(block.Dirt{}, 1), Count: 5}, nil)
		tx.SetBlock(d2, Drawer{}, nil)
		tx.SetBlock(d3, Drawer{Item: item.NewStack(block.Stone{}, 1), Count: 4}, nil)
		from, _ := targetOf(tx, d1, cube.FaceNorth, mgl64Centre())
		dcard, ok := readCard(tx, from)
		if !ok {
			t.Fatal("no drawer item read")
		}
		to, _ := targetOf(tx, d2, cube.FaceNorth, mgl64Centre())
		if !s.pasteCard(tx, to, *dcard) {
			t.Fatal("not pasted on the empty drawer")
		}
		if got := tx.Block(d2).(Drawer); !got.Locked || naming.ItemKey(got.Item.Item()) != naming.ItemKey(block.Dirt{}) {
			t.Fatalf("empty drawer after pasting: %+v", got)
		}
		other, _ := targetOf(tx, d3, cube.FaceNorth, mgl64Centre())
		if s.pasteCard(tx, other, *dcard) {
			t.Fatal("pasted on a drawer holding another item")
		}
	})
}

// TestCopierHandler: the copier in hand copies on a click and pastes while
// sneaking, and is emptied by a click on the air.
func TestCopierHandler(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		a := chestWithBundle(tx, machinePos, itemnet.Input, 5, 9)
		b := chestWithBundle(tx, machinePos.Add(cube.Pos{0, 0, 4}), itemnet.Off, 0, 0)
		p.SetHeldItems(item.NewStack(sgitem.ConfigCopier{}, 1), item.Stack{})
		held, off := p.HeldItems()
		s.useCopier(p, tx, a, cube.FaceWest, mgl64Centre(), held, off)
		held, _ = p.HeldItems()
		if _, ok := cardOf(held); !ok {
			t.Fatal("a click did not copy")
		}
		s.lastOpen = nil
		p.StartSneaking()
		held, off = p.HeldItems()
		s.useCopier(p, tx, b, cube.FaceWest, mgl64Centre(), held, off)
		if got := tx.Block(b).(ConduitBundle); got.Extract[cube.FaceWest] != 5 {
			t.Fatalf("a sneaking click did not paste: %v", got.Extract)
		}
		p.StopSneaking()
		s.lastOpen = nil
		held, off = p.HeldItems()
		s.clearCopier(p, held, off)
		held, _ = p.HeldItems()
		if _, ok := cardOf(held); ok {
			t.Fatal("a click on the air did not empty the copier")
		}
	})
}

// mgl64Centre is the middle of a face, where a plate sits.
func mgl64Centre() mgl64.Vec3 { return mgl64.Vec3{0.5, 0.5, 0.5} }
