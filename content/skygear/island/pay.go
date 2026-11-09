package island

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Chunks cost cobblestone, and compressed cobblestone pays as the
// cobblestone it is worth: plain cobblestone first, then the compressed
// blocks from the lowest level. A block broken for the payment gives the
// difference back in cobblestone and lower compressed blocks
// (sgitem.PlanCompactPayment). Change that does not fit in the inventory
// goes to the lost-and-found box, as the buyer is looking at the expansion
// map and items dropped at their feet could be missed or lost.

// compactLevels are the items of base (level 0) and its compressed blocks
// (1 to CompactMaxLevel; nil above the base's own levels).
func compactLevels(base sgitem.CompactBase) [sgitem.CompactMaxLevel + 1]world.Item {
	var out [sgitem.CompactMaxLevel + 1]world.Item
	out[0] = base.Block
	for l := 1; l <= base.Levels; l++ {
		out[l] = sgitem.CompactBlock{Kind: base.Kind, Level: l}
	}
	return out
}

// compactBaseFor is the compressing base whose base block is it.
func compactBaseFor(it world.Item) (sgitem.CompactBase, bool) {
	k := naming.ItemKey(it)
	for _, b := range sgitem.CompactBases {
		if naming.ItemKey(b.Block) == k {
			return b, true
		}
	}
	return sgitem.CompactBase{}, false
}

// held counts the plain items of each level in inv.
func held(inv *inventory.Inventory, levels [sgitem.CompactMaxLevel + 1]world.Item) [sgitem.CompactMaxLevel + 1]int {
	var have [sgitem.CompactMaxLevel + 1]int
	for l, it := range levels {
		if it != nil {
			have[l] = core.CountPlain(inv, it)
		}
	}
	return have
}

// Worth is how many of it inv holds, counting the compressed blocks of it
// as what they are worth; compressed is the part of that in compressed
// blocks.
func Worth(inv *inventory.Inventory, it world.Item) (total, compressed int) {
	base, ok := compactBaseFor(it)
	if !ok {
		return core.CountPlain(inv, it), 0
	}
	have := held(inv, compactLevels(base))
	for l, n := range have {
		total += n * sgitem.CompactWorth(l)
	}
	return total, total - have[0]
}

// Pay takes count of it from inv, compressed blocks of it counting as
// what they are worth. The change of a broken compressed block goes into
// inv; what does not fit is returned (nothing is lost: the caller keeps
// it). ok is false, and nothing is taken, when inv holds too little.
func Pay(inv *inventory.Inventory, it world.Item, count int) (left []item.Stack, ok bool) {
	base, compact := compactBaseFor(it)
	if !compact {
		return nil, core.TakePlain(inv, it, count)
	}
	levels := compactLevels(base)
	plan, ok := sgitem.PlanCompactPayment(held(inv, levels), count)
	if !ok {
		return nil, false
	}
	net := plan.Net()
	// Take first, so the change has the freed slots.
	for l, n := range net {
		if n < 0 {
			core.TakePlain(inv, levels[l], -n)
		}
	}
	for l := len(net) - 1; l >= 0; l-- {
		for n := net[l]; n > 0; {
			st := item.NewStack(levels[l], min(n, 64))
			n -= st.Count()
			if added, _ := inv.AddItem(st); added < st.Count() {
				left = append(left, st.Grow(-added))
			}
		}
	}
	return left, true
}
