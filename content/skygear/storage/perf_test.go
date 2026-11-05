package storage

import (
	"log/slog"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/platform/instance"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func perfSession(t testing.TB) *core.Session {
	t.Helper()
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(append(entity.DefaultRegistry.Types(), HolderType))}.New()
	t.Cleanup(func() { _ = w.Close() })
	s := core.New(core.Config{World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute), SaveID: 1, SaveName: "t", HostID: "h", Log: slog.Default()})
	core.Register(s)
	t.Cleanup(func() { core.Unregister(s); s.Close() })
	return s
}

func TestControllerSeesNewDrawer(t *testing.T) {
	s := perfSession(t)
	tick := int64(1)
	old := networkTick
	networkTick = func() int64 { return tick }
	t.Cleanup(func() { networkTick = old })
	ctl := cube.Pos{0, 64, 0}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(ctl, machine.New(machine.KindDrawerController, "", ""), nil)
		tx.SetBlock(ctl.Side(cube.FaceEast), Drawer{Facing: cube.South}, nil)
		if n := len(Network(tx, ctl)); n != 1 {
			t.Fatalf("%d drawers, want 1", n)
		}
		tx.SetBlock(ctl.Side(cube.FaceWest), Drawer{Facing: cube.South}, nil)
		tick++
		if n := len(Network(tx, ctl)); n != 2 {
			t.Fatalf("next tick: %d drawers, want 2", n)
		}
		// Cached within the tick, but values are always fresh.
		ControllerInsert(tx, ctl, item.NewStack(block.Cobblestone{}, 10))
		if got := ControllerPeek(tx, ctl, nil); got.Count() != 10 {
			t.Fatalf("peek count %d, want 10", got.Count())
		}
		tx.SetBlock(ctl.Side(cube.FaceEast), block.Air{}, nil)
		if n := len(Network(tx, ctl)); n != 1 {
			t.Fatalf("broken drawer still found: %d", n)
		}
	})
}

func TestLabelBatched(t *testing.T) {
	s := perfSession(t)
	pos := cube.Pos{0, 64, 0}
	s.World.Do(func(tx *world.Tx) {
		d := Drawer{Facing: cube.South}
		tx.SetBlock(pos, d, nil)
		refreshDisplay(s, tx, pos, d)
		d.Item, d.Count = item.NewStack(block.Cobblestone{}, 1), 0
		sent := 0
		var last debug.Shape
		count := func() {
			if sh, ok := s.Shape(DrawerLabelKey(pos)); ok && sh != last {
				last = sh
				sent++
			}
		}
		tick := int64(100)
		d.Tick(tick, pos, tx)
		for i := 1; i <= 10; i++ {
			tick++
			d.Count = int64(i)
			SetDrawer(s, tx, pos, d)
			d.Tick(tick, pos, tx)
			count()
		}
		if sent > 1 {
			t.Fatalf("%d label updates in 10 ticks, want <= 1", sent)
		}
		for i := 0; i < 20; i++ {
			tick++
			d.Tick(tick, pos, tx)
			count()
		}
		got, _ := s.Shape(DrawerLabelKey(pos))
		if txt := got.(*debug.Text).Text; txt != d.label() {
			t.Fatalf("final label %q, want %q", txt, d.label())
		}
		if sent > 2 {
			t.Fatalf("%d label updates, want <= 2", sent)
		}
	})
}
