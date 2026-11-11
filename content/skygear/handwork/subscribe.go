package handwork

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the handwork steps of the use chain.
func subscribe() {
	subscribeUse()
}

// A stick opens or fills a chest and strips; a bare hand strips fibre and
// tills seeds.
func subscribeUse() {
	core.UseOnBlock.Handle(110, "handwork.stick", func(e core.UseOnBlockEvent) bool {
		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.On(130, "handwork.gather", func(e core.UseOnBlockEvent) {
		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
		TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
	})
}
