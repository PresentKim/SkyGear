package skygear

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/screens"
)

// The scoreboards update every second.
func init() {
	core.Second.On(100, "screens.boards", func(e core.Tick) {
		screens.UpdateBoards(e.S, e.Tx)
	})
}
