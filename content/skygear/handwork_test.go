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
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
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
		grass, stone := cube.Pos{1, island.IslandY - 1, 1}, cube.Pos{2, island.IslandY - 1, 1}
		tx.SetBlock(grass, block.Grass{}, nil)
		tx.SetBlock(stone, block.Stone{}, nil)
		stickStack := item.NewStack(item.Stick{}, 1)
		if handwork.UseStick(s.Session, p, tx, stone, cube.FaceUp, stickStack) {
			t.Fatal("a stick on stone was handled")
		}
		for range 20 {
			delete(handwork.LastStick.Of(s.Session), p.H())
			if !handwork.UseStick(s.Session, p, tx, grass, cube.FaceUp, stickStack) {
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
		handwork.UseStick(s.Session, p, tx, grass, cube.FaceUp, stickStack)
		handwork.UseStick(s.Session, p, tx, grass, cube.FaceUp, stickStack)
		total2 := 0
		for _, n := range droppedItems(tx) {
			total2 += n
		}
		if total2 != 20 {
			t.Errorf("double click gave %d pebbles, want 20 in all", total2)
		}
	})
}

// TestAxeStripsFiber: stripping a log with any axe, a wooden one too,
// drops fibers now and then; a bark stripper item counts like an axe, and
// one that opts out, or a non-stripper, drops none.
func TestAxeStripsFiber(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		pos := cube.Pos{1, island.IslandY, 1}
		tx.SetBlock(pos, fairy.OakLog, nil)
		fibers := func() int { return droppedItems(tx)[naming.ItemKey(sgitem.BarkFiber{})] }
		for _, it := range []world.Item{item.Stick{}, testStripper{strips: false}} {
			for range 50 {
				handwork.StripFiber(s.Session, nil, tx, pos, cube.FaceUp, item.NewStack(it, 1))
			}
			if n := fibers(); n != 0 {
				t.Fatalf("%T stripped %d fibers", it, n)
			}
		}
		last := 0
		for _, it := range []world.Item{item.Axe{Tier: item.ToolTierWood}, item.Axe{Tier: item.ToolTierStone}, testStripper{strips: true}} {
			for range 50 {
				handwork.StripFiber(s.Session, nil, tx, pos, cube.FaceUp, item.NewStack(it, 1))
			}
			if n := fibers() - last; n < 20 {
				t.Fatalf("%d fibers from 50 strips with %#v", n, it)
			}
			last = fibers()
		}
	})
}

// TestTinkerAxeStripsFiber: a tool-table axe strips a log like an axe, wears
// one durability and drops fibers; a tool-table pickaxe does neither.
func TestTinkerAxeStripsFiber(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		pos := cube.Pos{1, island.IslandY, 1}
		axe := tinker.Tool{Kind: tinker.ToolAxe, Head: "flint", Handle: "wood"}
		pick := tinker.Tool{Kind: tinker.ToolPickaxe, Head: "flint", Handle: "wood"}
		fibers := func() int { return droppedItems(tx)[naming.ItemKey(sgitem.BarkFiber{})] }

		tx.SetBlock(pos, fairy.OakLog, nil)
		var ctx item.UseContext
		if pick.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, nil, &ctx) {
			t.Fatal("a pickaxe stripped a log")
		}
		if l := tx.Block(pos).(block.Log); l.Stripped {
			t.Fatal("the log is stripped after a pickaxe")
		}
		if !axe.UseOnBlock(pos, cube.FaceUp, mgl64.Vec3{}, tx, nil, &ctx) || !tx.Block(pos).(block.Log).Stripped || ctx.Damage != 1 {
			t.Fatalf("a tinker axe did not strip the log for one durability (damage %d)", ctx.Damage)
		}

		tx.SetBlock(pos, fairy.OakLog, nil)
		for range 50 {
			handwork.StripFiber(s.Session, nil, tx, pos, cube.FaceUp, item.NewStack(pick, 1))
		}
		if n := fibers(); n != 0 {
			t.Fatalf("a tinker pickaxe stripped %d fibers", n)
		}
		for range 50 {
			handwork.StripFiber(s.Session, nil, tx, pos, cube.FaceUp, item.NewStack(axe, 1))
		}
		if n := fibers(); n < 20 {
			t.Fatalf("%d fibers from 50 tinker axe strips", n)
		}
	})
}

// testStripper is a non-axe item that strips bark when strips is set.
type testStripper struct{ strips bool }

func (testStripper) EncodeItem() (string, int16) { return "skygear:test_stripper", 0 }
func (t testStripper) StripsBark() bool          { return t.strips }

// TestDropsHaveMotion: dropAt pushes every drop; none starts dead still.
func TestDropsHaveMotion(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		for range 20 {
			s.DropAt(tx, cube.Pos{1, island.IslandY + 2, 1}.Vec3Centre(), item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 1))
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

// TestNoFiberString: bark fibers no longer make string (string comes from
// elsewhere).
func TestNoFiberString(t *testing.T) {
	for _, r := range recipe.Recipes() {
		for _, out := range r.Output() {
			if _, ok := out.Item().(block.String); !ok {
				continue
			}
			for _, in := range r.Input() {
				if st, ok := in.(item.Stack); ok {
					if _, fiber := st.Item().(sgitem.BarkFiber); fiber {
						t.Fatalf("fibers make string: %#v", r)
					}
				}
			}
		}
	}
}

// TestGrassSeeds: tilling dirt with a hoe sometimes drops grass seeds, and the
// seeds turn dirt into grass.
func TestGrassSeeds(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{4, island.IslandY + 2, 4}
		hoe := item.NewStack(item.Hoe{Tier: item.ToolTierWood}, 1)
		for range 200 {
			tx.SetBlock(pos, block.Dirt{}, nil)
			handwork.TillSeeds(s.Session, p, tx, pos, cube.FaceUp, hoe)
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
