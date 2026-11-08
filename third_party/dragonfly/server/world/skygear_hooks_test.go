package world

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// skygear patch: tests of BlockChangeHook and ColumnHandler.

type hookCounter struct {
	mu    sync.Mutex
	calls []hookCall
	fn    func(tx *Tx, pos cube.Pos, before, after Block)
}

type hookCall struct {
	pos           cube.Pos
	before, after Block
}

func (h *hookCounter) n() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.calls)
}

func (h *hookCounter) record(tx *Tx, pos cube.Pos, before, after Block) {
	h.mu.Lock()
	h.calls = append(h.calls, hookCall{pos, before, after})
	h.mu.Unlock()
	if h.fn != nil {
		h.fn(tx, pos, before, after)
	}
}

func hookWorld(p Provider, h Handler) *World {
	w := Config{Synchronous: true, Blocks: hookRegistry(), Provider: p, SaveInterval: time.Hour, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}.New()
	if h != nil {
		w.Handle(h)
	}
	return w
}

type hookBlock struct {
	c *hookCounter
	v int
}

func (hookBlock) EncodeBlock() (string, map[string]any) { return "skygear:hook_test", nil }
func (hookBlock) Hash() (uint64, uint64)                { return 0xfeed0001, 0 }
func (hookBlock) Model() BlockModel                     { return unitModel{} }
func (b hookBlock) EncodeNBT() map[string]any           { return map[string]any{"v": int32(b.v)} }
func (b hookBlock) DecodeNBT(m map[string]any) any      { b.v = int(m["v"].(int32)); return b }
func (b hookBlock) BlockChanged(tx *Tx, pos cube.Pos, before, after Block) {
	b.c.record(tx, pos, before, after)
}

type unitModel struct{}

func (unitModel) BBox(cube.Pos, BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 1, 1)}
}
func (unitModel) FaceSolid(cube.Pos, cube.Face, BlockSource) bool { return true }

type colHandler struct {
	NopHandler
	load, unload func(tx *Tx, pos ChunkPos, be map[cube.Pos]Block)
}

func (h colHandler) HandleColumnLoad(tx *Tx, pos ChunkPos, be map[cube.Pos]Block) {
	if h.load != nil {
		h.load(tx, pos, be)
	}
}
func (h colHandler) HandleColumnUnload(tx *Tx, pos ChunkPos, be map[cube.Pos]Block) {
	if h.unload != nil {
		h.unload(tx, pos, be)
	}
}

func hookRegistry() BlockRegistry {
	r := NewBlockRegistry()
	r.RegisterBlockState(BlockState{Name: "skygear:hook_test", Properties: map[string]any{}})
	r.RegisterBlock(hookBlock{})
	return r
}

func regHook() {}

func TestBlockChangedOnce(t *testing.T) {
	regHook()
	c := &hookCounter{}
	w := hookWorld(NopProvider{}, nil)
	defer w.Close()
	pos := cube.Pos{1, 64, 1}
	<-w.exec(func(tx *Tx) {
		tx.SetBlock(pos, hookBlock{c: c, v: 1}, nil) // after implements: 1
		if c.n() != 1 {
			t.Fatalf("placing: %d calls, want 1", c.n())
		}
		tx.SetBlock(pos, hookBlock{c: c, v: 1}, nil) // same value: 0
		if c.n() != 1 {
			t.Fatalf("same value: %d calls, want 1", c.n())
		}
		tx.SetBlock(pos, w.conf.Blocks.Air(), nil) // before implements: 1
		if c.n() != 2 {
			t.Fatalf("removing: %d calls, want 2", c.n())
		}
		if got := c.calls[1]; got.before.(hookBlock).v != 1 || got.after != w.conf.Blocks.Air() {
			t.Fatalf("bad args %+v", got)
		}
		tx.SetBlock(pos, w.conf.Blocks.Air(), nil)
		if c.n() != 2 {
			t.Fatalf("air over air: %d calls, want 2", c.n())
		}
	})
}

