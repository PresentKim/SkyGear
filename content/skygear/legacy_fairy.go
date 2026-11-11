package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/fairy"
)

// The fairy acts every second.
func init() {
	core.Second.On(40, "fairy.tick", func(e core.Tick) {
		fairy.Tick(e.S, e.Tx)
	})
}
