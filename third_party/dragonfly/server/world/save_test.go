package world

import (
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// countProvider counts StoreColumn calls per position.
type countProvider struct {
	NopProvider
	mu     sync.Mutex
	stored map[ChunkPos]int
}

func (p *countProvider) StoreColumn(pos ChunkPos, _ Dimension, _ *chunk.Column) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stored[pos]++
	return nil
}

func (p *countProvider) count(pos ChunkPos) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stored[pos]
}

func (p *countProvider) total() (n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.stored {
		n += c
	}
	return
}

func saveWorld(t *testing.T) (*World, *countProvider) {
	p := &countProvider{stored: map[ChunkPos]int{}}
	w := Config{Synchronous: true, Provider: p, SaveInterval: 60 * time.Second}.New()
	return w, p
}

func addCols(w *World, n int, modified bool) {
	<-w.exec(func(tx *Tx) {
		for i := range n {
			w.chunks[ChunkPos{int32(i), 0}] = &Column{
				Chunk:         chunk.New(w.conf.Blocks, w.Range()),
				BlockEntities: map[cube.Pos]Block{},
				modified:      modified,
			}
		}
	})
}

func step(w *World, now time.Time) {
	<-w.exec(func(tx *Tx) { w.incrementalSave(tx, now) })
}

func TestIncrementalSaveCoversAll(t *testing.T) {
	w, p := saveWorld(t)
	defer w.Close()
	const n = 100
	addCols(w, n, true)
	start := time.Now()
	// One tick per second for one SaveInterval.
	for s := 0; s <= 60; s++ {
		step(w, start.Add(time.Duration(s)*time.Second))
		if s == 0 && p.total() > saveBatch {
			t.Fatalf("first step stored %d columns, want <= %d", p.total(), saveBatch)
		}
	}
	for i := range n {
		if p.count(ChunkPos{int32(i), 0}) == 0 {
			t.Fatalf("column %d not stored within one interval", i)
		}
	}
}

func TestCloseSavesEverything(t *testing.T) {
	w, p := saveWorld(t)
	addCols(w, 100, true)
	_ = w.Close()
	for i := range 100 {
		if p.count(ChunkPos{int32(i), 0}) == 0 {
			t.Fatalf("column %d missed save on Close", i)
		}
	}
}

func TestSaveStoresAllModified(t *testing.T) {
	w, p := saveWorld(t)
	defer w.Close()
	addCols(w, 40, true)
	w.Save()
	if p.total() != 40 {
		t.Fatalf("Save stored %d columns, want 40", p.total())
	}
}

func TestModifiedResetOnlyWithoutBlockEntities(t *testing.T) {
	w, p := saveWorld(t)
	defer w.Close()
	addCols(w, 2, true)
	<-w.exec(func(tx *Tx) {
		w.chunks[ChunkPos{1, 0}].BlockEntities[cube.Pos{1, 1, 1}] = saveTestBE{}
	})
	w.Save()
	w.Save()
	if got := p.count(ChunkPos{0, 0}); got != 1 {
		t.Fatalf("plain column stored %d times, want 1", got)
	}
	if got := p.count(ChunkPos{1, 0}); got != 2 {
		t.Fatalf("block entity column stored %d times, want 2", got)
	}
}

type saveTestBE struct{}

func (saveTestBE) EncodeBlock() (string, map[string]any) { return "minecraft:chest", nil }
func (saveTestBE) Hash() (uint64, uint64)                { return 1, 0 }
func (saveTestBE) Model() BlockModel                     { return nil }
func (saveTestBE) DecodeNBT(map[string]any) any          { return saveTestBE{} }
func (saveTestBE) EncodeNBT() map[string]any             { return map[string]any{"id": "Chest"} }
