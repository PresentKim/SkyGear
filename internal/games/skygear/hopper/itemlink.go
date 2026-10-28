package hopper

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/itemio"
)

var RegisterItemIOOnce sync.Once

// hopperCooldown is the ticks a hopper waits after it moved an item
// (vanilla's 8).
const hopperCooldown = 8

// StepHopper moves one item from inv into the block on the out side of pos
// and one item from the block on the in side of pos into inv. A vanilla
// hopper faces out and takes from above; a reverse hopper faces up and takes
// from below. It reports what moved.
func StepHopper(tx *world.Tx, pos cube.Pos, inv *inventory.Inventory, out, in cube.Face) (inserted, extracted bool) {
	dest := pos.Side(out)
	for slot, st := range inv.Slots() {
		if st.Empty() {
			continue
		}
		if itemio.Insert(tx, dest, out.Opposite(), st.Grow(1-st.Count())) == 1 {
			_ = inv.SetItem(slot, st.Grow(-1))
			inserted = true
			coolHopper(tx, dest)
			break
		}
	}
	origin := pos.Side(in)
	from := in.Opposite()
	peek := itemio.Peek(tx, origin, from, func(st item.Stack) bool { return inventoryFits(inv, st) })
	if !peek.Empty() {
		if got := itemio.Extract(tx, origin, from, peek, 1); !got.Empty() {
			if n, _ := inv.AddItem(got); n > 0 {
				extracted = true
				coolHopper(tx, origin)
			}
		}
	}
	return inserted, extracted
}

// inventoryFits reports whether inv has room for one more of st.
func inventoryFits(inv *inventory.Inventory, st item.Stack) bool {
	for _, have := range inv.Slots() {
		if have.Empty() || (have.Comparable(st) && have.Count() < have.MaxCount()) {
			return true
		}
	}
	return false
}

// coolHopper makes a vanilla hopper that just got or lost an item wait, so
// the item does not run through a line of hoppers in one tick (the hoppers
// of the game move on a fixed beat of their own).
func coolHopper(tx *world.Tx, pos cube.Pos) {
	if h, ok := tx.Block(pos).(block.Hopper); ok {
		h.TransferCooldown = hopperCooldown
		tx.SetBlock(pos, h, nil)
	}
}
