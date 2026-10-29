package platform

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
)

// uiDebugCmd shows or hides the debug line of the player's own Ore UI screens
// (common ui.SetOreDebug): the renderer's state and the fonts it found,
// for a screenshot from a device that draws the screens wrong.
type uiDebugCmd struct {
	playerOnly
}

func (uiDebugCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	on := !ui.OreDebug(p)
	ui.SetOreDebug(p, on)
	key := "platform.uidebug.off"
	if on {
		key = "platform.uidebug.on"
	}
	p.Message(i18n.T(i18n.Of(p), key))
}
