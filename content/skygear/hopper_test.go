package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/hopper"
)

func chestCount(tx *world.Tx, pos cube.Pos, it item.Stack) int {
	n := 0
	for _, st := range tx.Block(pos).(block.Chest).Inventory(tx, pos).Slots() {
		if !st.Empty() && st.Comparable(it) {
			n += st.Count()
		}
	}
	return n
}

// TestWoodHopperIsSlowAndOneSlot: a wood hopper under a chest passes the
// items on to a chest below, one per four seconds, holding one at a time.
func TestWoodHopperIsSlowAndOneSlot(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	top, mid, bottom := cube.Pos{2, island.IslandY + 4, 2}, cube.Pos{2, island.IslandY + 3, 2}, cube.Pos{2, island.IslandY + 2, 2}
	wheat := item.NewStack(item.Wheat{}, 1)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(top, block.NewChest(), nil)
		_, _ = tx.Block(top).(block.Chest).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 3))
		tx.SetBlock(mid, hopper.NewFlowHopper("wood_hopper", cube.FaceDown), nil)
		tx.SetBlock(bottom, block.NewChest(), nil)
		h := tx.Block(mid).(hopper.FlowHopper)
		if n := h.InsertItems(tx, mid, cube.FaceUp, item.NewStack(item.Wheat{}, 2)); n != 2 {
			t.Fatalf("a wood hopper took %d wheat, want both (one slot, a stack)", n)
		}
		if n := h.InsertItems(tx, mid, cube.FaceUp, item.NewStack(item.Stick{}, 1)); n != 0 {
			t.Errorf("a second kind fit into the one slot: %d", n)
		}
		_ = h.Inv.SetItem(0, item.Stack{})
	})
	tick(s, 60)
	s.World.Do(func(tx *world.Tx) {
		if n := chestCount(tx, bottom, wheat); n > 1 {
			t.Errorf("%d wheat passed in 3 seconds, a wood hopper moves one per 4", n)
		}
	})
	tick(s, 20*30)
	s.World.Do(func(tx *world.Tx) {
		if n := chestCount(tx, bottom, wheat); n != 3 {
			t.Errorf("%d wheat arrived, want 3", n)
		}
	})
}

// TestReverseHopperMovesUp: a reverse hopper takes from the block below and
// gives to the block above, and does nothing the other way.
func TestReverseHopperMovesUp(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	below, mid, above := cube.Pos{4, island.IslandY + 2, 4}, cube.Pos{4, island.IslandY + 3, 4}, cube.Pos{4, island.IslandY + 4, 4}
	side := cube.Pos{5, island.IslandY + 3, 4}
	wheat := item.NewStack(item.Wheat{}, 1)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(below, block.NewChest(), nil)
		_, _ = tx.Block(below).(block.Chest).Inventory(tx, below).AddItem(item.NewStack(item.Wheat{}, 4))
		tx.SetBlock(mid, hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp), nil)
		tx.SetBlock(above, block.NewChest(), nil)
		tx.SetBlock(side, block.NewChest(), nil)
	})
	tick(s, 60)
	s.World.Do(func(tx *world.Tx) {
		if n := chestCount(tx, above, wheat); n < 3 {
			t.Errorf("%d wheat went up, want at least 3 in three seconds", n)
		}
		if n := chestCount(tx, side, wheat); n != 0 {
			t.Errorf("%d wheat went sideways", n)
		}
	})
}

// TestHopperNBT: a hopper keeps its items through its NBT and drops them
// with itself.
func TestHopperNBT(t *testing.T) {
	h := hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp)
	_ = h.Inv.SetItem(0, item.NewStack(item.Wheat{}, 5))
	_ = h.Inv.SetItem(3, item.NewStack(item.Stick{}, 2))
	back := h.DecodeNBT(h.EncodeNBT()).(hopper.FlowHopper)
	if got, _ := back.Inv.Item(0); got.Count() != 5 {
		t.Errorf("slot 0 after NBT: %v", got)
	}
	if got, _ := back.Inv.Item(3); got.Count() != 2 {
		t.Errorf("slot 3 after NBT: %v", got)
	}
	if drops := back.BreakInfo().Drops(nil, nil); len(drops) != 3 {
		t.Errorf("%d drops, want the hopper and its two stacks", len(drops))
	}
}

