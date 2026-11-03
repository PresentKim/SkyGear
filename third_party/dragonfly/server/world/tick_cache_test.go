package world

import (
	"maps"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

// skygear patch: tests for the cached tick set.

// oldTickSet computes the tick set the way the uncached tick did.
func oldTickSet(w *World, r int32) map[ChunkPos]*Column {
	var loaded []ChunkPos
	_, loaders := w.allViewers()
	for _, l := range loaders {
		l.mu.RLock()
		loaded = append(loaded, l.pos)
		l.mu.RUnlock()
	}
	out := map[ChunkPos]*Column{}
	for pos, c := range w.chunks {
		if (ticker{}).anyWithinDistance(pos, loaded, r) {
			out[pos] = c
		}
	}
	return out
}

// TestTickSetMatchesFresh verifies the cached tick set equals the freshly
// computed one after every kind of mutation.
func TestTickSetMatchesFresh(t *testing.T) {
	w := Config{}.New()
	defer w.Close()

	runWorld(w, func(tx *Tx) {
		check := func(step string, r int32) {
			t.Helper()
			got := map[ChunkPos]*Column{}
			for _, tc := range w.tickSet.get(w, r) {
				if _, dup := got[tc.pos]; dup {
					t.Fatalf("%s: duplicate chunk %v", step, tc.pos)
				}
				got[tc.pos] = tc.col
			}
			if want := oldTickSet(w, r); !maps.Equal(got, want) {
				t.Fatalf("%s: cached %d chunks, fresh %d", step, len(got), len(want))
			}
		}
		load := func(cx, cz int) { tx.Block(cube.Pos{cx * 16, 64, cz * 16}) }

		for x := -20; x < 20; x++ {
			for z := -20; z < 20; z++ {
				load(x, z)
			}
		}
		check("no loaders", 3)

		a := NewLoader(0, w, NopViewer{})
		b := NewLoader(0, w, NopViewer{})
		check("add", 3)
		a.Move(tx, mgl64.Vec3{5 * 16, 64, 5 * 16})
		check("move a", 3)
		b.Move(tx, mgl64.Vec3{-8 * 16, 64, 2 * 16})
		check("move b", 3)
		check("range", 6)
		w.SetTickRange(2)
		check("set range", 2)
		a.Move(tx, mgl64.Vec3{100 * 16, 64, 0})
		check("move out of loaded", 3)
		a.Move(tx, mgl64.Vec3{19 * 16, 64, 19 * 16})
		check("move to edge", 3)
		load(30, 30)
		check("chunk added", 3)
		a.Move(tx, mgl64.Vec3{30 * 16, 64, 30 * 16})
		check("chunk added near", 3)
		w.closeChunk(tx, ChunkPos{30, 30}, w.chunks[ChunkPos{30, 30}])
		check("chunk closed", 3)
		b.Close(tx)
		check("loader closed", 3)
		a.Close(tx)
		check("all closed", 3)
	})
}

// TestTickSetPeriodicRefresh verifies a mutation that bypasses the dirty flag
// is picked up by the forced recompute after tickSetRefresh ticks, and that a
// closed chunk is dropped from the set.
func TestTickSetPeriodicRefresh(t *testing.T) {
	w := Config{}.New()
	defer w.Close()

	runWorld(w, func(tx *Tx) {
		tx.Block(cube.Pos{0, 64, 0})
		l := NewLoader(0, w, NopViewer{})
		defer l.Close(tx)
		if n := len(w.tickSet.get(w, 2)); n != 1 {
			t.Fatalf("got %d chunks, want 1", n)
		}
		// Bypass every dirty marker.
		col := w.chunks[ChunkPos{}]
		delete(w.chunks, ChunkPos{})
		w.tickDirty.Store(false)
		for i := 1; i < tickSetRefresh-1; i++ {
			w.tickSet.get(w, 2)
		}
		if n := len(w.tickSet.get(w, 2)); n != 1 {
			t.Fatalf("set refreshed early: %d chunks", n)
		}
		if n := len(w.tickSet.get(w, 2)); n != 0 {
			t.Fatalf("forced recompute did not run: %d chunks", n)
		}
		_ = col
	})
}

// TestTickSetClosedChunkDropped verifies a chunk closed via closeChunk is
// never returned afterwards.
func TestTickSetClosedChunkDropped(t *testing.T) {
	w := Config{}.New()
	defer w.Close()

	runWorld(w, func(tx *Tx) {
		tx.Block(cube.Pos{0, 64, 0})
		l := NewLoader(0, w, NopViewer{})
		defer l.Close(tx)
		w.tickSet.get(w, 2)
		w.closeChunk(tx, ChunkPos{}, w.chunks[ChunkPos{}])
		for _, tc := range w.tickSet.get(w, 2) {
			if tc.pos == (ChunkPos{}) {
				t.Fatal("closed chunk still in tick set")
			}
		}
	})
}
