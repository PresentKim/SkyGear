package ui

import (
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// PortChoice is one value a SidePort can take: drawn as a square of Colour
// ("#rrggbb"), or as Icon when it has no colour. Run sets it.
type PortChoice struct {
	Colour, Icon, Tip string
	Run               func(p *player.Player, tx *world.Tx)
}

// SidePort is a setting of a side: its label, the choice it has now (an index
// in Choices) and the choices.
type SidePort struct {
	Label   string
	Chosen  int
	Choices []PortChoice
}

// SideView is one side of the block of a SidesMenu: the side's name, the icon
// and tooltip of the block beside it and the ports set on the side (none:
// nothing to set there).
type SideView struct {
	Name, Icon, Tip string
	Ports           []SidePort
}

// SideBoard is the content of SidesMenu: the icon of the block in the middle,
// its six sides by cube.Face (down, up, north, south, west, east) and the side
// drawn as chosen (Focus, -1: none). Refresh builds the board again after a
// choice's Run changed it.
type SideBoard struct {
	Center  string
	Sides   [6]SideView
	Focus   int
	Refresh func(p *player.Player, tx *world.Tx) SideBoard
}

// The layout of SidesMenu in GUI pixels: the view is sidesW by sidesH, cards
// cardW by cardH in three columns cardGap apart, the block centreSize square,
// the choices panel pickerCols choices of choiceSize to a row.
const (
	sidesW, sidesH         = 310, 144
	cardW, cardH, cardGap  = 96, 56, 11
	centreSize             = 20
	pickerCols, choiceSize = 6, 12
	pickerPad              = 3
)

// sideCard is where the card of side f is: its top left corner in the view,
// and whether it is on the bottom row (its choices open above it). Up and
// down are in the middle column over and under the block, west and east at the
// top of the outer columns, north and south at their bottom.
func sideCard(f cube.Face) (x, y int, low bool) {
	left, mid, right := 0, cardW+cardGap, 2*(cardW+cardGap)
	top, bottom := 0, sidesH-cardH
	switch f {
	case cube.FaceUp:
		return mid, top, false
	case cube.FaceDown:
		return mid, bottom, true
	case cube.FaceWest:
		return left, top, false
	case cube.FaceEast:
		return right, top, false
	case cube.FaceNorth:
		return left, bottom, true
	}
	return right, bottom, true
}

// sideArrow is the arrow beside the name of side f.
func sideArrow(f cube.Face) string {
	switch f {
	case cube.FaceWest:
		return "←"
	case cube.FaceEast:
		return "→"
	case cube.FaceUp, cube.FaceNorth:
		return "↑"
	}
	return "↓"
}

// sidePicker is the port whose choices are open: the side and the port.
type sidePicker struct{ side, port int }

// SidesMenu shows a block with a card for each of its six sides: the
// neighbouring block and a box per port that cycles its choice when pressed
// (back with the right mouse button),
// with a plus button that opens all choices beside the card. A press shows
// the whole screen again.
func SidesMenu(p *player.Player, title string, b SideBoard, actions ...Option) {
	updateSides(p, title, b, nil, actions)
}

// sidesTurn counts the redrawn ports of SidesMenu screens, so their ids do
// not reuse the ids of the ports before them.
var sidesTurn int

// sidesData is the data of the sides screen.
type sidesData struct {
	Chrome dChrome     `json:"chrome"`
	Center string      `json:"center"`
	Cards  []dSideCard `json:"cards"`
	Picker *dPicker    `json:"picker"`
}

// dSideCard is the card of a side, at x, y in GUI pixels.
type dSideCard struct {
	X     int     `json:"x"`
	Y     int     `json:"y"`
	Arrow string  `json:"arrow"`
	Name  string  `json:"name"`
	Icon  string  `json:"icon"`
	Tip   string  `json:"tip"`
	Focus bool    `json:"focus"`
	Ports []dPort `json:"ports"`
	None  string  `json:"none"`
}

// dPort is a port of a side: its swatch (nil: no choices) and plus button.
type dPort struct {
	Label  string   `json:"label"`
	Swatch *dSwatch `json:"swatch"`
	Plus   dPlus    `json:"plus"`
}

// dSwatch is the current choice of a port; ID presses to the next choice and
// Alt to the one before.
type dSwatch struct {
	ID     string `json:"id"`
	Alt    string `json:"alt"`
	Colour string `json:"colour"`
	Icon   string `json:"icon"`
	Tip    string `json:"tip"`
}

// dPlus is the plus button of a port; On when its choices are open.
type dPlus struct {
	ID string `json:"id"`
	On bool   `json:"on"`
}

// dPicker is the panel of the open choices of a port.
type dPicker struct {
	X       int           `json:"x"`
	Y       int           `json:"y"`
	W       int           `json:"w"`
	Choices []dPickChoice `json:"choices"`
}

// dPickChoice is a choice in a dPicker.
type dPickChoice struct {
	ID     string `json:"id"`
	Colour string `json:"colour"`
	Icon   string `json:"icon"`
	Tip    string `json:"tip"`
	On     bool   `json:"on"`
}

// updateSides shows the whole sides screen of b to p, with the choices of
// open (nil: none) beside its card, keeping the screen open after the press
// that changed it.
func updateSides(p *player.Player, title string, b SideBoard, open *sidePicker, actions []Option) {
	oreScreens.Lock()
	sidesTurn++
	l := newOreLayout()
	l.prefix = "s" + strconv.Itoa(sidesTurn) + "_"
	oreScreens.Unlock()
	chrome, _ := l.chromeOf(title, actions)
	d := sidesData{Chrome: chrome, Center: oreIcon(b.Center), Cards: make([]dSideCard, 0, len(b.Sides))}
	again := func(b SideBoard, open *sidePicker) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { updateSides(p, title, b, open, actions) }
	}
	choose := func(run func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if run != nil {
				run(p, tx)
			}
			nb := b
			if b.Refresh != nil {
				nb = b.Refresh(p, tx)
			}
			updateSides(p, title, nb, nil, actions)
		}
	}
	for i, s := range b.Sides {
		x, y, _ := sideCard(cube.Face(i))
		c := dSideCard{X: x, Y: y, Arrow: sideArrow(cube.Face(i)), Name: s.Name, Icon: oreIcon(s.Icon), Tip: s.Tip,
			Focus: i == b.Focus, Ports: []dPort{}}
		if s.Icon == "" {
			c.Tip = ""
		}
		if len(s.Ports) == 0 {
			c.None = i18n.T(i18n.Of(p), "ui.sides.none")
		}
		for j, port := range s.Ports {
			dp := dPort{Label: port.Label}
			if n := len(port.Choices); n > 0 {
				at := min(max(port.Chosen, 0), n-1)
				cur := port.Choices[at]
				dp.Swatch = &dSwatch{
					ID:     l.id(choose(port.Choices[(at+1)%n].Run)),
					Alt:    l.id(choose(port.Choices[(at+n-1)%n].Run)),
					Colour: cur.Colour, Icon: oreIcon(cur.Icon), Tip: cur.Tip,
				}
			}
			here := sidePicker{i, j}
			if open != nil && *open == here {
				dp.Plus = dPlus{ID: l.id(again(b, nil)), On: true}
				d.Picker = pickerOf(l, i, port, choose)
			} else {
				dp.Plus = dPlus{ID: l.id(again(b, &here))}
			}
			c.Ports = append(c.Ports, dp)
		}
		d.Cards = append(d.Cards, c)
	}
	showScreen(p, l, "sides", d)
}

// pickerOf is the panel of the choices of port, on the card of side: under
// the card, or over it for a card on the bottom row. choose makes the run of
// a choice from the choice's Run.
func pickerOf(l *oreLayout, side int, port SidePort, choose func(func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx)) *dPicker {
	choices := make([]dPickChoice, 0, len(port.Choices))
	for k, c := range port.Choices {
		choices = append(choices, dPickChoice{ID: l.id(choose(c.Run)), Colour: c.Colour, Icon: oreIcon(c.Icon), Tip: c.Tip, On: k == port.Chosen})
	}
	rows := (len(port.Choices) + pickerCols - 1) / pickerCols
	w, h := pickerCols*choiceSize+2*pickerPad, rows*choiceSize+2*pickerPad
	x, y, low := sideCard(cube.Face(side))
	x = min(x+cardW-w, sidesW-w)
	if low {
		y -= h + 2
	} else {
		y += cardH + 2
	}
	return &dPicker{X: x, Y: y, W: w, Choices: choices}
}
