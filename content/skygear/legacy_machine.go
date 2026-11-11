package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The monitor reads its machines every second.
func init() {
	core.Second.On(80, "machine.monitor", func(e core.Tick) {
		machine.TickMonitor(e.S, e.Tx)
	})
}
