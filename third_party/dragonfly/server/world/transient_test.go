package world

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
)

type transientType struct{ testEntityType }

func (transientType) Transient() bool { return true }

type colStore struct {
	NopProvider
	last *chunk.Column
}

func (p *colStore) StoreColumn(_ ChunkPos, _ Dimension, c *chunk.Column) error {
	p.last = c
	return nil
}

func spawnAt(w *World, t EntityType) *EntityHandle {
	h := EntitySpawnOpts{Position: mgl64.Vec3{1, 4, 1}}.New(t, testEntityConfig{})
	<-w.exec(func(tx *Tx) { tx.AddEntity(h) })
	return h
}

func TestTransientNotStored(t *testing.T) {
	p := &colStore{}
	w := Config{Synchronous: true, Provider: p}.New()
	defer w.Close()
	addCols(w, 1, false)
	h := spawnAt(w, transientType{})
	<-w.exec(func(tx *Tx) {
		if w.chunks[ChunkPos{0, 0}].modified {
			t.Error("adding a transient entity set modified")
		}
	})
	<-w.exec(func(tx *Tx) { w.chunks[ChunkPos{0, 0}].modified = true })
	w.Save()
	if p.last == nil {
		t.Fatal("column not stored")
	}
	if len(p.last.Entities) != 0 {
		t.Fatalf("stored %d entities, want 0 transient", len(p.last.Entities))
	}
	<-w.exec(func(tx *Tx) {
		w.chunks[ChunkPos{0, 0}].modified = false
		e, _ := h.Entity(tx)
		tx.RemoveEntity(e)
		if w.chunks[ChunkPos{0, 0}].modified {
			t.Error("removing a transient entity set modified")
		}
	})
	// A normal entity is still stored and marks the column.
	p.last = nil
	spawnAt(w, testEntityType{})
	w.Save()
	if p.last == nil || len(p.last.Entities) != 1 {
		t.Fatalf("normal entity not stored: %+v", p.last)
	}
}

func TestPlayersIndex(t *testing.T) {
	w := Config{Synchronous: true}.New()
	defer w.Close()
	addCols(w, 1, false)
	pl := spawnAt(w, playerLikeType{})
	spawnAt(w, testEntityType{})
	count := func() (n int) {
		<-w.exec(func(tx *Tx) {
			for range tx.Players() {
				n++
			}
		})
		return
	}
	if n := count(); n != 1 {
		t.Fatalf("players = %d, want 1", n)
	}
	<-w.exec(func(tx *Tx) { e, _ := pl.Entity(tx); tx.RemoveEntity(e) })
	if n := count(); n != 0 {
		t.Fatalf("players after remove = %d, want 0", n)
	}
	_ = cube.Pos{}
}

type playerLikeType struct{ testEntityType }

func (playerLikeType) EncodeEntity() string { return "minecraft:player" }

func BenchmarkEntitiesWithin(b *testing.B) {
	w := Config{Synchronous: true}.New()
	defer w.Close()
	addCols(w, 1, false)
	for range 200 {
		spawnAt(w, testEntityType{})
	}
	for range 500 {
		spawnAt(w, transientType{})
	}
	box := cube.Box(0, 0, 0, 16, 16, 16)
	b.ReportAllocs()
	b.ResetTimer()
	<-w.exec(func(tx *Tx) {
		for range b.N {
			for range w.entitiesWithin(tx, box) {
			}
		}
	})
}
