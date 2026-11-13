package locator

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the locator bar feature, listed in the game's feature list.
var Feature = &core.Feature{Name: "locator", Setup: subscribe}

func subscribe() {
	core.Join.On(core.Normal, "locator.join", func(e core.Presence) {
		if Enabled {
			Join(e.S, e.P)
		}
	})
	core.Leave.On(core.Normal, "locator.leave", func(e core.Presence) {
		if Enabled {
			Leave(e.Tx, e.P)
		}
	})
	core.Second.On(core.Normal, "locator.update", func(e core.Tick) {
		if Enabled {
			Update(e.Tx)
		}
	})
}
