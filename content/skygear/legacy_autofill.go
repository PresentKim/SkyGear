package skygear

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/core"
)

// A broken tool and a consumed item are replaced from the inventory.
func init() {
	core.ItemDamage.On(20, "autofill.broken", func(e core.ItemDamageEvent) {
		// a held tool that breaks is replaced by another of its kind
		if held, _ := e.P.HeldItems(); *e.Damage > 0 && e.Item.Durability() >= 0 && *e.Damage >= e.Item.Durability() && sameItemName(held, e.Item) {
			core.RefillHandLater(e.P, e.Tx, e.Item)
		}
	})
	core.ItemConsume.On(20, "autofill.consume", func(e core.ItemEvent) {
		core.RefillHandLater(e.P, e.Tx, e.Item)
	})
}

// sameItemName reports whether a and b are the same item, however worn.
func sameItemName(a, b item.Stack) bool {
	n1, m1 := a.Item().EncodeItem()
	n2, m2 := b.Item().EncodeItem()
	return !a.Empty() && !b.Empty() && n1 == n2 && m1 == m2
}

// The last block of a stack placed is replaced from the inventory.
func init() {
	core.BlockPlace.On(40, "autofill.place", func(e core.BlockPlaceEvent) {
		if e.Held.Count() == 1 {
			core.RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
}

// A thrown snowball, egg or pearl may be the last of the stack.
func init() {
	core.Use.On(10, "autofill.use", func(e core.UseEvent) {
		if e.Held.Count() == 1 {
			core.RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
}
