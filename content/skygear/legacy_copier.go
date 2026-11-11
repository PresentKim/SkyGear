package skygear

import (
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The configuration copier copies and pastes settings and does nothing else.
func init() {
	core.UseOnBlock.Handle(70, "copier.use", func(e core.UseOnBlockEvent) bool {
		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
			e.Cancel()
			_, off := e.P.HeldItems()
			copier.UseCopier(e.S, e.P, e.Tx, e.Pos, e.Face, e.Click, e.Held, off)
			return true
		}
		return false
	})
	core.Use.On(50, "copier.clear", func(e core.UseEvent) {
		if _, ok := e.Held.Item().(sgitem.ConfigCopier); ok {
			copier.ClearCopier(e.S, e.P, e.Held, e.Off)
		}
	})
}
