package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/storage"
)

func TestDrawerHopperAndDisplay(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	top, mid, bottom := cube.Pos{1, 66, 1}, cube.Pos{1, 65, 1}, cube.Pos{1, 64, 1}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(mid, storage.Drawer{Facing: cube.North}, nil)
		tx.SetBlock(top, block.NewHopper(), nil)
		_, _ = tx.Block(top).(block.Hopper).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 10))
	})
	tick(s, 20*12)
	s.World.Do(func(tx *world.Tx) {
		d := tx.Block(mid).(storage.Drawer)
		if d.Count != 10 {
			t.Fatalf("drawer count = %d after hopper input, want 10", d.Count)
		}
		disp, ok := storage.LocalOf(s.Session).Displays[mid]
		if !ok {
			t.Fatal("drawer has no display")
		}
		e, ok := disp.Item.Entity(tx)
		if !ok {
			t.Fatal("display holder missing")
		}
		if main, _ := e.(*storage.ItemHolder).HeldItems(); main.Empty() {
			t.Fatal("display does not hold the stored item")
		}
		l, ok := shapeAs(s, storage.DrawerLabelKey(mid)).(*debug.Text)
		if !ok || l.Text != "10" {
			t.Fatalf("count text = %+v", l)
		}
		tx.SetBlock(bottom, block.NewHopper(), nil)
	})
	tick(s, 20*12)
	s.World.Do(func(tx *world.Tx) {
		if d := tx.Block(mid).(storage.Drawer); d.Count != 0 {
			t.Fatalf("hopper below did not empty the drawer (%d left)", d.Count)
		}
		// Removing the drawer by other means makes its display clean up.
		tx.SetBlock(mid, nil, nil)
	})
	// Display and timed upkeep run every 20 ticks (phased), so cover the window.
	tick(s, 41)
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if _, ok := e.(*storage.ItemHolder); ok {
				t.Fatal("display entity outlived its drawer")
			}
		}
		if _, ok := s.Shape(storage.DrawerLabelKey(mid)); ok {
			t.Fatal("count text outlived its drawer")
		}
		tx.SetBlock(mid, storage.Drawer{Facing: cube.North}, nil)
	})
	tick(s, 41)
	s.World.Do(func(tx *world.Tx) {
		n := 0
		for e := range tx.Entities() {
			if _, ok := e.(*storage.ItemHolder); ok {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("a drawer placed again at the same spot has %d display entities, want 1", n)
		}
	})
}

// TestDrawerStatesMatchRuntimeIDs: the client numbers a custom block's states
// in the order of its state values, so drawer runtime IDs must follow
// drawerFacings, with south (the default, drawn in the inventory) first.
func TestDrawerStatesMatchRuntimeIDs(t *testing.T) {
	if storage.DrawerFacings[0] != cube.South {
		t.Fatalf("default drawer facing %v, want south", storage.DrawerFacings[0])
	}
	values := storage.Drawer{}.States()[storage.FacingState]
	var last uint32
	for i, f := range storage.DrawerFacings {
		if values[i] != storage.FacingNames[f] {
			t.Errorf("state %d is %v, want %v", i, values[i], storage.FacingNames[f])
		}
		rid := world.BlockRuntimeID(storage.Drawer{Facing: f})
		if i > 0 && rid != last+1 {
			t.Errorf("%v has runtime ID %d after %d", f, rid, last)
		}
		last = rid
	}
}
