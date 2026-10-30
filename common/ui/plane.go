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
// buttons at the bottom; the screen is not darkened, so the world shows. A
// spot or button that shows PlaneMenu again updates the screen open.
func PlaneMenu(p *player.Player, v Plane) {
	l := newOreLayout()
	l.size = "plane"
	l.close = v.Close
	for _, k := range keyOrder {
		l.key(k, v.Keys[k])
	}
	d := planeData{FOV: v.FOV, Dist: v.Distance, Spots: make([]dSpot, 0, len(v.Spots)), Info: v.Info, Corner: v.Corner}
	for _, sp := range v.Spots {
		d.Spots = append(d.Spots, dSpot{ID: l.id(sp.Run), At: [4]float64{sp.X, sp.Z, sp.W, sp.H}, On: sp.On, Tip: sp.Tip})
	}
	d.Actions = l.buttonsOf(v.Actions)
	showScreen(p, l, "plane", d)
}

// planeData is the data of the plane screen.
type planeData struct {
	FOV     float64   `json:"fov"`
	Dist    float64   `json:"dist"`
	Spots   []dSpot   `json:"spots"`
	Info    string    `json:"info"`
	Actions []dButton `json:"actions"`
	Corner  string    `json:"corner"`
}

// dSpot is a button over a place of the ground: its rectangle in blocks
// (x, z, width, height).
type dSpot struct {
	ID  string     `json:"id"`
	At  [4]float64 `json:"at"`
	On  bool       `json:"on"`
	Tip string     `json:"tip"`
}
