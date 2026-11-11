package skygear

import (
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The weather advances every second.
func init() {
	core.Second.On(50, "climate.tick", func(e core.Tick) {
		climate.Tick(e.S, e.Tx)
	})
}

// The weather is shown to a joining player and applied when the session starts.
func init() {
	core.Join.On(30, "climate.showWeather", func(e core.Presence) {
		climate.ShowWeather(e.S, e.P)
	})
	core.Start.On(20, "climate.apply", func(e core.Tick) {
		climate.Apply(e.S, e.Tx)
	})
}
