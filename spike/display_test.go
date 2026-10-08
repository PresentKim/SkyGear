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

// A drawer can show its stored item as a floating item entity with zero
// gravity, no pickup and no expiry. This checks the entity stays in place and
// that a dropped stack of the same item next to it does not merge into it:
// without the unique "drawer_display" value both entities merge and vanish.
func TestDisplayItemEntity(t *testing.T) {
	w := newWorld(t, "")
	defer w.Close()
	displayPos := mgl64.Vec3{0.5, 64.5, -0.1}
	var display *world.EntityHandle
	w.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 100)
		tx.SetBlock(cube.Pos{0, 64, 0}, block.Stone{}, nil)
		tx.SetBlock(cube.Pos{0, 63, -1}, block.Stone{}, nil)
		display = world.EntitySpawnOpts{Position: displayPos}.New(entity.ItemType, entity.ItemBehaviourConfig{
			Item:              item.NewStack(item.Wheat{}, 1).WithValue("drawer_display", true),
			Gravity:           0,
			ExistenceDuration: math.MaxInt64,
			PickupDelay:       math.MaxInt64,
		})
		tx.AddEntity(display)
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, 64.2, -0.5}}, item.NewStack(item.Wheat{}, 5)))
	})
	tick(w, 20*60*6) // six minutes: past the vanilla 5 minute item lifetime
	w.Do(func(tx *world.Tx) {
		e, ok := display.Entity(tx)
		if !ok {
			t.Fatal("display item entity disappeared")
		}
		if d := e.Position().Sub(displayPos).Len(); d > 0.01 {
			t.Errorf("display moved by %.3f blocks", d)
		}
		got := e.(*entity.Ent).Behaviour().(*entity.ItemBehaviour).Item().Count()
		t.Logf("display stack count after a same-item drop nearby: %d", got)
		if got != 1 {
			t.Errorf("dropped wheat merged into the display (count %d)", got)
		}
	})
}
