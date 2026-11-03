package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
)

func subCacheWorld(t *testing.T) *world.World {
	registerTestRecipes()
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func loadedColumn(tx *world.Tx) *world.Column {
	l := world.NewLoader(1, tx.World(), world.NopViewer{})
	l.Move(tx, mgl64.Vec3{8, 64, 8})
	l.Load(tx, 10)
	c, _ := l.Chunk(world.ChunkPos{})
	return c
}

func TestSubChunkCacheInvalidatedOnSetBlock(t *testing.T) {
	w := subCacheWorld(t)
	w.Do(func(tx *world.Tx) {
		col := loadedColumn(tx)
		ind := int16(col.SubIndex(64))
		tx.SetBlock(cube.Pos{1, 64, 1}, block.Stone{}, nil)
		a := subChunkNetData(col, ind, true)
		if b := subChunkNetData(col, ind, true); b != a {
			t.Fatal("unchanged column did not hit the cache")
		}
		tx.SetBlock(cube.Pos{2, 64, 2}, block.Dirt{}, nil)
		c := subChunkNetData(col, ind, true)
		if c == a || bytes.Equal(c.payload, a.payload) {
			t.Fatal("block change served stale bytes")
		}
		want := chunk.EncodeSubChunk(col.Chunk, chunk.NetworkEncoding, int(ind))
		if !bytes.Equal(c.payload, want) {
			t.Fatal("cached payload differs from a fresh encode")
		}
		// A biome change through the column keeps the entry consistent.
		col.SetBiome(1, 64, 1, uint32(biome.Desert{}.EncodeBiome()))
		d := subChunkNetData(col, ind, true)
		if !bytes.Equal(d.payload, chunk.EncodeSubChunk(col.Chunk, chunk.NetworkEncoding, int(ind))) {
			t.Fatal("payload stale after biome change")
		}
		// State derivation writes through Column.SetBlock, which must invalidate.
		col.SetBlock(3, 64, 3, 0, world.DefaultBlockRegistry.BlockRuntimeID(block.Gold{}))
		e := subChunkNetData(col, ind, true)
		if bytes.Equal(e.payload, d.payload) {
			t.Fatal("Column.SetBlock did not invalidate")
		}
		// Fake chunks bypass the cache.
		f1, f2 := subChunkNetData(col, ind, false), subChunkNetData(col, ind, false)
		if f1 == f2 {
			t.Fatal("uncached entry was shared")
		}
	})
}

func BenchmarkSubChunkEntry(b *testing.B) {
	registerTestRecipes()
	w := world.Config{Synchronous: true}.New()
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		col := loadedColumn(tx)
		for x := range 16 {
			for z := range 16 {
				tx.SetBlock(cube.Pos{x, 60, z}, block.Stone{}, nil)
			}
		}
		ind := int16(col.SubIndex(60))
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			subChunkNetData(col, ind, true) // viewer 1
			subChunkNetData(col, ind, true) // viewer 2
		}
	})
}
