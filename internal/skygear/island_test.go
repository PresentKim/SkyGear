package skygear

import (
	"slices"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/internal/skygear/textfmt"
)

func TestChunkRules(t *testing.T) {
	s := testSession(t)
	if !s.canBuild(cube.Pos{0, 64, 0}) || !s.canBuild(cube.Pos{15, 64, 15}) {
		t.Fatal("starting island must cover the first chunk")
	}
	if s.canBuild(cube.Pos{16, 64, 0}) || s.canBuild(cube.Pos{-1, 64, 0}) {
		t.Fatal("blocks outside owned chunks must be protected")
	}
	cands := s.purchasableChunks()
	if len(cands) != 4 || !slices.Contains(cands, ChunkPos{1, 0}) || slices.Contains(cands, ChunkPos{1, 1}) {
		t.Fatalf("purchasable = %v, want the 4 edge neighbours", cands)
	}
	// The island grows up to islandRadius chunks from the centre; bought
	// chunks are kept loaded.
	var kept []ChunkPos
	s.keep = func(_ *world.Tx, c ChunkPos) { kept = append(kept, c) }
	s.state.Chunks = append(s.state.Chunks, ChunkPos{islandRadius, 0})
	for _, c := range s.purchasableChunks() {
		if c[0] > islandRadius {
			t.Fatalf("chunk %v outside the island radius is purchasable", c)
		}
	}
	s.keepLoaded(nil, ChunkPos{5, 0})
	if len(kept) != 1 {
		t.Fatal("bought chunk not kept loaded")
	}
	if chunkCost(4).Count != 64 || chunkCost(5).Count != 128 {
		t.Fatalf("chunk costs %v %v", chunkCost(4), chunkCost(5))
	}
}

func TestLostAndFound(t *testing.T) {
	s := testSession(t)
	s.world.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, s.world, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 50)
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0, -70, 0}}, item.NewStack(item.Diamond{}, 3)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{5, -70, 0}}, item.NewStack(item.Apple{}, 1)))
	})
	for range 20 {
		s.world.AdvanceTick()
	}
	s.addLost(item.NewStack(item.Diamond{}, 2))
	if len(s.state.Lost) != 2 {
		t.Fatalf("lost entries = %d, want 2 (diamonds merged, apple)", len(s.state.Lost))
	}
	if top := s.state.Lost[0]; top.Count != 5 {
		t.Fatalf("most recent entry count = %d, want 5 diamonds", top.Count)
	}
	s.clock.Advance(19 * time.Minute)
	s.addLost(item.NewStack(item.Apple{}, 1))
	s.clock.Advance(2 * time.Minute)
	s.purgeLost()
	if len(s.state.Lost) != 1 || s.state.Lost[0].Count != 2 {
		t.Fatalf("after a day only the recently dropped apples remain, got %+v", s.state.Lost)
	}
}

func TestNearestBlockColumn(t *testing.T) {
	s := testSession(t)
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{3, 63, 0}, block.Stone{}, nil)
		tx.SetBlock(cube.Pos{-10, 63, 0}, block.Stone{}, nil)
		x, z, ok := s.nearestBlockColumn(tx, 7, 1)
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
