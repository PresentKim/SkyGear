package intro

import "github.com/presentkim/skygear/content/skygear/core"

// Feature is the first-entry camera, listed in the game's feature list.
var Feature = &core.Feature{Name: "intro", Setup: subscribe}

func subscribe() {
	core.Join.On(core.Normal, "intro.firstEntry", func(e core.Presence) {
		if e.First {
			Play(e.P)
		}
	})
}
