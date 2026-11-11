package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/handwork"
)

// A stick opens or fills a chest and strips; a bare hand strips fibre and
// tills seeds.
func init() {
	core.UseOnBlock.Handle(110, "handwork.stick", func(e core.UseOnBlockEvent) bool {
		if handwork.UseStickOnChest(e.S, e.P, e.Tx, e.Pos, e.Held) || handwork.UseStick(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.On(130, "handwork.gather", func(e core.UseOnBlockEvent) {
		handwork.StripFiber(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
		handwork.TillSeeds(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
	})
}
