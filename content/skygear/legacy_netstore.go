package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/netstore"
)

// Disk contents changed since the controller last flushed them are part of the
// save.
func init() {
	core.Autosave.On(10, "netstore.flush", func(e core.Tick) {
		netstore.Disks(e.S).Flush()
	})
	core.Close.On(30, "netstore.flush", func(e core.Tick) {
		netstore.Disks(e.S).Flush()
	})
}
