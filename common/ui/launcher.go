package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// LauncherToken is the title of the lobby's main menu: no window, a column
// of large buttons at the top left of the screen.
const LauncherToken = "§d§f§n§r"

// LauncherMenu shows the lobby's main menu: large buttons with an icon, a
// label and a chevron. Labels are unique language keys of the server pack,
// which the HUD copy (dfui_hooks.lobby_menu) also draws, so the screen does
// not change while the form is away. onClose, if set, runs on close without
// a press.
func LauncherMenu(p *player.Player, onClose func(p *player.Player, tx *world.Tx), opts ...Option) {
	sendMenu(p, LauncherToken, "", onClose, len(opts), opts)
}
