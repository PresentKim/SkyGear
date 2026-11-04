// Package invclick is what the vanilla inventory does with a click: the
// stack held on the cursor and the slots it is clicked on, dragged over or
// collected from. A screen keeps its slots in Areas and its cursor stack,
// and asks these functions for the result.
package invclick

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
)

// Area is a run of slots.
type Area interface {
	Len() int
	Item(i int) item.Stack
	Set(i int, st item.Stack)
	// Accepts reports whether st may be put into slot i.
	Accepts(i int, st item.Stack) bool
}

// Ref is one slot of an Area.
type Ref struct {
	A Area
	I int
}

// Click is a left click (right false) or a right click on slot i of a with
// cursor held; it returns the stack held after it.
func Click(a Area, i int, cursor item.Stack, right bool) item.Stack {
	slot := a.Item(i)
	switch {
	case cursor.Empty() && slot.Empty():
		return cursor
	case cursor.Empty():
		take := slot.Count()
		if right {
			take = (take + 1) / 2
		}
		a.Set(i, slot.Grow(-take))
		return slot.Grow(take - slot.Count())
	case !a.Accepts(i, cursor):
		return cursor
	case slot.Empty() || slot.Comparable(cursor):
		put := cursor.Count()
		if right {
			put = 1
		}
		room := cursor.MaxCount() - slot.Count()
		put = min(put, room)
		if put <= 0 {
			return cursor
		}
		a.Set(i, cursor.Grow(slot.Count()+put-cursor.Count()))
		return cursor.Grow(-put)
	default:
		a.Set(i, cursor)
		return slot
	}
}

// Collect fills cursor up to its largest stack from the slots of areas that
// hold the same item, the smaller stacks first like the vanilla double
// click.
func Collect(cursor item.Stack, areas ...Area) item.Stack {
	if cursor.Empty() {
		return cursor
	}
	for _, full := range []bool{false, true} {
		for _, a := range areas {
			for i := range a.Len() {
				room := cursor.MaxCount() - cursor.Count()
				if room <= 0 {
					return cursor
				}
				st := a.Item(i)
				if st.Empty() || !st.Comparable(cursor) || (st.Count() == st.MaxCount()) != full {
					continue
				}
				n := min(room, st.Count())
				a.Set(i, st.Grow(-n))
				cursor = cursor.Grow(n)
			}
		}
	}
	return cursor
}

// DoubleClick is the second left click of a double click on slot i of a:
// with nothing held it picks the slot up first, then the cursor collects
// the same item from the areas (Collect).
func DoubleClick(a Area, i int, cursor item.Stack, areas ...Area) item.Stack {
	if cursor.Empty() {
		cursor = Click(a, i, cursor, false)
	}
	return Collect(cursor, areas...)
}

// Drag spreads cursor over the distinct slots dragged across: evenly (right
// false) or one each, onto empty slots and slots of the same item that take
// it. It returns what stays on the cursor.
func Drag(slots []Ref, cursor item.Stack, right bool) item.Stack {
	var to []Ref
	for _, r := range slots {
		st := r.A.Item(r.I)
		if !r.A.Accepts(r.I, cursor) || (!st.Empty() && !st.Comparable(cursor)) || st.Count() >= cursor.MaxCount() {
			continue
		}
		to = append(to, r)
	}
	if cursor.Empty() || len(to) == 0 {
		return cursor
	}
	each := 1
	if !right {
		each = max(1, cursor.Count()/len(to))
	}
	for _, r := range to {
		if cursor.Empty() {
			break
		}
		st := r.A.Item(r.I)
		n := min(each, cursor.Count(), cursor.MaxCount()-st.Count())
		r.A.Set(r.I, cursor.Grow(st.Count()+n-cursor.Count()))
		cursor = cursor.Grow(-n)
	}
	return cursor
}

// Move puts st into the areas, in order: onto the stacks of the same item
// first, then into empty slots. It returns what did not fit.
func Move(st item.Stack, to ...Area) item.Stack {
	for _, empty := range []bool{false, true} {
		for _, a := range to {
			for i := range a.Len() {
				if st.Empty() {
					return st
				}
				has := a.Item(i)
				if has.Empty() != empty || (!empty && !has.Comparable(st)) || !a.Accepts(i, st) {
					continue
				}
				n := min(st.Count(), st.MaxCount()-has.Count())
				if n <= 0 {
					continue
				}
				a.Set(i, st.Grow(has.Count()+n-st.Count()))
				st = st.Grow(-n)
			}
		}
	}
	return st
}

// Inventory is the slots from to to (not included) of an inventory, in that
// order.
type Inventory struct {
	Inv      *inventory.Inventory
	From, To int
}

func (v Inventory) Len() int { return v.To - v.From }

func (v Inventory) Item(i int) item.Stack {
	st, _ := v.Inv.Item(v.From + i)
	return st
}

func (v Inventory) Set(i int, st item.Stack) { _ = v.Inv.SetItem(v.From+i, st) }

func (Inventory) Accepts(int, item.Stack) bool { return true }

// Main and Hotbar are the 27 inventory slots and the 9 hotbar slots of a
// player's inventory.
func Main(inv *inventory.Inventory) Inventory   { return Inventory{inv, 9, 36} }
func Hotbar(inv *inventory.Inventory) Inventory { return Inventory{inv, 0, 9} }

// Slots is an Area of plain slots kept in a slice.
type Slots []item.Stack

func (s Slots) Len() int                   { return len(s) }
func (s Slots) Item(i int) item.Stack      { return s[i] }
func (s Slots) Set(i int, st item.Stack)   { s[i] = st }
func (Slots) Accepts(int, item.Stack) bool { return true }
