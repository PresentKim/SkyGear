package screens

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// The game menu and the sidebar are these screens; the quest panel follows
// the progress.
func init() {
	core.ShowMenu = ShowMenu
	core.UpdateBoard = UpdateBoard
	progress.OnChange = UpdateBoards
}
