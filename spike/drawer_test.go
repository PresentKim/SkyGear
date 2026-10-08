package spike

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func loadAround(w *world.World) {
	w.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 100)
	})
}

// hopper(top) -> drawer -> hopper(bottom): a custom block implementing
// block.HopperInsertable/HopperExtractable joins vanilla hopper lines, and
// the drawer only accepts the item type it holds.
func TestDrawerHopperChain(t *testing.T) {
	w := newWorld(t, "")
	defer w.Close()
	loadAround(w)
	top, mid, bottom := cube.Pos{0, 66, 0}, cube.Pos{0, 65, 0}, cube.Pos{0, 64, 0}
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(mid, Drawer{Cap: 1000}, nil)
		tx.SetBlock(top, block.NewHopper(), nil)
		tx.SetBlock(bottom, block.NewHopper(), nil)
		_, _ = tx.Block(top).(block.Hopper).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 10))
		_, _ = tx.Block(top).(block.Hopper).Inventory(tx, top).AddItem(item.NewStack(item.Apple{}, 3))
	})
	tick(w, 20*20)
	w.Do(func(tx *world.Tx) {
		d := tx.Block(mid).(Drawer)
		out := tx.Block(bottom).(block.Hopper).Inventory(tx, bottom)
		in := tx.Block(top).(block.Hopper).Inventory(tx, top)
		wheatOut := 0
		for _, s := range out.Items() {
			if _, ok := s.Item().(item.Wheat); ok {
				wheatOut += s.Count()
			}
		}
		if int(d.Count)+wheatOut != 10 {
			t.Errorf("wheat drawer=%d + bottom hopper=%d, want 10 in total", d.Count, wheatOut)
		}
		if !in.ContainsItem(item.NewStack(item.Apple{}, 3)) {
			t.Error("drawer locked to wheat must not accept apples")
		}
		t.Logf("after 20s: drawer=%d bottom hopper wheat=%d", d.Count, wheatOut)
	})
}

// A hopper directly under the display item's block cell absorbs the display
// entity like any dropped item. The drawer must place the display where no
// hopper can be, or re-create it and remove the absorbed copy.
func TestHopperAbsorbsDisplayItem(t *testing.T) {
	w := newWorld(t, "")
	defer w.Close()
	loadAround(w)
	var display *world.EntityHandle
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 0}, block.NewHopper(), nil)
		display = world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, 65.5, 0.5}}.New(entity.ItemType, entity.ItemBehaviourConfig{
			Item:              item.NewStack(item.Wheat{}, 1).WithValue("drawer_display", true),
			ExistenceDuration: math.MaxInt64,
			PickupDelay:       math.MaxInt64,
		})
		tx.AddEntity(display)
	})
	tick(w, 40)
	w.Do(func(tx *world.Tx) {
		_, alive := display.Entity(tx)
		t.Logf("display alive above hopper: %v", alive)
		if alive {
			t.Skip("hopper did not absorb the display; no workaround needed")
		}
	})
}