func TestBlockChangedReentrant(t *testing.T) {
	regHook()
	c := &hookCounter{}
	pos := cube.Pos{2, 64, 2}
	c.fn = func(tx *Tx, p cube.Pos, before, after Block) {
		if _, ok := after.(hookBlock); ok {
			tx.SetBlock(p, hookBlock{c: c, v: after.(hookBlock).v + 1}, nil)
		}
	}
	w := hookWorld(NopProvider{}, nil)
	defer w.Close()
	<-w.exec(func(tx *Tx) {
		tx.SetBlock(pos, hookBlock{c: c, v: 1}, nil)
		if got := tx.Block(pos).(hookBlock).v; got != 2 {
			t.Fatalf("block v = %d, want 2", got)
		}
	})
	if c.n() != 1 {
		t.Fatalf("%d calls, want 1 (nested change on the same position is not re-announced)", c.n())
	}
}

func TestHookPanicRecovered(t *testing.T) {
	regHook()
	c := &hookCounter{fn: func(*Tx, cube.Pos, Block, Block) { panic("boom") }}
	w := hookWorld(NopProvider{}, nil)
	defer w.Close()
	<-w.exec(func(tx *Tx) { tx.SetBlock(cube.Pos{3, 64, 3}, hookBlock{c: c}, nil) })
	done := false
	<-w.exec(func(tx *Tx) { done = true })
	if !done {
		t.Fatal("world did not keep running")
	}
	// The same must hold when the hook panics at a SetBlock made by the
	// panic-free path again.
	<-w.exec(func(tx *Tx) { tx.SetBlock(cube.Pos{3, 64, 3}, tx.World().conf.Blocks.Air(), nil) })
}

type saveProvider struct {
	NopProvider
	mu     sync.Mutex
	stored map[ChunkPos]*chunk.Column
}

func (p *saveProvider) StoreColumn(pos ChunkPos, _ Dimension, c *chunk.Column) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := *c
	cp.Chunk = nil
	p.stored[pos] = &cp
	return nil
}

func TestColumnLoadHook(t *testing.T) {
	regHook()
	var loads []ChunkPos
	var tx0 *Tx
	h := colHandler{load: func(tx *Tx, pos ChunkPos, be map[cube.Pos]Block) {
		loads = append(loads, pos)
		tx0 = tx
		if be == nil {
			t.Error("nil block entity map")
		}
	}}
	w := hookWorld(NopProvider{}, h)
	defer w.Close()
	<-w.exec(func(tx *Tx) {
		tx.SetBlock(cube.Pos{17, 64, 0}, tx.World().conf.Blocks.Air(), nil) // loads chunk (1,0)
		tx.Block(cube.Pos{17, 64, 0})
	})
	if len(loads) != 1 || loads[0] != (ChunkPos{1, 0}) || tx0 == nil {
		t.Fatalf("loads = %v", loads)
	}
}

func TestColumnUnloadBeforeSave(t *testing.T) {
	regHook()
	p := &saveProvider{stored: map[ChunkPos]*chunk.Column{}}
	pos := cube.Pos{4, 64, 4}
	h := colHandler{unload: func(tx *Tx, cp ChunkPos, be map[cube.Pos]Block) {
		be[pos] = hookBlock{v: 42}
	}}
	w := hookWorld(p, h)
	<-w.exec(func(tx *Tx) {
		tx.SetBlock(pos, hookBlock{c: &hookCounter{}, v: 1}, nil)
		w.closeChunk(tx, ChunkPos{0, 0}, w.chunks[ChunkPos{0, 0}])
	})
	got := p.stored[ChunkPos{0, 0}]
	if got == nil {
		t.Fatal("column not saved")
	}
	var found any
	for _, be := range got.BlockEntities {
		if be.Pos == pos {
			found = be.Data["v"]
		}
	}
	if found != int32(42) {
		t.Fatalf("saved block entity value = %v, want 42 written by the unload hook", found)
	}
	_ = w.Close()
}
