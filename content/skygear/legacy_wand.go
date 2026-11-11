package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// The construction wand builds out the face clicked and does nothing else;
// used on the air while sneaking it shows its settings.
func init() {
	core.UseOnBlock.Handle(80, "wand.use", func(e core.UseOnBlockEvent) bool {
		if _, ok := sgitem.WandOf(e.Held); ok {
			e.Cancel()
			wand.Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
			return true
		}
		return false
	})
	core.Use.On(60, "wand.options", func(e core.UseEvent) {
		if _, ok := sgitem.WandOf(e.Held); ok && e.P.Sneaking() && !e.S.Debounced(e.P) {
			wand.ShowOptions(e.S, e.P)
		}
	})
}
