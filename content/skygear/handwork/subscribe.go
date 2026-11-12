package handwork

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the handwork steps of the use chain.
func subscribe() {
	subscribeUse()
}

// A stick fills a chest (sneaking) and knocks pebbles out of grass and dirt;
// a bare hand strips fibre and tills seeds. A stick used on any other block
// is recorded when the click went through (nothing cancelled it): the record
// handles the result, Monitor.
func subscribeUse() {
	core.UseOnBlock.On(core.Normal, "handwork.stick", func(e core.UseOnBlockEvent) {
		if UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
			e.Cancel()
		}
	})
	core.UseOnBlock.On(core.Normal, "handwork.gather", func(e core.UseOnBlockEvent) {
		StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
		TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
	})
	core.UseOnBlock.On(core.Monitor, "handwork.stickOther", func(e core.UseOnBlockEvent) {
		CountStickOther(e.S, e.P, e.Tx, e.Pos, e.Held)
	})
}