// TestHopperTurnRecipes: a hopper alone crafts into its reverse and the
// reverse alone into the hopper, for the wood and the iron one.
func TestHopperTurnRecipes(t *testing.T) {
	pairs := [][2]world.Item{
		{block.NewHopper(), hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp)},
		{hopper.NewFlowHopper("wood_hopper", cube.FaceDown), hopper.NewFlowHopper("wood_hopper_reverse", cube.FaceUp)},
	}
	has := func(in, out world.Item) bool {
		for _, r := range recipe.Recipes() {
			if _, ok := r.(recipe.Shapeless); !ok || len(r.Input()) != 1 || len(r.Output()) != 1 {
				continue
			}
			i, ok := r.Input()[0].(item.Stack)
			if ok && progress.RecipeKey(i.Item()) == progress.RecipeKey(in) && progress.RecipeKey(r.Output()[0].Item()) == progress.RecipeKey(out) {
				return true
			}
		}
		return false
	}
	for _, p := range pairs {
		if !has(p[0], p[1]) || !has(p[1], p[0]) {
			t.Errorf("no turn recipes between %s and %s", progress.RecipeKey(p[0]), progress.RecipeKey(p[1]))
		}
	}
}

// TestHopperLinesAndVanillaHopperFeedsIt: a vanilla hopper passes items into
// a wood hopper (itemio sees it as any block with an inventory).
func TestVanillaHopperFeedsWoodHopper(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	vanilla, wood := cube.Pos{8, island.IslandY + 3, 8}, cube.Pos{8, island.IslandY + 2, 8}
	s.World.Do(func(tx *world.Tx) {
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		_ = h.Inventory(tx, vanilla).SetItem(0, item.NewStack(item.Wheat{}, 2))
		tx.SetBlock(vanilla, h, nil)
		tx.SetBlock(wood, hopper.NewFlowHopper("wood_hopper", cube.FaceDown), nil)
		if _, ex := itemio.IO(tx, wood); ex == nil {
			t.Fatal("the wood hopper has no Extractor")
		}
	})
	tick(s, 20)
	s.World.Do(func(tx *world.Tx) {
		if got, _ := tx.Block(wood).(hopper.FlowHopper).Inv.Item(0); got.Count() != 2 {
			t.Errorf("the wood hopper holds %v, want the wheat of the vanilla hopper", got)
		}
	})
}

// TestHopperGeometryMirrorsX: the client's geometry x axis is mirrored, so
// east and west use each other's model; north and south are plain.
func TestHopperGeometryMirrorsX(t *testing.T) {
	for f, want := range map[cube.Face]string{
		cube.FaceEast: "west", cube.FaceWest: "east", cube.FaceNorth: "north", cube.FaceSouth: "south", cube.FaceDown: "down",
	} {
		if got := hopper.HopperGeometry(f); got != "geometry.skygear_flow_hopper_"+want {
			t.Errorf("%v: %s", f, got)
		}
	}
}

// TestChestOpensUnderFlowHopper: a chest opens only when the block above
// diffuses light; the game's hoppers must not count as solid.
func TestChestOpensUnderFlowHopper(t *testing.T) {
	for _, id := range hopper.HopperOrder {
		var b world.Block = hopper.NewFlowHopper(id, cube.FaceDown)
		d, ok := b.(block.LightDiffuser)
		if !ok || d.LightDiffusionLevel() > 2 {
			t.Errorf("%s blocks the chest below it", id)
		}
	}
}
