package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/bridge"
	"github.com/presentkim/skygear/content/skygear/island"
)

// bridgeEdge stands p at the south edge of a stone block, looking south and
// down past it, holding planks; it returns the cell in front of the stone.
func bridgeEdge(p *player.Player, tx *world.Tx) cube.Pos {
	base := cube.Pos{0, island.IslandY, 0}
	tx.SetBlock(base, block.Stone{}, nil)
	p.Teleport(mgl64.Vec3{0.5, float64(island.IslandY + 1), 0.9})
	r := p.Rotation()
	p.Move(mgl64.Vec3{}, -r.Yaw(), 60-r.Pitch())
	p.SetHeldItems(item.NewStack(block.Planks{}, 10), item.Stack{})
	return base.Side(cube.FaceSouth)
}

func isPlanks(tx *world.Tx, pos cube.Pos) bool {
	_, ok := tx.Block(pos).(block.Planks)
	return ok
}

// TestBridgeNeedsSneak: without sneaking, a block used on the air does
// nothing and no preview is shown; sneaking, it shows and places in the
// cell in front of the edge.
func TestBridgeNeedsSneak(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		cell := bridgeEdge(p, tx)
		held, _ := p.HeldItems()
		bridge.Update(s.Session, p, tx, p.Position(), p.Rotation(), held, false, true)
		if _, ok := bridge.PreviewCell(s.Session, p); ok {
			t.Fatal("preview shown without sneaking")
		}
		if bridge.Mark(s.Session, p) != "" {
			t.Fatal("crosshair mark without sneaking")
		}
		if bridge.Place(s.Session, p) {
			t.Fatal("the assist took a use without sneaking")
		}
		if isPlanks(tx, cell) {
			t.Fatal("placed without sneaking")
		}

		p.StartSneaking()
		bridge.Update(s.Session, p, tx, p.Position(), p.Rotation(), held, true, true)
		got, ok := bridge.PreviewCell(s.Session, p)
		if !ok || got != cell {
			t.Fatalf("preview %v %v, want %v", got, ok, cell)
		}
		if bridge.Mark(s.Session, p) != bridge.MarkSide {
			t.Fatalf("mark %q, want side", bridge.Mark(s.Session, p))
		}
		if !bridge.Place(s.Session, p) || !isPlanks(tx, cell) {
			t.Fatal("no block placed while sneaking")
		}

		// letting go of sneak hides the preview
		bridge.Update(s.Session, p, tx, p.Position(), p.Rotation(), held, false, true)
		if _, ok := bridge.PreviewCell(s.Session, p); ok {
			t.Fatal("preview kept after sneaking ended")
		}
	})
}

// TestBridgeOnePerPress: the uses of one press (a held button's repeats, a
// click sent twice) place one block, also when the repeat lands on the
// block just placed.
func TestBridgeOnePerPress(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		cell := bridgeEdge(p, tx)
		p.StartSneaking()
		for range 5 {
			if !bridge.Place(s.Session, p) {
				t.Fatal("the assist let a use through")
			}
		}
		if !isPlanks(tx, cell) {
			t.Fatal("no block placed")
		}
		next := cell.Side(cube.FaceSouth)
		if isPlanks(tx, next) || isPlanks(tx, cell.Side(cube.FaceDown)) {
			t.Fatal("one press placed more than one block")
		}
		held, _ := p.HeldItems()
		if held.Count() != 9 {
			t.Fatalf("%d planks left, want 9", held.Count())
		}
		// the repeat on the block just placed is part of the press
		if !bridge.Holding(s.Session, p) {
			t.Fatal("a repeat on the placed block was let through")
		}
	})
}
