package world_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
)

const loadRadius = 7 // (2*7+1)^2 = 225 columns

type countViewer struct {
	world.NopViewer
	n atomic.Int32
}

func (v *countViewer) ViewChunk(world.ChunkPos, world.Dimension, map[cube.Pos]world.Block, *chunk.Chunk) {
	v.n.Add(1)
}

func markerY(x, z int32) int { return int((x*7+z)%20+20) % 20 }

func loadAll(t *testing.T, dir string, workers int) map[world.ChunkPos]int {
	t.Helper()
	db, err := mcdb.Config{}.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	w := world.Config{Provider: db, ChunkLoadWorkers: workers, SaveInterval: -1}.New()
	defer w.Close()
	v := &countViewer{}
	n := (2*loadRadius + 1) * (2*loadRadius + 1)
	<-w.Do(func(tx *world.Tx) {
		l := world.NewLoader(loadRadius, w, v)
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, n)
	}).Done()
	// The loader views the chunks within its circle; wait until that settles.
	deadline := time.Now().Add(20 * time.Second)
	for last, stable := int32(-1), 0; stable < 10 && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if cur := v.n.Load(); cur == last && cur > 0 {
			stable++
		} else {
			last, stable = cur, 0
		}
	}
	got := map[world.ChunkPos]int{}
	<-w.Do(func(tx *world.Tx) {
		for x := int32(-loadRadius); x <= loadRadius; x++ {
			for z := int32(-loadRadius); z <= loadRadius; z++ {
				got[world.ChunkPos{x, z}] = -1
				for y := 0; y < 20; y++ {
					if _, ok := tx.Block(cube.Pos{int(x) * 16, y, int(z) * 16}).(block.Stone); ok {
						got[world.ChunkPos{x, z}] = y
					}
				}
			}
		}
	}).Done()
	return got
}

// TestConcurrentLoadSameColumns: loading with 4 workers through mcdb gives the
// same columns as with 1 worker.
func TestConcurrentLoadSameColumns(t *testing.T) {
	var p world.Provider = (*mcdb.DB)(nil)
	if c, ok := p.(world.ConcurrentProvider); !ok || !c.Concurrent() {
		t.Fatal("mcdb.DB is not a ConcurrentProvider")
	}
	dir := t.TempDir()
	db, err := mcdb.Config{}.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	stone := world.BlockRuntimeID(block.Stone{})
	for x := int32(-loadRadius); x <= loadRadius; x++ {
		for z := int32(-loadRadius); z <= loadRadius; z++ {
			c := chunk.New(world.DefaultBlockRegistry, world.Overworld.Range())
			c.SetBlock(0, int16(markerY(x, z)), 0, 0, stone)
			if err := db.StoreColumn(world.ChunkPos{x, z}, world.Overworld, &chunk.Column{Chunk: c}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	one := loadAll(t, dir, 1)
	four := loadAll(t, dir, 4)
	if len(one) != 225 || len(four) != 225 {
		t.Fatalf("columns: %d, %d", len(one), len(four))
	}
	for pos, y := range one {
		if y != markerY(pos[0], pos[1]) {
			t.Fatalf("1 worker: %v marker %d, want %d", pos, y, markerY(pos[0], pos[1]))
		}
		if four[pos] != y {
			t.Fatalf("%v: 4 workers %d, 1 worker %d", pos, four[pos], y)
		}
	}
}
