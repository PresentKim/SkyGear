package wand

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the wand steps of the use chain.
func subscribe() {
	subscribeUse()
	followLook()
}

// The construction wand builds out the face clicked and does nothing else.
// Where the island does not allow building it leaves the click alone: the
// island's guard refuses it. A held tool takes the click away from what the
// block itself would do with it: the Low tier. Used on the air while sneaking
// the wand shows its settings: no block is asked first there, so that is the
// item's own Use (sgitem.Wand, sgitem.InfinityWand), which Dragonfly calls
// unless a subscriber cancelled the use.
func subscribeUse() {
	core.UseOnBlock.On(core.Low, "wand.use", func(e core.UseOnBlockEvent) {
		if _, ok := sgitem.WandOf(e.Held); ok && island.CanBuild(e.S, e.Pos) {
			e.Cancel()
			Use(e.S, e.P, e.Tx, e.Pos, e.Face, e.Held)
		}
	})
	sgitem.WandUsed = func(tx *world.Tx, u item.User) {
		p, ok := u.(*player.Player)
		s, ok2 := core.Of(tx)
		if ok && ok2 && p.Sneaking() && !s.Debounced(p) {
			ShowOptions(s, p)
		}
	}
}

// The preview follows where a player holding a wand looks.
func followLook() {
	look.Follow("wand.preview", func(t look.Target) {
		ShowPreview(t.S, t.P, t.Tx)
	})
}
