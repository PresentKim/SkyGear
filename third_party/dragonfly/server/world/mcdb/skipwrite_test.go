package mcdb

import (
	"bytes"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
)

type batchOps struct{ puts, dels [][]byte }

func (o *batchOps) Put(k, v []byte) { o.puts = append(o.puts, bytes.Clone(k)) }
func (o *batchOps) Delete(k []byte) { o.dels = append(o.dels, bytes.Clone(k)) }

func openTest(t *testing.T, dir string) (*DB, *[]batchOps) {
	t.Helper()
	br := world.NewBlockRegistry()
	br.Finalize()
	db, err := Config{Blocks: br}.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	var seen []batchOps
	db.onBatch = func(b *leveldb.Batch) {
		var o batchOps
		_ = b.Replay(&o)
		seen = append(seen, o)
	}
	return db, &seen
}

func column(db *DB, fill uint32) *chunk.Column {
	c := chunk.New(db.conf.Blocks, world.Overworld.Range())
	if fill != 0 {
		c.SetBlock(1, 10, 1, 0, fill)
	}
	return &chunk.Column{Chunk: c, BlockEntities: []chunk.BlockEntity{{Pos: [3]int{1, 10, 1}, Data: map[string]any{"id": "X"}}}}
}

func TestSecondStoreNoPuts(t *testing.T) {
	db, seen := openTest(t, t.TempDir())
	defer db.Close()
	col := column(db, 5)
	if err := db.StoreColumn(world.ChunkPos{1, 2}, world.Overworld, col); err != nil {
		t.Fatal(err)
	}
	if err := db.StoreColumn(world.ChunkPos{1, 2}, world.Overworld, col); err != nil {
		t.Fatal(err)
	}
	first, second := (*seen)[0], (*seen)[1]
	if len(first.puts) < 4 {
		t.Fatalf("first store has %d puts", len(first.puts))
	}
	// Only the block entities are written again.
	if len(second.puts) != 1 {
		t.Fatalf("second store has %d puts, want 1 (block entities)", len(second.puts))
	}
}

func TestChangedSubchunkWritten(t *testing.T) {
	dir := t.TempDir()
	db, seen := openTest(t, dir)
	pos := world.ChunkPos{0, 0}
	_ = db.StoreColumn(pos, world.Overworld, column(db, 5))
	_ = db.StoreColumn(pos, world.Overworld, column(db, 6))
	if len((*seen)[1].puts) < 2 {
		t.Fatalf("changed column wrote %d puts", len((*seen)[1].puts))
	}
	db.Close()
	db2, _ := openTest(t, dir)
	defer db2.Close()
	got, err := db2.LoadColumn(pos, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	if got.Chunk.Block(1, 10, 1, 0) != 6 {
		t.Fatalf("block after reopen = %d, want 6", got.Chunk.Block(1, 10, 1, 0))
	}
}

func TestEmptySubchunkDeleted(t *testing.T) {
	dir := t.TempDir()
	db, seen := openTest(t, dir)
	pos := world.ChunkPos{3, 3}
	_ = db.StoreColumn(pos, world.Overworld, column(db, 5))
	_ = db.StoreColumn(pos, world.Overworld, column(db, 0))
	second := (*seen)[1]
	if len(second.dels) == 0 {
		t.Fatal("emptied sub chunk was not deleted")
	}
	// Store the empty column again: nothing but the block entities is written, deletes are not repeated for sub chunks.
	_ = db.StoreColumn(pos, world.Overworld, column(db, 0))
	db.Close()
	db2, _ := openTest(t, dir)
	defer db2.Close()
	got, err := db2.LoadColumn(pos, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	air := db2.conf.Blocks.BlockRuntimeID(db2.conf.Blocks.Air())
	if got.Chunk.Block(1, 10, 1, 0) != air {
		t.Fatalf("block after reopen = %d, want air %d", got.Chunk.Block(1, 10, 1, 0), air)
	}
}

func TestConcurrentStoreLoad(t *testing.T) {
	db, _ := openTest(t, t.TempDir())
	defer db.Close()
	db.onBatch = nil
	var wg sync.WaitGroup
	for g := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 20 {
				pos := world.ChunkPos{int32(g), int32(i % 3)}
				col := column(db, uint32(5+i%2))
				if err := db.StoreColumn(pos, world.Overworld, col); err != nil {
					t.Error(err)
					return
				}
				got, err := db.LoadColumn(pos, world.Overworld)
				if err != nil {
					t.Error(err)
					return
				}
				if b := got.Chunk.Block(1, 10, 1, 0); b != uint32(5+i%2) {
					t.Errorf("loaded block %d, want %d", b, 5+i%2)
					return
				}
			}
		}()
	}
	wg.Wait()
}
