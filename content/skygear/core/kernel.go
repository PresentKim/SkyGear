package core

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/platform/account"
)

// KernelFeature is the kernel's own feature (the shapes and parts steps of
// the lifecycle); the game lists it first.
var KernelFeature = &Feature{Name: "core", Setup: subscribeKernel}

// AutofillFeature is the autofill feature (the refill of a used-up hand
// stack); the game lists it after the features whose steps it follows.
var AutofillFeature = &Feature{
	Name: "autofill",
	Settings: []account.Setting{
		{Key: AutofillSetting, NameKey: "setting.autofill", Default: true, Group: "setting.group.play"},
	},
	Setup: subscribeAutofill,
}

// Shapes are sent when a player joins and cleared, client-side, before the
// player leaves the instance world; the core session drops the player's
// per-player parts.
func subscribeKernel() {
	Join.On(40, "core.shapes", func(e Presence) {
		e.S.SendShapes(e.P)
	})
	Leave.On(20, "core.parts", func(e Presence) {
		e.S.Leave(e.P)
	})
	Leave.On(50, "core.shapes", func(e Presence) {
		e.P.RemoveAllDebugShapes()
	})
}

// A broken tool and a consumed item are replaced from the inventory, the last
// block of a stack placed is replaced, and a thrown snowball, egg or pearl may
// be the last of the stack.
func subscribeAutofill() {
	ItemDamage.On(20, "autofill.broken", func(e ItemDamageEvent) {
		// a held tool that breaks is replaced by another of its kind
		if held, _ := e.P.HeldItems(); *e.Damage > 0 && e.Item.Durability() >= 0 && *e.Damage >= e.Item.Durability() && sameItemName(held, e.Item) {
			RefillHandLater(e.P, e.Tx, e.Item)
		}
	})
	ItemConsume.On(20, "autofill.consume", func(e ItemEvent) {
		RefillHandLater(e.P, e.Tx, e.Item)
	})
	BlockPlace.On(40, "autofill.place", func(e BlockPlaceEvent) {
		if e.Held.Count() == 1 {
			RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
	Use.On(10, "autofill.use", func(e UseEvent) {
		if e.Held.Count() == 1 {
			RefillHandLater(e.P, e.Tx, e.Held)
		}
	})
}

// sameItemName reports whether a and b are the same item, however worn.
func sameItemName(a, b item.Stack) bool {
	n1, m1 := a.Item().EncodeItem()
	n2, m2 := b.Item().EncodeItem()
	return !a.Empty() && !b.Empty() && n1 == n2 && m1 == m2
}
