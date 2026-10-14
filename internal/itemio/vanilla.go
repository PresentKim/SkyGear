package itemio

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
)

// hopperInsertable runs the hopper rules of a Dragonfly block (furnaces,
// brewing stands, ...) with a stand-in hopper that holds one item at a time
// and faces the block like a hopper on the face the items come through.
type hopperInsertable struct{ b block.HopperInsertable }

func (h hopperInsertable) InsertItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack) int {
	n := 0
	for n < st.Count() {
		hopper := block.NewHopper()
		hopper.Facing = face.Opposite()
		_ = hopper.Inventory(tx, pos).SetItem(0, st.Grow(1-st.Count()))
		if !h.b.InsertItem(hopper, pos, tx) {
			break
		}
		n++
	}
	return n
}

// container moves items in and out of the inventory of a block.Container,
// through any face.
type container struct{}

func inventoryOf(tx *world.Tx, pos cube.Pos) *inventory.Inventory {
	if c, ok := tx.Block(pos).(block.Container); ok {
		return c.Inventory(tx, pos)
	}
	return nil
}

func (container) InsertItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack) int {
	inv := inventoryOf(tx, pos)
	if inv == nil {
		return 0
	}
	n, _ := inv.AddItem(st)
	return n
}

func (container) PeekItems(tx *world.Tx, pos cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	return peekSlots(inventoryOf(tx, pos), nil, want)
}

func (container) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	return takeSlots(inventoryOf(tx, pos), nil, st, n)
}

// peekSlots is the first stack of the slots (all if nil) want accepts.
func peekSlots(inv *inventory.Inventory, slots []int, want func(item.Stack) bool) item.Stack {
	if inv == nil {
		return item.Stack{}
	}
	if slots == nil {
		for i := range inv.Size() {
			slots = append(slots, i)
		}
	}
	for _, i := range slots {
		if st, _ := inv.Item(i); !st.Empty() && (want == nil || want(st)) {
			return st
		}
	}
	return item.Stack{}
}

// takeSlots takes up to n items of the kind of st from the slots (all if
// nil) and returns them.
func takeSlots(inv *inventory.Inventory, slots []int, st item.Stack, n int) item.Stack {
	if inv == nil {
		return item.Stack{}
	}
	if slots == nil {
		for i := range inv.Size() {
			slots = append(slots, i)
		}
	}
	got := 0
	var out item.Stack
	for _, i := range slots {
		if got >= n {
			break
		}
		cur, _ := inv.Item(i)
		if cur.Empty() || !cur.Comparable(st) {
			continue
		}
		take := min(n-got, cur.Count())
		_ = inv.SetItem(i, cur.Grow(-take))
		got += take
		out = cur.Grow(got - cur.Count())
	}
	return out
}

// smelterOutput takes the products of a furnace, blast furnace or smoker
// through its bottom face: the output slot, and the fuel slot's non-fuel
// (empty buckets), as a hopper under it does.
type smelterOutput struct{}

func smelterSlots(tx *world.Tx, pos cube.Pos) (*inventory.Inventory, []int) {
	inv := inventoryOf(tx, pos)
	if inv == nil {
		return nil, nil
	}
	slots := []int{2}
	if fuel, _ := inv.Item(1); !fuel.Empty() {
		if f, ok := fuel.Item().(item.Fuel); !ok || f.FuelInfo().Duration == 0 {
			slots = append(slots, 1)
		}
	}
	return inv, slots
}

func (smelterOutput) PeekItems(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack {
	if face != cube.FaceDown {
		return item.Stack{}
	}
	inv, slots := smelterSlots(tx, pos)
	return peekSlots(inv, slots, want)
}

func (smelterOutput) ExtractItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack {
	if face != cube.FaceDown {
		return item.Stack{}
	}
	inv, slots := smelterSlots(tx, pos)
	return takeSlots(inv, slots, st, n)
}

// brewingOutput takes the potions of a brewing stand (its three bottle
// slots) through its bottom face.
type brewingOutput struct{}

var brewingSlots = []int{1, 2, 3}

func (brewingOutput) PeekItems(tx *world.Tx, pos cube.Pos, face cube.Face, want func(item.Stack) bool) item.Stack {
	if face != cube.FaceDown {
		return item.Stack{}
	}
	return peekSlots(inventoryOf(tx, pos), brewingSlots, want)
}

func (brewingOutput) ExtractItems(tx *world.Tx, pos cube.Pos, face cube.Face, st item.Stack, n int) item.Stack {
	if face != cube.FaceDown {
		return item.Stack{}
	}
	return takeSlots(inventoryOf(tx, pos), brewingSlots, st, n)
}
