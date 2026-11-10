package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// RadialItem is a sector of a RadialMenu: its icon and label, the small
// line under the label (the current value of a setting), the line shown
// under the menu while it is chosen, and whether it is the current choice.
// A nil Run leaves it dim.
type RadialItem struct {
	Icon, Label, Value, Info string
	On                       bool
	Run                      func(p *player.Player, tx *world.Tx)
}

// RadialMenu is a ring of choices in the middle of the dimmed game, no
// window: ore-ui-react's RadialMenu with an icon, a label and a value on
// each sector. Title and Line are over the ring, Info under it while no
// sector is chosen. The button in the middle runs Back (nil: it closes the
// menu, as Esc does).
type RadialMenu struct {
	Title, Line, Info string
	Items             []RadialItem
	Back              func(p *player.Player, tx *world.Tx)
	// BackTip is the tooltip of the button in the middle.
	BackTip string
}

// dRadialItem is a sector of the radial screen.
type dRadialItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
	Info  string `json:"info"`
	Icon  string `json:"icon"`
	On    bool   `json:"on"`
}

// radialData is the data of the radial screen; Back is "close" when the
// middle button closes the menu.
type radialData struct {
	Title   string        `json:"title"`
	Line    string        `json:"line"`
	Info    string        `json:"info"`
	Items   []dRadialItem `json:"items"`
	Back    string        `json:"back"`
	BackTip string        `json:"backTip"`
}

// ShowRadial shows p a RadialMenu. The menus of one size switch in place,
// so going into a group and back does not open a new screen.
func ShowRadial(p *player.Player, m RadialMenu) {
	l := newOreLayout()
	l.size = "radial"
	d := radialData{Title: untoken(m.Title), Line: m.Line, Info: m.Info, Items: []dRadialItem{}, Back: "close", BackTip: m.BackTip}
	for _, it := range m.Items {
		d.Items = append(d.Items, dRadialItem{ID: l.id(it.Run), Label: it.Label, Value: it.Value, Info: it.Info, Icon: oreIcon(it.Icon), On: it.On})
	}
	if m.Back != nil {
		d.Back = l.id(m.Back)
	}
	showScreen(p, l, "radial", d)
}
