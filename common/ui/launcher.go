package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// LauncherToken is the title of the lobby's main menu (packs/dfui
// launcher_form): no window, a column of large buttons at the top left of the
// screen.
const LauncherToken = "§d§f§n§r"

// LauncherMenu shows the lobby's main menu: a column of large buttons, each
// an icon (a texture path), a label and a chevron. A label is a key of the
// server pack's language files: the pack localizes it, and the HUD copy of the
// menu (dfui_hooks.lobby_menu) draws the same keys, so the screen does not
// change while the form is away. onClose, if not nil, runs when the player closes the
// form without a press.
// Labels must be unique.
func LauncherMenu(p *player.Player, onClose func(p *player.Player, tx *world.Tx), opts ...Option) {
	sendMenu(p, LauncherToken, "", onClose, opts)
}
