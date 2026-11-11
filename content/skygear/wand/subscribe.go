package wand

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the wand steps of the use chain.
func subscribe() {
	subscribeUse()
	followLook()
}

// The construction wand builds out the face clicked and does nothing else;
// used on the air while sneaking it shows its settings.
func subscribeUse() {
	core.UseOnBlock.Handle(80, "wand.use", func(e core.UseOnBlockEvent) bool {
		if _, ok := sgitem.WandOf(e.Held); ok {
			e.Cancel()
			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
			return true
		}
		return false
	})
	core.Use.On(60, "wand.options", func(e core.UseEvent) {
		if _, ok := sgitem.WandOf(e.Held); ok && e.P.Sneaking() && !e.S.Debounced(e.P) {
			ShowOptions(e.S, e.P)
		}
	})
}

// The preview follows where a player holding a wand looks.
func followLook() {
	look.Follow("wand.preview", func(t look.Target) {
		ShowPreview(t.S, t.P, t.Tx)
	})
}
