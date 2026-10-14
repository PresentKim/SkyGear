// Package itemio is the one interface through which blocks give and take
// items: hoppers, and later the pipes (docs/tech/itemio.md). A block that
// holds items implements Inserter and Extractor (or registers an Adapter for
// a block it cannot add methods to); everything else moves items with
// Insert, Peek, Extract and Transfer, which also know the vanilla blocks
// (chests, furnaces, brewing stands).
//
// Positions and faces: pos is the block that gives or takes, and face is the
// face of that block the items pass through. Items entering a block from
// above enter through cube.FaceUp; items leaving a block downwards leave
// through cube.FaceDown.
package itemio

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Inserter is a block items can be put into.
type Inserter interface {
	// InsertItems puts as many of st as the block takes through face and
	// returns how many it took (0 when none).
	InsertItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int
}

// Extractor is a block items can be taken from.
type Extractor interface {
	// PeekItems returns the first stack that can be taken through face and
	// that want accepts (nil accepts any), without taking it. The stack
	// holds everything available of that kind; it is empty when there is
	// nothing.
	PeekItems(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack
	// ExtractItems takes up to n items of the kind of st through face and
	// returns them (an empty stack when none could be taken).
	ExtractItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack
}

// Adapter gives the Inserter and Extractor (either may be nil) of a block
// that has no methods of its own for them, e.g. a block of another package
// whose items live in the game's state.
type Adapter func(tx *world.Tx, pos cube.Pos, b world.Block) (Inserter, Extractor)

var adapters []Adapter

// RegisterAdapter adds an adapter; adapters run before the vanilla rules.
func RegisterAdapter(a Adapter) { adapters = append(adapters, a) }

// IO returns the Inserter and Extractor of the block at pos; either is nil
// if the block cannot do that.
func IO(tx *world.Tx, pos cube.Pos) (Inserter, Extractor) {
	b := tx.Block(pos)
	in, _ := b.(Inserter)
	ex, _ := b.(Extractor)
	for _, a := range adapters {
		i, e := a(tx, pos, b)
		if in == nil {
			in = i
		}
		if ex == nil {
			ex = e
		}
	}
	if in == nil {
		in = vanillaInserter(b)
	}
	if ex == nil {
		ex = vanillaExtractor(b)
	}
	return in, ex
}

// Insert puts st into the block at pos through face and returns how many
// items were taken.
func Insert(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int {
	if st.Empty() {
		return 0
	}
	in, _ := IO(tx, pos)
	if in == nil {
		return 0
	}
	return max(0, min(st.Count(), in.InsertItems(tx, pos, face, st)))
}

// Peek returns the first stack of the block at pos that can be taken
// through face and want accepts, without taking it.
func Peek(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack {
	_, ex := IO(tx, pos)
	if ex == nil {
		return item.Stack{}
	}
	return ex.PeekItems(tx, pos, face, want)
}

// Extract takes up to n items of the kind of st from the block at pos
// through face.
func Extract(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack {
	_, ex := IO(tx, pos)
	if ex == nil || n <= 0 {
		return item.Stack{}
	}
	return ex.ExtractItems(tx, pos, face, st, n)
}

// Transfer moves up to max items of one kind from the block at from (out of
// its face fromFace) to the block at to (into its face toFace) and returns
// how many moved. want filters what may leave (nil: anything). A kind the
// receiver refuses is skipped for the next kinds the source holds.
func Transfer(tx *world.Tx, from cube.Pos, fromFace cube.Face, to cube.Pos, toFace cube.Face, max int, want func(item.Stack) bool) int {
	if max <= 0 {
		return 0
	}
	in, _ := IO(tx, to)
	_, ex := IO(tx, from)
	if in == nil || ex == nil {
		return 0
	}
	var refused []item.Stack
	for range 64 {
		peek := ex.PeekItems(tx, from, fromFace, func(st item.Stack) bool {
			for _, r := range refused {
				if st.Comparable(r) {
					return false
				}
			}
			return want == nil || want(st)
		})
		if peek.Empty() {
			return 0
		}
		offer := peek.Grow(min(max, peek.Count()) - peek.Count())
		n := min(offer.Count(), in.InsertItems(tx, to, toFace, offer))
		if n <= 0 {
			refused = append(refused, peek)
			continue
		}
		return ex.ExtractItems(tx, from, fromFace, offer, n).Count()
	}
	return 0
}

// vanillaInserter is the Inserter of a block of Dragonfly that has no
// methods of ours: blocks with their own hopper rules (furnaces, brewing
// stands, ...) first, then plain containers.
func vanillaInserter(b world.Block) Inserter {
	if h, ok := b.(block.HopperInsertable); ok {
		return hopperInsertable{h}
	}
	if _, ok := b.(block.Container); ok {
		return container{}
	}
	return nil
}

// vanillaExtractor is the Extractor of a block of Dragonfly: furnaces and
// brewing stands give their products below, containers give any slot.
func vanillaExtractor(b world.Block) Extractor {
	switch b.(type) {
	case block.Furnace, block.BlastFurnace, block.Smoker:
		return smelterOutput{}
	case block.BrewingStand:
		return brewingOutput{}
	}
	if _, ok := b.(block.Container); ok {
		return container{}
	}
	return nil
}
