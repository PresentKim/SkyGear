package conduit

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/platform/instance"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func refreshSession(t *testing.T) *core.Session {
	t.Helper()
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(append(entity.DefaultRegistry.Types(), VisualType, PlateType))}.New()
	t.Cleanup(func() { _ = w.Close() })
	s := core.New(core.Config{World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute), SaveID: 1, SaveName: "t", HostID: "h"})
	t.Cleanup(s.Close)
	core.Register(s)
	t.Cleanup(func() { core.Unregister(s) })
	w.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
	return s
}

// TestBundleRepairsOnNeighbourUpdate: a block placed beside a bundle (with
// block updates) is picked up by the next block update, not by ticking the bundle.
func TestBundleRepairsOnNeighbourUpdate(t *testing.T) {
	s := refreshSession(t)
	pos := cube.Pos{3, 64, 3}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, Bundle{Item: 1}, nil) })
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos.Side(cube.FaceEast), block.NewChest(), nil)
	})
	s.World.AdvanceTick() // block updates run on the tick
	s.World.Do(func(tx *world.Tx) {
		b := tx.Block(pos).(Bundle)
		if b.IConn == 0 {
			t.Fatalf("IConn %b, want the chest side", b.IConn)
		}
		if _, ok := LocalOf(s).Plates[PlateKey{pos, FaceIndex(cube.FaceEast)}]; !ok {
			t.Fatal("no plate on the chest side")
		}
	})
}

// TestOrphanVisualClosed: a visual whose bundle is gone closes within 20 ticks.
func TestOrphanVisualClosed(t *testing.T) {
	s := refreshSession(t)
	pos := cube.Pos{3, 64, 3}
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, Bundle{Item: 1}, nil) })
	for range 8 {
		s.World.AdvanceTick()
	}
	h, ok := LocalOf(s).Visuals[pos]
	if !ok {
		t.Fatal("no visual")
	}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, block.Air{}, &world.SetOpts{DisableBlockUpdates: true})
	})
	for range 20 {
		s.World.AdvanceTick()
	}
	s.World.Do(func(tx *world.Tx) {
		if _, alive := h.Entity(tx); alive {
			t.Fatal("orphan visual still open")
		}
	})
}

// TestBrokenLoneConduitLeavesGrids: a conduit with no bundle beside it is
// dropped from the networks at once when it is broken.
func TestBrokenLoneConduitLeavesGrids(t *testing.T) {
	s := refreshSession(t)
	pos := cube.Pos{3, 64, 3}
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Bundle{Energy: 1}, nil)
		if st := network.Energy(s).Step(tx, 4); st.Networks != 1 {
			t.Fatalf("setup: %d networks", st.Networks)
		}
		tx.SetBlock(pos, block.Air{}, &world.SetOpts{DisableBlockUpdates: true})
		if st := network.Energy(s).Step(tx, 4); st.Networks != 0 {
			t.Fatalf("broken conduit still a network (%d)", st.Networks)
		}
	})
}
