package world

import (
	"reflect"
	"runtime/debug"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// skygear patch: hooks that report block changes and column loads/unloads.

// BlockChangeHook may be implemented by a Block to be told when it is
// replaced or placed. BlockChanged is called once per Tx.SetBlock, after the
// block was stored and the neighbours were updated: on the block that was
// there before if it implements BlockChangeHook, otherwise on the block that
// is there now if that does. It is not called if both are equal.
//
// The hook may call Tx.SetBlock itself. A change at pos made while the hook
// of pos is running is stored but does not run the hook again.
type BlockChangeHook interface {
	BlockChanged(tx *Tx, pos cube.Pos, before, after Block)
}

// ColumnHandler may be implemented by the Handler of a World to be told when
// a column is added to or removed from the world. blockEntities is the map of
// that column itself: HandleColumnUnload runs before the column is saved, so
// what it writes there is saved. Both are called in a transaction.
type ColumnHandler interface {
	HandleColumnLoad(tx *Tx, pos ChunkPos, blockEntities map[cube.Pos]Block)
	HandleColumnUnload(tx *Tx, pos ChunkPos, blockEntities map[cube.Pos]Block)
}

// blockChanged runs the BlockChangeHook for a change of the block at pos.
func (tx *Tx) blockChanged(pos cube.Pos, before, after Block) {
	var hook BlockChangeHook
	if h, ok := before.(BlockChangeHook); ok {
		hook = h
	} else if h, ok := after.(BlockChangeHook); ok {
		hook = h
	}
	if hook == nil || blocksEqual(before, after) {
		return
	}
	w := tx.World()
	if _, busy := w.hookActive[pos]; busy {
		return
	}
	if w.hookActive == nil {
		w.hookActive = make(map[cube.Pos]struct{})
	}
	w.hookActive[pos] = struct{}{}
	defer delete(w.hookActive, pos)
	hook.BlockChanged(tx, pos, before, after)
}

// blocksEqual checks two blocks for equality by value without panicking on
// blocks that are not comparable.
func blocksEqual(a, b Block) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta := reflect.TypeOf(a)
	if ta != reflect.TypeOf(b) {
		return false
	}
	if ta.Comparable() {
		return safeEqual(a, b)
	}
	return reflect.DeepEqual(a, b)
}

// safeEqual compares two comparable interface values, treating a panic
// (interface fields holding uncomparable values) as inequality.
func safeEqual(a, b Block) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = reflect.DeepEqual(a, b)
		}
	}()
	return a == b
}

// columnLoaded tells the ColumnHandler of the world that a column was added.
func (w *World) columnLoaded(tx *Tx, pos ChunkPos, c *Column) {
	if h, ok := w.Handler().(ColumnHandler); ok {
		w.runColumnHook(func() { h.HandleColumnLoad(tx, pos, c.BlockEntities) })
	}
}

// columnUnloaded tells the ColumnHandler of the world that a column is about
// to be saved and removed.
func (w *World) columnUnloaded(tx *Tx, pos ChunkPos, c *Column) {
	if h, ok := w.Handler().(ColumnHandler); ok {
		w.runColumnHook(func() { h.HandleColumnUnload(tx, pos, c.BlockEntities) })
	}
}

// runColumnHook runs f and logs a panic, so a faulty hook cannot leave a
// column half added or half removed.
func (w *World) runColumnHook(f func()) {
	defer func() {
		if r := recover(); r != nil {
			w.conf.Log.Error("column hook panicked", "panic", r, "stack", string(debug.Stack()))
		}
	}()
	f()
}
