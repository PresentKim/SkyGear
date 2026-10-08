package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func TestDrawerStorageRules(t *testing.T) {
	d := Drawer{}
	cobble := item.NewStack(block.Cobblestone{}, 64)
	if n := d.insert(cobble, 64); n != 64 || d.Count != 64 {
		t.Fatalf("insert = %d, count %d", n, d.Count)
	}
	if d.insert(item.NewStack(block.Dirt{}, 1), 1) != 0 {
		t.Fatal("a drawer holding cobblestone accepted dirt")
	}
	if d.capacity() != DrawerBaseStacks*64 {
		t.Fatalf("capacity = %d", d.capacity())
	}
	d.Count = d.capacity() - 10
	if n := d.insert(cobble, 64); n != 10 || d.Count != d.capacity() {
		t.Fatalf("insert into almost full = %d", n)
	}
	d.Void = true
	if n := d.insert(cobble, 64); n != 64 || d.Count != d.capacity() {
		t.Fatalf("void drawer must take and destroy overflow, took %d count %d", n, d.Count)
	}
	d.Capacity = 1
	if d.capacity() != 2*DrawerBaseStacks*64 {
		t.Fatal("capacity upgrade must double the capacity")
	}

	d = Drawer{}
	d.insert(cobble, 3)
	if out := d.extract(64); out.Count() != 3 || !d.Item.Empty() {
		t.Fatalf("extract all = %d, unlocked drawer must forget its item", out.Count())
	}
	d.Locked = true
	d.insert(cobble, 3)
	d.extract(64)
	if d.Item.Empty() {
		t.Fatal("a locked drawer forgot its item when emptied")
	}
	if d.insert(item.NewStack(block.Dirt{}, 1), 1) != 0 {
		t.Fatal("a locked empty drawer accepted another item")
	}
}

func TestDrawerItemKeepsContents(t *testing.T) {
	d := Drawer{Count: 1234, Capacity: 2, Locked: true}
	d.Item = item.NewStack(block.Log{Wood: block.OakWood()}, 1)
	st := d.toItem()
	back := item.ReadNBT(item.WriteNBT(st, true), nil) // as when stored and dropped
	r := drawerFromItem(back, cube.East)
	if r.Count != 1234 || r.Capacity != 2 || !r.Locked || r.Void || !r.Item.Comparable(d.Item) || r.Facing != cube.East {
		t.Fatalf("restored drawer = %+v", r)
	}
	if len(st.Lore()) == 0 {
		t.Fatal("drawer item with contents must describe them in its lore")
	}
	if e := (Drawer{}).toItem(); len(e.Values()) != 0 {
		t.Fatal("an empty drawer item must stay stackable (no values)")
	}
}

func TestDrawerHopperAndDisplay(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	top, mid, bottom := cube.Pos{1, 66, 1}, cube.Pos{1, 65, 1}, cube.Pos{1, 64, 1}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(mid, Drawer{Facing: cube.North}, nil)
		tx.SetBlock(top, block.NewHopper(), nil)
		_, _ = tx.Block(top).(block.Hopper).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 10))
	})
	tick(s, 20*12)
	s.world.Do(func(tx *world.Tx) {
		d := tx.Block(mid).(Drawer)
		if d.Count != 10 {
			t.Fatalf("drawer count = %d after hopper input, want 10", d.Count)
		}
		disp, ok := s.displays[mid]
		if !ok {
			t.Fatal("drawer has no display")
		}
		e, ok := disp.item.Entity(tx)
		if !ok {
			t.Fatal("display holder missing")
		}
		if main, _ := e.(*ItemHolder).HeldItems(); main.Empty() {
			t.Fatal("display does not hold the stored item")
		}
		l, _ := disp.label.Entity(tx)
		if l.(*ItemHolder).NameTag() != "10" {
			t.Fatalf("label = %q", l.(*ItemHolder).NameTag())
		}
		tx.SetBlock(bottom, block.NewHopper(), nil)
	})
	tick(s, 20*12)
	s.world.Do(func(tx *world.Tx) {
		if d := tx.Block(mid).(Drawer); d.Count != 0 {
			t.Fatalf("hopper below did not empty the drawer (%d left)", d.Count)
		}
		// Removing the drawer by other means makes its display clean up.
		tx.SetBlock(mid, nil, nil)
	})
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if _, ok := e.(*ItemHolder); ok {
				t.Fatal("display entity outlived its drawer")
			}
		}
		tx.SetBlock(mid, Drawer{Facing: cube.North}, nil)
	})
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		n := 0
		for e := range tx.Entities() {
			if _, ok := e.(*ItemHolder); ok {
				n++
			}
		}
		if n != 2 {
			t.Fatalf("a drawer placed again at the same spot has %d display entities, want 2", n)
		}
	})
}
