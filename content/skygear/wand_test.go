package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/autofill"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/preview"
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
		if !autofill.RefillHand(p, stone) {
			t.Fatal("empty hand not refilled")
		}
		if held, _ := p.HeldItems(); held.Count() != 7 {
			t.Fatalf("hand holds %d, want the 7 of the main inventory", held.Count())
		}
		if autofill.RefillHand(p, stone) {
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
		if _, err := account.Toggle(p, autofill.Setting); err != nil {
			t.Fatal(err)
		}
		p.SetHeldItems(item.Stack{}, item.Stack{})
		if autofill.RefillHand(p, stone) {
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
	hd := s.Handler(nil).(*core.PlayerHandler)
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

// TestPreviewGlassBlock: there is a preview glass for every colour, each
// drawing the vanilla stained glass of its colour, with collision and
// selection boxes of no size (a point off the corner, as an all-zero box
// means none set and the full cube), and none is an item to give. The wand
// shows green and the hammer's vein blue.
func TestPreviewGlassBlock(t *testing.T) {
	if preview.WandColour != item.ColourGreen() || preview.VeinColour != item.ColourBlue() {
		t.Fatalf("wand %v, vein %v, want green and blue", preview.WandColour, preview.VeinColour)
	}
	for _, c := range item.Colours() {
		g := preview.Glass{Colour: c}
		props := g.Properties()
		if !props.Cube {
			t.Fatalf("%v: properties %+v, want a cube", c, props)
		}
		for name, box := range map[string]cube.BBox{"collision": props.CollisionBox, "selection": props.SelectionBox} {
			if box == (cube.BBox{}) {
				t.Errorf("%v: %s box unset, the client would get the full cube", c, name)
			}
			if box.Width() != 0 || box.Height() != 0 || box.Length() != 0 {
				t.Errorf("%v: %s box %v has a size", c, name, box)
			}
		}
		if tex := props.Textures["*"].Encode()["texture"]; tex != preview.GlassTexture(c) {
			t.Errorf("%v: texture %v, want %s", c, tex, preview.GlassTexture(c))
		}
		name, _ := g.EncodeBlock()
		if _, ok := world.DefaultBlockRegistry.CustomBlocks()[name]; !ok {
			t.Errorf("%s not registered", name)
		}
		if _, ok := world.ItemByName(name, 0); ok {
			t.Errorf("%s is an item", name)
		}
	}
	for id := range giveItems() {
		if strings.Contains(id, "preview_glass") {
			t.Fatalf("/sgadmin give lists %s", id)
		}
	}
}
