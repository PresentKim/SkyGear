package skygear

import (
	"slices"
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

func TestChunkRules(t *testing.T) {
	s := testSession(t)
	if !island.CanBuild(s.Session, cube.Pos{0, 64, 0}) || !island.CanBuild(s.Session, cube.Pos{15, 64, 15}) {
		t.Fatal("starting island must cover the first chunk")
	}
	if island.CanBuild(s.Session, cube.Pos{16, 64, 0}) || island.CanBuild(s.Session, cube.Pos{-1, 64, 0}) {
		t.Fatal("blocks outside owned chunks must be protected")
	}
	cands := island.Purchasable(s.Session)
	if len(cands) != 4 || !slices.Contains(cands, island.ChunkPos{1, 0}) || slices.Contains(cands, island.ChunkPos{1, 1}) {
		t.Fatalf("purchasable = %v, want the 4 edge neighbours", cands)
	}
	// The island grows up to islandRadius chunks from the centre; bought
	// chunks are kept loaded.
	var kept []island.ChunkPos
	island.SetKeep(s.Session, func(_ *world.Tx, c island.ChunkPos) { kept = append(kept, c) })
	island.StateOf(s.Session).Chunks = append(island.StateOf(s.Session).Chunks, island.ChunkPos{island.Radius, 0})
	for _, c := range island.Purchasable(s.Session) {
		if c[0] > island.Radius {
			t.Fatalf("chunk %v outside the island radius is purchasable", c)
		}
	}
	island.KeepLoaded(s.Session, nil, island.ChunkPos{5, 0})
	if len(kept) != 1 {
		t.Fatal("bought chunk not kept loaded")
	}
	if island.ChunkCost(4).Count != 64 || island.ChunkCost(5).Count != 128 {
		t.Fatalf("chunk costs %v %v", island.ChunkCost(4), island.ChunkCost(5))
	}
}

func TestLostAndFound(t *testing.T) {
	s := testSession(t)
	s.World.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.World, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0, -70, 0}}, item.NewStack(item.Diamond{}, 3)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{5, -70, 0}}, item.NewStack(item.Apple{}, 1)))
	})
	for range 20 {
		s.World.AdvanceTick()
	}
	island.AddLost(s.Session, item.NewStack(item.Diamond{}, 2))
	if len(island.StateOf(s.Session).Lost) != 2 {
		t.Fatalf("lost entries = %d, want 2 (diamonds merged, apple)", len(island.StateOf(s.Session).Lost))
	}
	if top := island.StateOf(s.Session).Lost[0]; top.Count != 5 {
		t.Fatalf("most recent entry count = %d, want 5 diamonds", top.Count)
	}
	s.Clock.Advance(19 * time.Minute)
	island.AddLost(s.Session, item.NewStack(item.Apple{}, 1))
	s.Clock.Advance(2 * time.Minute)
	island.PurgeLost(s.Session)
	if len(island.StateOf(s.Session).Lost) != 1 || island.StateOf(s.Session).Lost[0].Count != 2 {
		t.Fatalf("after a day only the recently dropped apples remain, got %+v", island.StateOf(s.Session).Lost)
	}
}

func TestNearestBlockColumn(t *testing.T) {
	s := testSession(t)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{3, 63, 0}, block.Stone{}, nil)
		tx.SetBlock(cube.Pos{-10, 63, 0}, block.Stone{}, nil)
		x, z, ok := island.NearestBlockColumn(s.Session, tx, 7, 1)
		if !ok || x != 3 || z != 0 {
			t.Fatalf("nearest = %d %d %v, want 3 0", x, z, ok)
		}
	})
}

func TestFormatInt(t *testing.T) {
	for v, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -12400: "-12,400"} {
		if got := textfmt.Int(v); got != want {
			t.Errorf("formatInt(%d) = %q, want %q", v, got, want)
		}
	}
}
