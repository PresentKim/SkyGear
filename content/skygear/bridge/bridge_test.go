package bridge

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
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
		got, ok := Find(tx, eye, dir, body, block.Planks{}, false)
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
		if got, ok := Find(tx, eye, dir, body, block.Planks{}, false); ok {
			t.Fatalf("target %v in the empty air", got)
		}
	})
}

// TestFindSkipsActivatable: a crafting table is not built off unless sneaking, as
// using it would open it.
func TestFindSkipsActivatable(t *testing.T) {
	w := testWorld(t)
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 0}, block.CraftingTable{}, nil)
		eye, dir, body := edgeView()
		if got, ok := Find(tx, eye, dir, body, block.Planks{}, false); ok {
			t.Fatalf("target %v off a crafting table", got)
		}
		if _, ok := Find(tx, eye, dir, body, block.Planks{}, true); !ok {
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
		got, ok := Find(tx, eye, dir, cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3).Translate(feet), block.Planks{}, false)
		if !ok {
			t.Fatal("no target over the pillar")
		}
		if got.Support != (cube.Pos{0, 66, 2}) || got.Face != cube.FaceUp || got.Mark() != MarkUp {
			t.Fatalf("target %v %v, want (0,66,2) up", got.Support, got.Face)
		}
	})
}
