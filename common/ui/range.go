package ui

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/invui"
)

// RangeAxis is an axis of a RangeScreen: its label and value, and the
// buttons that take one off and add one (a nil Run leaves the button dim).
// An axis without a label is left out.
type RangeAxis struct {
	Label, Value string
	Minus, Plus  Option
}

// RangeScreen sets an area: the x, y and z of its offset and of its reach
// under their headings, a 3D preview of it and more buttons.
type RangeScreen struct {
	Title                       string
	OffsetHeading, ReachHeading string
	Offset, Reach               [3]RangeAxis
	// Preview is the preview box that shows the area, 1 to
	// len(invui.RangePreviews) (the area is copied there for the player);
	// 0 shows none.
	Preview int
	// Options are tabs and control bar buttons where they go, the rest as
	// list rows.
	Options []Option
	// Close runs when the player closes the screen.
	Close func(p *player.Player, tx *world.Tx)
}

type dRangeAxis struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Minus dButton `json:"minus"`
	Plus  dButton `json:"plus"`
}

// rangeData is the data of the range screen: the offset's axes then the
// reach's.
type rangeData struct {
	Chrome  dChrome      `json:"chrome"`
	Heads   [2]string    `json:"heads"`
	Axes    []dRangeAxis `json:"axes"`
	Preview int          `json:"preview"`
	Items   []dItem      `json:"items"`
}

// ShowRange shows p a RangeScreen.
func ShowRange(p *player.Player, r RangeScreen) {
	l := newOreLayout()
	l.close = r.Close
	chrome, rest := l.chromeOf(r.Title, r.Options)
	d := rangeData{Chrome: chrome, Heads: [2]string{r.OffsetHeading, r.ReachHeading}, Preview: r.Preview, Items: l.itemsOf(rest)}
	for _, ax := range append(r.Offset[:], r.Reach[:]...) {
		d.Axes = append(d.Axes, dRangeAxis{Label: ax.Label, Value: ax.Value, Minus: l.buttonOf(ax.Minus), Plus: l.buttonOf(ax.Plus)})
	}
	showScreen(p, l, "range", d)
}

func rangePage(v rangeData) *menuPage {
	pg := newMenuPage(invui.KeyRange)
	pg.chrome(v.Chrome)
	for i, ax := range v.Axes {
		if i >= invui.RangeAxes || ax.Label == "" {
			continue
		}
		pg.text(invui.Cell0+i, "§r"+ink(ax.Label+"§r  "+ax.Value))
		step := func(slot int, b dButton) {
			kind := invui.KindRow
			if b.ID == "" {
				kind = invui.KindStatic
			}
			pg.button(slot, "", kind, b.ID)
		}
		step(invui.RowBtn0+2*i, ax.Minus)
		step(invui.RowBtn0+2*i+1, ax.Plus)
	}
	for i, h := range v.Heads {
		if h != "" {
			pg.text(invui.Cell0+invui.RangeHead0+i, "§r"+ink(h))
		}
	}
	if v.Preview > 0 && v.Preview <= len(invui.RangePreviews) {
		pg.text(invui.Cell0+invui.RangeView, fmt.Sprintf("v%d", v.Preview))
	}
	buttons := make([]dButton, 0, len(v.Items))
	for _, it := range v.Items {
		buttons = append(buttons, it.dButton)
	}
	pg.buttons(invui.RangeRow0, invui.KindRow, buttons)
	return pg
}
