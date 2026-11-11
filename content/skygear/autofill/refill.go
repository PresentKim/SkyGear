// Package autofill refills a hand whose stack ran out.
package autofill

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/platform/account"
)

// Autofill (like the Mouse Wheelie mod's refill): when the stack in a
// player's hand runs out (the last block placed, the last food eaten, the
// tool broken), the same item comes into the hand from the rest of the
// inventory. Every way the game uses up a held stack goes through here: the
// player's own placing, eating and tool wear (the handler), the sieve and
// the construction wand.

// Setting is the player setting (account.Setting) that turns
// autofill on.
const Setting = "autofill"

// On reports whether p turned autofill on.
func On(p *player.Player) bool { return account.On(p, Setting) }

// sameKind reports whether st can stand in for proto: the same stack, or for
// an item that wears, the same item however worn.
func sameKind(st, proto item.Stack) bool {
	if st.Empty() || proto.Empty() {
		return false
	}
	if _, durable := proto.Item().(item.Durable); durable {
		n1, m1 := st.Item().EncodeItem()
		n2, m2 := proto.Item().EncodeItem()
		return n1 == n2 && m1 == m2
	}
	return st.Comparable(proto)
}

// refillSlot is the inventory slot autofill takes proto's kind from: the
// main inventory before the hotbar, so the hotbar keeps its layout. The
// hand is empty when this runs, so its own slot never matches.
func refillSlot(p *player.Player, proto item.Stack) (int, bool) {
	best := -1
	for slot, st := range p.Inventory().Slots() {
		if !sameKind(st, proto) {
			continue
		}
		if slot >= 9 {
			return slot, true
		}
		if best < 0 {
			best = slot
		}
	}
	return best, best >= 0
}

// RefillHand fills p's main hand, if it is empty, with a stack of proto's
// kind from the rest of the inventory, when p turned autofill on. It reports
// whether it did.
func RefillHand(p *player.Player, proto item.Stack) bool {
	held, off := p.HeldItems()
	if !held.Empty() || !On(p) {
		return false
	}
	slot, ok := refillSlot(p, proto)
	if !ok {
		return false
	}
	st, _ := p.Inventory().Item(slot)
	_ = p.Inventory().SetItem(slot, item.Stack{})
	p.SetHeldItems(st, off)
	return true
}

// RefillHandLater runs RefillHand once the game has used up what it is about
// to (a placed block or eaten food is taken from the hand after the handler
// returns).
func RefillHandLater(p *player.Player, tx *world.Tx, proto item.Stack) {
	if proto.Empty() || !On(p) {
		return
	}
	h := p.H()
	tx.Defer(func(tx *world.Tx) {
		if e, ok := h.Entity(tx); ok {
			if p, ok := e.(*player.Player); ok {
				RefillHand(p, proto)
			}
		}
	})
}
