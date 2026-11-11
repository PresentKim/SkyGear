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
