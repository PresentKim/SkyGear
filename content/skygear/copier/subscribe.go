package copier

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the copier steps of the use chain.
func subscribe() {
	subscribeUse()
}

// The configuration copier copies and pastes settings and does nothing else.
// Where the island does not allow building it leaves the click alone: the
// island's guard refuses it. A held tool takes the click away from what the
// block itself would do with it: the Low tier.
func subscribeUse() {
	core.UseOnBlock.Handle(core.Low, "copier.use", func(e core.UseOnBlockEvent) bool {
		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok && island.CanBuild(e.S, e.Pos) {
			e.Cancel()
			_, off := e.P.HeldItems()
			UseCopier(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held, off)
			return true
		}
		return false
	})
	core.Use.On(core.Normal, "copier.clear", func(e core.UseEvent) {
		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
			ClearCopier(e.S, e.P, e.Held, e.Off)
		}
	})
}
