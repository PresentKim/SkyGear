package waila

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Layout is how the panel lays out a container's contents.
type Layout string

const (
	// LayoutChest shows up to 9 kinds of items in one row, stacks of a kind
	// added up; kinds that do not fit are left out (11th playtest).
	LayoutChest Layout = "chest"
	// LayoutFurnace shows input, fuel and output.
	LayoutFurnace Layout = "furnace"
	// LayoutHopper shows five slots.
	LayoutHopper Layout = "hopper"
	// LayoutBrewing shows a brewing stand's five slots.
	LayoutBrewing Layout = "brewstand"
)

// Container is a container's contents for the panel, in slot order.
type Container struct {
	Layout Layout
	Items  []item.Stack
	// Progress, if not empty, is drawn in place of the furnace arrow (an
	// arrow item filled to the progress), at panel slot progressSlot.
	Progress item.Stack
}

// progressSlot is the panel slot of Container.Progress (unused by the
// furnace layout).
const progressSlot = 17

// panelSlots is how many container slots the panel has (index 8 is kept
// for the block icon by the pack, as in WAILA).
const panelSlots = 18

// Field widths of the subtitle in bytes (the pack's '%.Ns' cuts bytes): the
// icon and each slot's item in iconWidth, a slot's count in countWidth.
const (
	iconWidth  = 11
	countWidth = 8
)

// chestRow is the panel slots of the one chest row (index 8 is the icon's).
var chestRow = []int{0, 1, 2, 3, 4, 5, 6, 7, 9}

// slots places the contents on the panel's slots. Chests add up stacks of a
// kind and fill their row in order; other containers keep their slot
// positions.
func (c Container) slots() (out [panelSlots]item.Stack) {
	if c.Layout != LayoutChest {
		var indexes []int
		for i := range panelSlots {
			if i != 8 {
				indexes = append(indexes, i)
			}
		}
		for i, it := range c.Items {
			if i < len(indexes) {
				out[indexes[i]] = it
			}
		}
		if !c.Progress.Empty() {
			out[progressSlot] = c.Progress
		}
		return out
	}
	var kinds []item.Stack
	for _, it := range c.Items {
		if it.Empty() {
			continue
		}
		merged := false
		for k, have := range kinds {
			if have.Comparable(it) {
				kinds[k] = have.Grow(it.Count())
				merged = true
				break
			}
		}
		if !merged {
			kinds = append(kinds, it)
		}
	}
	for i, it := range kinds {
		if i >= len(chestRow) {
			break
		}
		out[chestRow[i]] = it
	}
	return out
}

// empty reports whether the container holds nothing.
func (c Container) empty() bool {
	for _, it := range c.Items {
		if !it.Empty() {
			return false
		}
	}
	return true
}

// itemField is an item's subtitle field: its runtime ID × 65536 (0, air,
// for nothing) left padded with 'x' to iconWidth. The pack removes the 'x's
// and reads the number like the form item icons. Zero padding
// ("-0013107200") and an added offset did not read as items (11th
// playtest).
func itemField(it item.Stack) string {
	n := 0
	if !it.Empty() {
		if rid, _, ok := world.ItemRuntimeID(it.Item()); ok {
			n = int(rid) * 65536
		}
	}
	s := strconv.Itoa(n)
	if len(s) > iconWidth {
		s = "0"
	}
	return strings.Repeat("x", iconWidth-len(s)) + s
}

// countField is a slot's count: "§f" and the number (in thousands from
// 10000), padded with '~' to countWidth bytes; only '~' for one or none.
// It starts with "§f": text that looks like a number becomes a number in
// the pack, and a label shows nothing for a number (the counts did not
// show, 11th playtest).
func countField(n int) string {
	text := ""
	switch {
	case n <= 1:
	case n < 10000:
		text = fmt.Sprintf("§f%d", n)
	default:
		text = fmt.Sprintf("§f%dk", min(n/1000, 999))
	}
	return pad(text, countWidth)
}

// pad cuts s to at most n bytes (whole characters) and fills it with '~'
// to n bytes.
func pad(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r == '~' {
			continue
		}
		if b.Len()+utf8.RuneLen(r) > n {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + strings.Repeat("~", n-b.Len())
}

// token is the title token that shows the container panel.
func (c Container) token() string {
	return "__waila:inv." + string(c.Layout) + "__"
}
