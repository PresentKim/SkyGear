package world_test

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
)

// skygear patch: baseline benchmarks for the tick loop and column saving. They only use package-level APIs
// so that they keep compiling after later tick and save patches.

const (
	skygearBenchChunksX  = 30
	skygearBenchChunksZ  = 20 // 600 loaded chunks in total.
	skygearBenchLoaders  = 450
	skygearBenchTickers  = 5000
	skygearBenchSaveCols = 100
)

// skygearTicker is a block with block entity data whose Tick does nothing.
type skygearTicker struct{}

func (skygearTicker) EncodeBlock() (string, map[string]any) { return "skygear:bench_ticker", nil }
func (skygearTicker) Hash() (uint64, uint64)                { return 1 << 47, 0 }
func (skygearTicker) Model() world.BlockModel               { return skygearModel{} }
func (skygearTicker) DecodeNBT(map[string]any) any          { return skygearTicker{} }
func (skygearTicker) EncodeNBT() map[string]any             { return map[string]any{"id": "SkygearBench"} }
func (skygearTicker) Tick(int64, cube.Pos, *world.Tx)       {}

type skygearModel struct{}

func (skygearModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 1, 1)}
}
func (skygearModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return true }

func skygearRegistry() world.BlockRegistry {
	r := world.NewBlockRegistry()
	r.RegisterBlockState(world.BlockState{Name: "skygear:bench_ticker", Properties: map[string]any{}})
	r.RegisterBlock(skygearTicker{})
	return r
}

// skygearLoader is a Viewer that ignores everything.
type skygearLoader struct{ world.NopViewer }

func skygearRun(w *world.World, f func(*world.Tx)) {
	w.Do(f).Wait(context.Background())
}

// skygearBenchWorld returns a synchronous world with 600 loaded chunks, loaders radius-0 loaders and tickers
// ticking block entities spread over the chunks.
func skygearBenchWorld(loaders, tickers int) (*world.World, []*world.Loader) {
	return skygearBenchWorldSync(true, loaders, tickers)
}

func skygearBenchWorldSync(synchronous bool, loaders, tickers int) (*world.World, []*world.Loader) {
	w := world.Config{Synchronous: synchronous, Blocks: skygearRegistry()}.New()
	var ls []*world.Loader
	skygearRun(w, func(tx *world.Tx) {
		for i := range skygearBenchChunksX * skygearBenchChunksZ {
			cx, cz := i%skygearBenchChunksX, i/skygearBenchChunksX
			if i < loaders {
				l := world.NewLoader(0, w, skygearLoader{})
				l.Move(tx, mgl64.Vec3{float64(cx*16 + 8), 64, float64(cz*16 + 8)})
				l.Load(tx, 1)
				ls = append(ls, l)
			} else {
				// Chunks without a loader are still loaded by touching them.
				tx.Block(cube.Pos{cx * 16, 64, cz * 16})
			}
		}
		for i := range tickers {
			chunk := i % (skygearBenchChunksX * skygearBenchChunksZ)
			cx, cz := chunk%skygearBenchChunksX, chunk/skygearBenchChunksX
			n := i / (skygearBenchChunksX * skygearBenchChunksZ)
			tx.SetBlock(cube.Pos{cx*16 + n%16, 64 + n/256, cz*16 + (n/16)%16}, skygearTicker{}, nil)
		}
	})
	return w, ls
}

func BenchmarkSkygearTickLoaders(b *testing.B) {
	w, ls := skygearBenchWorld(skygearBenchLoaders, 0)
	defer w.Close()
	if len(ls) != skygearBenchLoaders {
		b.Fatalf("loaders = %d", len(ls))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w.AdvanceTick()
	}
}

func BenchmarkSkygearBlockEntityTick(b *testing.B) {
	w, _ := skygearBenchWorld(skygearBenchLoaders, skygearBenchTickers)
	defer w.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w.AdvanceTick()
	}
}

// BenchmarkSkygearTickLoadersRange ticks a non-synchronous-position world: 450 radius-0 loaders at distinct
// chunks and one loader with a 12 chunk simulation distance over 600 loaded chunks.
func BenchmarkSkygearTickLoadersRange(b *testing.B) {
	w, _ := skygearBenchWorldSync(false, skygearBenchLoaders, skygearBenchTickers)
	defer w.Close()
	w.SetTickRange(12)
	var big *world.Loader
	skygearRun(w, func(tx *world.Tx) {
		big = world.NewLoader(12, w, skygearLoader{})
		big.Move(tx, mgl64.Vec3{15 * 16, 64, 10 * 16})
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w.AdvanceTick()
	}
}

// skygearColumn builds a column with 24 sub chunks of stone-less (air) data plus one block entity.
func skygearColumn(br world.BlockRegistry, seed int) *chunk.Column {
	c := chunk.New(br, cube.Range{-64, 319})
	rid := br.BlockRuntimeID(skygearTicker{})
	for y := -64; y < 320; y += 4 {
		c.SetBlock(uint8(seed%16), int16(y), uint8((seed/16)%16), 0, rid)
	}
	return &chunk.Column{
		Chunk:         c,
		BlockEntities: []chunk.BlockEntity{{Pos: cube.Pos{seed % 16, 64, 0}, Data: map[string]any{"id": "SkygearBench"}}},
	}
}

func BenchmarkSkygearSaveColumn(b *testing.B) {
	br := skygearRegistry()
	br.Finalize()
	db, err := mcdb.Config{Blocks: br}.Open(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	cols := make([]*chunk.Column, skygearBenchSaveCols)
	for i := range cols {
		cols[i] = skygearColumn(br, i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for i, col := range cols {
			if err := db.StoreColumn(world.ChunkPos{int32(i), 0}, world.Overworld, col); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSkygearDiskEncode(b *testing.B) {
	br := skygearRegistry()
	br.Finalize()
	col := skygearColumn(br, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if d := chunk.Encode(col.Chunk, chunk.DiskEncoding); len(d.SubChunks) != 24 {
			b.Fatalf("sub chunks = %d", len(d.SubChunks))
		}
	}
}
