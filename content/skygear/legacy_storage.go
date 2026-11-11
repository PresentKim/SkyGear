package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The drawer displays are removed when the save closes.
func init() {
	core.Close.On(10, "storage.displays", func(e core.Tick) {
		storage.RemoveDisplays(e.S, e.Tx)
	})
}
