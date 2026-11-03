package itemio

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Lister is a block that can tell what it holds, for storages that mirror an
// outside inventory (the storage network's external storage).
type Lister interface {
	// ListItems returns everything the block holds that can be taken through
	// face, one stack per kind with the whole count.
	ListItems(tx *world.Tx, pos cube.Pos, face cube.Face) []item.Stack
}

// ListerAdapter gives the Lister of a block that has no method of its own
// (nil if the block cannot list).
type ListerAdapter func(tx *world.Tx, pos cube.Pos, b world.Block) Lister

var listers []ListerAdapter

// RegisterLister adds an adapter; adapters run before the vanilla rules.
func RegisterLister(a ListerAdapter) { listers = append(listers, a) }

// List returns what the block at pos holds, one stack per kind. Plain
// containers list every slot; furnaces and brewing stands list nothing (their
// inputs cannot be taken back out through a face).
func List(tx *world.Tx, pos cube.Pos, face cube.Face) []item.Stack {
	b := tx.Block(pos)
	if l, ok := b.(Lister); ok {
		return l.ListItems(tx, pos, face)
	}
	for _, a := range listers {
		if l := a(tx, pos, b); l != nil {
			return l.ListItems(tx, pos, face)
		}
	}
	switch b.(type) {
	case block.Furnace, block.BlastFurnace, block.Smoker, block.BrewingStand:
		return nil
	}
	inv := inventoryOf(tx, pos)
	if inv == nil {
		return nil
	}
	var out []item.Stack
next:
	for _, st := range inv.Slots() {
		if st.Empty() {
			continue
		}
		for i := range out {
			if out[i].Comparable(st) {
				out[i] = out[i].Grow(st.Count())
				continue next
			}
		}
		out = append(out, st)
	}
	return out
}
