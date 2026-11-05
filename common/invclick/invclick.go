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

// Ref is slot I of an Area named Name; refs to one slot share the name and
// the index.
type Ref struct {
	Name string
	A    Area
	I    int
}

func (r Ref) same(o Ref) bool { return r.Name == o.Name && r.I == o.I }

// Click is a left click (right false) or a right click on slot i of a with
// cursor held; it returns the stack held after it. As in Bedrock, a right
// click with a stack held over another item does nothing; a left click
// swaps them.
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
	case right:
		return cursor
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

// Split is a drag of a held stack across slots, spread as it goes like the
// Bedrock inventory: the press places onto its slot at once (Click), and
// each further slot reached spreads the stack held before the press again
// over all of them, evenly (left) or one each (right), the rest staying
// held.
type Split struct {
	right bool
	start item.Stack
	cells []splitCell
	// swapped is set when the press's slot held another item.
	swapped bool
}

type splitCell struct {
	Ref
	base int
}

// StartSplit is the press of a drag on slot r with cursor held, before the
// press's Click: it records what the slot and the cursor hold.
func StartSplit(r Ref, cursor item.Stack, right bool) *Split {
	st := r.A.Item(r.I)
	return &Split{right: right, start: cursor, cells: []splitCell{{r, st.Count()}}, swapped: !st.Empty() && !st.Comparable(cursor)}
}

// Placed reports whether the press placed the held item on its slot, which
// a drag needs (a swap starts none).
func (s *Split) Placed() bool {
	c := s.cells[0]
	st := c.A.Item(c.I)
	return !s.swapped && !st.Empty() && st.Comparable(s.start) && st.Count() > c.base
}

func (s *Split) takes(r Ref) bool {
	st := r.A.Item(r.I)
	return r.A.Accepts(r.I, s.start) && (st.Empty() || st.Comparable(s.start)) && st.Count() < s.start.MaxCount()
}

// Spread adds the slots in refs not reached yet that take the item and
// spreads the stack held before the press over every slot reached. It
// returns what stays held.
func (s *Split) Spread(refs []Ref) item.Stack {
	for _, r := range refs {
		known := false
		for _, c := range s.cells {
			known = known || c.same(r)
		}
		if !known && s.takes(r) {
			s.cells = append(s.cells, splitCell{r, r.A.Item(r.I).Count()})
		}
	}
	each := 1
	if !s.right {
		each = max(1, s.start.Count()/len(s.cells))
	}
	left := s.start.Count()
	for _, c := range s.cells {
		n := max(0, min(each, left, s.start.MaxCount()-c.base))
		left -= n
		if c.base+n == 0 {
			c.A.Set(c.I, item.Stack{})
			continue
		}
		c.A.Set(c.I, s.start.Grow(c.base+n-s.start.Count()))
	}
	if left == 0 {
		return item.Stack{}
	}
	return s.start.Grow(left - s.start.Count())
}

// MoveAll moves every stack of the item of focus out of the areas from with
// put, which takes a stack and returns what it did not take: the Bedrock
// shift double click.
func MoveAll(focus item.Stack, put func(item.Stack) item.Stack, from ...Area) {
	for _, a := range from {
		for i := range a.Len() {
			st := a.Item(i)
			if st.Empty() || !st.Comparable(focus) {
				continue
			}
			a.Set(i, put(st))
		}
	}
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
