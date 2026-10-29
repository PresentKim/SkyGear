package skygear

import (
	"cmp"
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// chestSortUses sneaking stick uses on the same chest, each less than
// chestSortGap after the last, sort it.
const (
	chestSortUses = 3
	chestSortGap  = time.Second
)

// chestStreak is a player's sneaking stick uses in a row on the chest at pos,
// the last at last.
type chestStreak struct {
	pos  cube.Pos
	uses int
	last time.Time
}

// useStickOnChest puts the items of p's inventory past the hotbar that the
// chest at pos already holds into it, when p uses a stick on it while sneaking; the
// chestSortUses-th use in a row also sorts it. It reports whether the use
// was a sneaking stick use on a chest.
func (s *session) useStickOnChest(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
	if _, ok := held.Item().(item.Stick); !ok || !p.Sneaking() {
		return false
	}
	c, ok := tx.Block(pos).(block.Chest)
	if !ok {
		return false
	}
	if s.stickDebounced(p) {
		return true
	}
	inv := c.Inventory(tx, pos)
	depositMatching(p.Inventory(), inv)
	if s.countChestUse(p, pos) >= chestSortUses {
		delete(s.chestStreaks, p.H())
		sortInventory(inv)
	}
	return true
}

// countChestUse records p's sneaking stick use on the chest at pos and
// returns how many it made on it in a row.
func (s *session) countChestUse(p *player.Player, pos cube.Pos) int {
	if s.chestStreaks == nil {
		s.chestStreaks = map[*world.EntityHandle]chestStreak{}
	}
	cur := s.chestStreaks[p.H()]
	if cur.pos != pos || time.Since(cur.last) >= chestSortGap {
		cur = chestStreak{pos: pos}
	}
	cur.uses++
	cur.last = time.Now()
	s.chestStreaks[p.H()] = cur
	return cur.uses
}

// hotbarSize is the number of player inventory slots, from the first, that
// make up the hotbar.
const hotbarSize = 9

// depositMatching moves the stacks of from past the hotbar that to already
// holds the like of into to, as far as they fit.
func depositMatching(from, to *inventory.Inventory) {
	for slot, it := range from.Slots() {
		if slot < hotbarSize || it.Empty() || !holdsLike(to, it) {
			continue
		}
		if n, _ := to.AddItem(it); n > 0 {
			_ = from.SetItem(slot, it.Grow(-n))
		}
	}
}

// holdsLike reports whether inv holds a stack that it stacks with.
func holdsLike(inv *inventory.Inventory, it item.Stack) bool {
	for _, cur := range inv.Slots() {
		if !cur.Empty() && cur.Comparable(it) {
			return true
		}
	}
	return false
}

// sortInventory merges the stacks of inv that stack together and lays them
// out from the first slot, ordered by item identifier.
func sortInventory(inv *inventory.Inventory) {
	var merged []item.Stack
	for _, it := range inv.Slots() {
		if it.Empty() {
			continue
		}
		if i := slices.IndexFunc(merged, it.Comparable); i >= 0 {
			merged[i] = merged[i].Grow(it.Count())
		} else {
			merged = append(merged, it)
		}
	}
	slices.SortStableFunc(merged, func(a, b item.Stack) int {
		return cmp.Compare(naming.ItemKey(a.Item()), naming.ItemKey(b.Item()))
	})
	var stacks []item.Stack
	for _, it := range merged {
		for left := it.Count(); left > 0; left -= it.MaxCount() {
			stacks = append(stacks, it.Grow(min(left, it.MaxCount())-it.Count()))
		}
	}
	for slot := range inv.Size() {
		var it item.Stack
		if slot < len(stacks) {
			it = stacks[slot]
		}
		_ = inv.SetItem(slot, it)
	}
}
