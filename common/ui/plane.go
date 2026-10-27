package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// PlaneSpot is a button over a place of the ground a straight-down camera
// shows: its rectangle in blocks from the point under the camera (X east, Z
// south, from the north-west corner), whether it is highlighted, its tooltip
// and what pressing it does.
type PlaneSpot struct {
	X, Z, W, H float64
	On         bool
	Tip        string
	Run        func(p *player.Player, tx *world.Tx)
}

// Plane is the content of PlaneMenu.
type Plane struct {
	// FOV is the camera's vertical field of view in degrees and Distance its
	// height over the ground, in blocks: the renderer works out from them
	// where each spot is on the screen.
	FOV, Distance float64
	Spots         []PlaneSpot
	// Info is the text of the panel at the bottom; Actions its buttons.
	Info    string
	Actions []Option
	// Corner is the text of a panel at the top left (none: no panel).
	Corner string
	// Close runs when the player closes the screen (Esc).
	Close func(p *player.Player, tx *world.Tx)
	// Keys are what the keys of the keyboard do (none: a key moves the
	// keyboard selection).
	Keys map[Key]func(p *player.Player, tx *world.Tx)
}

// PlaneMenu shows buttons over the ground the player's camera looks straight
// down on, each the size of its place on the screen, with a panel of text and
// buttons at the bottom; the screen is not darkened, so the world shows. It
// is a DDUI screen only (OreUIFor). A spot or button that shows PlaneMenu
// again updates the screen open.
func PlaneMenu(p *player.Player, v Plane) {
	l := newOreLayout()
	l.size = "plane"
	l.close = v.Close
	for _, k := range keyOrder {
		l.key(k, v.Keys[k])
	}
	var spots []any
	for _, sp := range v.Spots {
		n := box("mc-spot", "on", sp.On, "at", []float64{sp.X, sp.Z, sp.W, sp.H})
		if sp.Tip != "" {
			n["tip"] = sp.Tip
			quick(n)
		}
		spots = append(spots, l.press(n, sp.Run))
	}
	var buttons []any
	for _, a := range v.Actions {
		buttons = append(buttons, l.actionButton(a))
	}
	panel := box("mc-plane-panel", "ink", "dark", "children", append([]any{txt("mc-plane-info", v.Info)}, buttons...))
	children := []any{node("plane", "fov", v.FOV, "dist", v.Distance, "children", spots), panel}
	if v.Corner != "" {
		children = append(children, box("mc-plane-panel mc-plane-corner", "ink", "dark", "children", []any{txt("mc-plane-info", v.Corner)}))
	}
	showOre(p, l, box("mc-screen mc-plane-screen", "children", children))
}
