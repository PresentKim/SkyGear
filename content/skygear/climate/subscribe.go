package climate

import "github.com/presentkim/skygear/content/skygear/core"

// subscribe registers the weather on the second, the join and the start.
func subscribe() {
	subscribeSecond()
	subscribeSession()
}

// The weather advances every second.
func subscribeSecond() {
	core.Second.On(core.Normal, "climate.tick", func(e core.Tick) {
		Tick(e.S, e.Tx)
	})
}

// The weather is shown to a joining player and applied when the session starts.
func subscribeSession() {
	core.Join.On(core.Normal, "climate.showWeather", func(e core.Presence) {
		ShowWeather(e.S, e.P)
	})
	core.Start.On(core.Normal, "climate.apply", func(e core.Tick) {
		Apply(e.S, e.Tx)
	})
}
