package bridge

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func testWorld(t *testing.T) *world.World {
	t.Helper()
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	w.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
	})
	return w
}

// standing at the south edge of the block at (0,64,0), looking south and
// down past it
func edgeView() (eye, dir mgl64.Vec3, body cube.BBox) {
	feet := mgl64.Vec3{0.5, 65, 0.9}
	return feet.Add(mgl64.Vec3{0, 1.62, 0}), cube.Rotation{0, 60}.Vec3(),
		cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3).Translate(feet)
}

// TestFindEdge: off an edge, the block goes in front of the block stood on.
func TestFindEdge(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 0}, block.Stone{}, nil)
		eye, dir, body := edgeView()
		got, ok := Find(tx, eye, dir, body, block.Planks{})
		if !ok {
			t.Fatal("no target off the edge")
		}
		if got.Support != (cube.Pos{0, 64, 0}) || got.Face != cube.FaceSouth {
			t.Fatalf("target %v %v, want (0,64,0) south", got.Support, got.Face)
		}
		if got.Mark() != MarkSide {
			t.Fatalf("mark %q, want %q", got.Mark(), MarkSide)
		}
		if c := got.ClickPos; c[2] != 1 || c[0] < 0 || c[0] > 1 || c[1] < 0 || c[1] > 1 {
			t.Fatalf("click %v not on the south face", c)
		}
	})
}

// TestFindNothing: with no block to build off, there is no target.
func TestFindNothing(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		eye, dir, body := edgeView()
		if got, ok := Find(tx, eye, dir, body, block.Planks{}); ok {
			t.Fatalf("target %v in the empty air", got)
		}
	})
}

// TestFindOffActivatable: the player sneaks, so a crafting table is built
// off (the sneaking use places the block and does not open it).
func TestFindOffActivatable(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 0}, block.CraftingTable{}, nil)
		eye, dir, body := edgeView()
		if _, ok := Find(tx, eye, dir, body, block.Planks{}); !ok {
			t.Fatal("no target off a crafting table while sneaking")
		}
	})
}

// TestFindUp: looking up past the top of a pillar, the block goes on top.
func TestFindUp(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 66, 2}, block.Stone{}, nil)
		feet := mgl64.Vec3{0.5, 65, 0.5}
		eye := feet.Add(mgl64.Vec3{0, 1.62, 0})
		// aim at the cell above the pillar, (0,67,2)
		dir := mgl64.Vec3{0.5, 67.5, 2.5}.Sub(eye).Normalize()
		got, ok := Find(tx, eye, dir, cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3).Translate(feet), block.Planks{})
		if !ok {
			t.Fatal("no target over the pillar")
		}
		if got.Support != (cube.Pos{0, 66, 2}) || got.Face != cube.FaceUp || got.Mark() != MarkUp {
			t.Fatalf("target %v %v, want (0,66,2) up", got.Support, got.Face)
		}
	})
}

// TestNewPress: uses closer than pressGap are one press, a held button's
// repeat never starts one, and the first use always does.
func TestNewPress(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		last    time.Time
		trigger uint32
		want    bool
	}{
		{"first use", time.Time{}, protocol.TriggerTypePlayerInput, true},
		{"first use, unknown trigger", time.Time{}, protocol.TriggerTypeUnknown, true},
		{"click sent twice", now.Add(-20 * time.Millisecond), protocol.TriggerTypePlayerInput, false},
		{"repeat of a held button", now.Add(-200 * time.Millisecond), protocol.TriggerTypeUnknown, false},
		{"held a long time", now.Add(-time.Second), protocol.TriggerTypeSimulationTick, false},
		{"held, first repeat", time.Time{}, protocol.TriggerTypeSimulationTick, false},
		{"next click", now.Add(-pressGap), protocol.TriggerTypePlayerInput, true},
	}
	for _, c := range cases {
		if got := newPress(c.last, now, c.trigger); got != c.want {
			t.Errorf("%s: newPress = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestHookTurnsPreviewClickToAir: a click on the preview's cell (a block
// only the client sees) becomes a use on the air, and the block the client
// predicted next to it is noted; other clicks are left alone.
func TestHookTurnsPreviewClickToAir(t *testing.T) {
	c := &conn{}
	c.setPreview(cube.Pos{1, 64, 2}, true)
	d := &protocol.UseItemTransactionData{ActionType: protocol.UseItemActionClickBlock, BlockPosition: protocol.BlockPos{1, 64, 2}, BlockFace: int32(cube.FaceUp), TriggerType: protocol.TriggerTypePlayerInput}
	c.see(d)
	if d.ActionType != protocol.UseItemActionClickAir {
		t.Fatalf("action %d, want a use on the air", d.ActionType)
	}
	if g, ok := c.takeGhost(); !ok || g != (cube.Pos{1, 65, 2}) {
		t.Fatalf("ghost %v %v, want (1,65,2)", g, ok)
	}
	if c.lastTrigger() != protocol.TriggerTypePlayerInput {
		t.Fatalf("trigger %d not kept", c.lastTrigger())
	}
	other := &protocol.UseItemTransactionData{ActionType: protocol.UseItemActionClickBlock, BlockPosition: protocol.BlockPos{1, 63, 2}, TriggerType: protocol.TriggerTypeSimulationTick}
	c.see(other)
	if other.ActionType != protocol.UseItemActionClickBlock {
		t.Fatal("a click on a real block was changed")
	}
	if _, ok := c.takeGhost(); ok {
		t.Fatal("ghost noted for a real block")
	}
	c.setPreview(cube.Pos{}, false)
	d.ActionType = protocol.UseItemActionClickBlock
	c.see(d)
	if d.ActionType != protocol.UseItemActionClickBlock {
		t.Fatal("click changed with no preview")
	}
}

// TestTargetCell: the cell is the support's neighbour on the face, whole
// block coordinates even off a slab.
func TestTargetCell(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 0}, block.Slab{Block: block.Stone{}, Double: true}, nil)
		eye, dir, body := edgeView()
		got, ok := Find(tx, eye, dir, body, block.Planks{})
		if !ok {
			t.Fatal("no target")
		}
		if got.Cell() != (cube.Pos{0, 64, 1}) {
			t.Fatalf("cell %v, want (0,64,1)", got.Cell())
		}
	})
}
