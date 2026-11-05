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

// SideBoard is the content of SidesMenu: its six sides by cube.Face (down,
// up, north, south, west, east), the side drawn as chosen (Focus, -1: none)
// and the guide line under the cards. Refresh builds the board again after a
// choice's Run changed it.
type SideBoard struct {
	Sides   [6]SideView
	Focus   int
	Guide   string
	Refresh func(p *player.Player, tx *world.Tx) SideBoard
}

// sideOrder is the order of the cards, three to a row: west, up and east on
// the top row, north, down and south under them.
var sideOrder = [6]cube.Face{cube.FaceWest, cube.FaceUp, cube.FaceEast, cube.FaceNorth, cube.FaceDown, cube.FaceSouth}

// sidePicker is the port whose choices are open: the side and the port.
type sidePicker struct{ side, port int }

// SidesMenu shows a card for each of the six sides of a block, in sideOrder:
// the neighbouring block and a square per port that cycles its choice when
// pressed (back with the right mouse button), with a plus button that opens
// all choices in a panel. A press shows the whole screen again.
func SidesMenu(p *player.Player, title string, b SideBoard, actions ...Option) {
	updateSides(p, title, b, nil, actions)
}

// sidesTurn counts the redrawn ports of SidesMenu screens, so their ids do
// not reuse the ids of the ports before them.
var sidesTurn int

// sidesData is the data of the sides screen: the cards in sideOrder.
type sidesData struct {
	Chrome dChrome     `json:"chrome"`
	Cards  []dSideCard `json:"cards"`
	Guide  string      `json:"guide"`
	Picker *dPicker    `json:"picker"`
}

// dSideCard is the card of a side.
type dSideCard struct {
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
	d := sidesData{Chrome: chrome, Guide: b.Guide, Cards: make([]dSideCard, 0, len(b.Sides))}
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
	for _, f := range sideOrder {
		i, s := int(f), b.Sides[f]
		c := dSideCard{Name: s.Name, Icon: oreIcon(s.Icon), Tip: s.Tip, Focus: i == b.Focus, Ports: []dPort{}}
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
				d.Picker = pickerOf(l, port, choose)
			} else {
				dp.Plus = dPlus{ID: l.id(again(b, &here))}
			}
			c.Ports = append(c.Ports, dp)
		}
		d.Cards = append(d.Cards, c)
	}
	showScreen(p, l, "sides", d)
}

// pickerOf is the panel of the choices of port. choose makes the run of a
// choice from the choice's Run.
func pickerOf(l *oreLayout, port SidePort, choose func(func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx)) *dPicker {
	choices := make([]dPickChoice, 0, len(port.Choices))
	for k, c := range port.Choices {
		choices = append(choices, dPickChoice{ID: l.id(choose(c.Run)), Colour: c.Colour, Icon: oreIcon(c.Icon), Tip: c.Tip, On: k == port.Chosen})
	}
	return &dPicker{Choices: choices}
}
