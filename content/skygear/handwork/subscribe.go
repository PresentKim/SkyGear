package handwork

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the handwork steps of the use chain.
func subscribe() {
	subscribeUse()
}

// A stick opens or fills a chest and strips; a bare hand strips fibre and
// tills seeds.
//
// handwork.stick also counts a stick used on any other block, when the click
// reaches it. storage.sneakUse and conduit.plate are in the same tier, are
// registered before it and take such a click first, so it counts nothing:
// that order is one of the exceptions the tier order guard keeps (the root's
// orderKept).
func subscribeUse() {
	core.UseOnBlock.Handle(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) bool {
		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.On(core.Normal, "handwork.gather", func(e core.UseOnBlockEvent) {
		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
		TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
	})
}
