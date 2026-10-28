package ui

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Home is the content of HomeScreen.
type Home struct {
	// Buttons are the menu's buttons from the top: an icon, the text and what
	// pressing it does.
	Buttons []Option
	// Footer is the line at the bottom right of the screen.
	Footer string
	// Close runs when the player closes the screen (Esc).
	Close func(p *player.Player, tx *world.Tx)
}

// HomeScreen shows p the lobby's main screen, like a game's title screen: the
// column of large buttons under the lobby's header (lobbySkin) and a footer
// line. It has the size of the windows, so the screens its buttons show take
// its place in the open screen and come back to it the same way, without the
// screen closing in between. It is a DDUI screen only (OreUIFor).
func HomeScreen(p *player.Player, h Home) {
	l := newOreLayout()
	l.size = fmt.Sprintf("%dx%d", windowW, windowH)
	l.close = h.Close
	var buttons []any
	for _, o := range h.Buttons {
		ch := []any{img("mc-home-icon", o.Icon), txt("mc-home-label", o.Label), img("mc-home-chevron", "textures/ui/sg/chevron_right")}
		buttons = append(buttons, l.press(box("mc-btn mc-home-btn", "children", ch), o.Run))
	}
	children := []any{box("mc-home-menu", "children", buttons)}
	if h.Footer != "" {
		children = append(children, txt("mc-home-foot", h.Footer))
	}
	showOre(p, l, box("mc-screen mc-home", "children", children))
}
