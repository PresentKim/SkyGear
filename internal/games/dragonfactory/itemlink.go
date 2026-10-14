package dragonfactory

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/itemio"
)

// How the game's blocks move items (docs/tech/itemio.md): everything goes
// through internal/itemio. Machines, drawers and barrels implement its
// interfaces; the composter (a vanilla block) is served by an adapter; hoppers
// of every kind (the vanilla one through a Dragonfly patch hook, ours in
// hopper.go) move one item per step with stepHopper.

var registerItemIOOnce sync.Once

// registerItemIO connects the game's blocks to itemio and the vanilla hopper
// to the game's item transfer.
func registerItemIO() {
	registerItemIOOnce.Do(func() {
		itemio.RegisterAdapter(func(tx *world.Tx, _ cube.Pos, b world.Block) (itemio.Inserter, itemio.Extractor) {
			if _, ok := b.(block.Composter); ok {
				if s := activeGameSession(tx); s != nil {
					return composterIO{s}, composterIO{s}
				}
			}
			return nil, nil
		})
		block.HopperTransfer = func(h block.Hopper, pos cube.Pos, tx *world.Tx) (bool, bool) {
			return stepHopper(tx, pos, h.Inventory(tx, pos), h.Facing, cube.FaceUp)
		}
	})
}

// hopperCooldown is the ticks a hopper waits after it moved an item
// (vanilla's 8).
const hopperCooldown = 8

// stepHopper moves one item from inv into the block on the out side of pos
// and one item from the block on the in side of pos into inv. A vanilla
// hopper faces out and takes from above; a reverse hopper faces up and takes
// from below. It reports what moved.
func stepHopper(tx *world.Tx, pos cube.Pos, inv *inventory.Inventory, out, in cube.Face) (inserted, extracted bool) {
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

// coolHopper makes a hopper that just got or lost an item wait, so the item
// does not run through a line of hoppers in one tick.
func coolHopper(tx *world.Tx, pos cube.Pos) {
	switch b := tx.Block(pos).(type) {
	case block.Hopper:
		b.TransferCooldown = hopperCooldown
		tx.SetBlock(pos, b, nil)
	case hopperLike:
		tx.SetBlock(pos, b.cooled(), nil)
	}
}

// hopperLike is a hopper of the game: it can be set to wait.
type hopperLike interface {
	world.Block
	cooled() world.Block
}
