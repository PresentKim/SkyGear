package ui

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// PortChoice is one value a SidePort can take: drawn as a square of Colour
// ("#rrggbb") in DDUI, or as Icon when it has no colour and in JSON UI. Run
// sets it.
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

// The layout of SidesMenu in GUI pixels, in the window (windowW by
// windowH): the view is sidesW by
// sidesH, a card cardW by cardH in three columns cardGap apart, the block in
// the middle centreSize square, the choices panel pickerCols choices of
// choiceSize to a row.
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
		return "textures/ui/df/arrow_left"
	case cube.FaceEast:
		return "textures/ui/df/arrow_right"
	case cube.FaceUp, cube.FaceNorth:
		return "textures/ui/df/arrow_up"
	}
	return "textures/ui/df/arrow_down"
}

// sidePicker is the port whose choices are open: the side and the port.
type sidePicker struct{ side, port int }

// SidesMenu shows a block in the middle with a card for each of its six sides
// around it, joined to it by a line, each card with the arrow and name of the
// side, the block beside it and a box for each port side by side: the square
// of its choice, which turns it to the next choice when pressed, its label
// and a plus button that opens the choices in a panel beside the card. A
// change redraws only the ports. A side without ports shows a grey box in
// their place. Without DDUI it is an ItemBoard with a row for each side, and
// the choices are an ItemGrid.
func SidesMenu(p *player.Player, title string, b SideBoard, actions ...Option) {
	if !OreUIFor(p) {
		ItemBoard(p, title, "", sideRows(title, b, actions), actions...)
		return
	}
	l := newOreLayout()
	tabs, bar, _ := l.chrome(actions, false)
	cx, cy := sidesW/2, sidesH/2
	pos := func(x, y int) map[string]any { return map[string]any{"left": gu(x), "top": gu(y)} }
	var view []any
	for i := range b.Sides {
		x, y, _ := sideCard(cube.Face(i))
		view = append(view, node("line", "class", "oui-line-seg mc-side-line",
			"x1", float64(cx)*0.2, "y1", float64(cy)*0.2, "x2", float64(x+cardW/2)*0.2, "y2", float64(y+cardH/2)*0.2))
	}
	view = append(view, box("mc-side-centre", "style", pos(cx-centreSize/2, cy-centreSize/2), "children", []any{img("", b.Center)}))
	for i, s := range b.Sides {
		x, y, _ := sideCard(cube.Face(i))
		class := "mc-side-card"
		if i == b.Focus {
			class += " on"
		}
		block := img("mc-side-block", s.Icon)
		block["tip"] = s.Tip
		view = append(view, box(class, "style", pos(x, y), "children", []any{
			box("mc-side-head", "children", []any{img("mc-side-arrow", sideArrow(cube.Face(i))), txt("mc-side-name", s.Name)}),
			box("mc-side-body", "children", []any{block, node("region", "name", sideRegion(i))}),
		}))
	}
	view = append(view, node("region", "name", "picker"))
	content := box("mc-sides", "style", map[string]any{"width": gu(sidesW), "height": gu(sidesH)}, "children", view)
	showOre(p, l, l.window(title, tabs, bar, content))
	updateSides(p, b, nil)
}

func sideRegion(side int) string { return "side" + strconv.Itoa(side) }

// sidesTurn counts the redrawn ports of SidesMenu screens, so their ids do
// not reuse the ids of the ports before them.
var sidesTurn int

// choiceNode is the square of a choice: its colour, or its icon.
func choiceNode(class string, c PortChoice) map[string]any {
	if c.Colour != "" {
		return box(class, "style", map[string]any{"backgroundColor": c.Colour})
	}
	return box(class+" icon", "children", []any{img("", c.Icon)})
}

// updateSides draws the ports of the sides of b into the screen p has open,
// and the choices of open (nil: none) beside its card, keeping the screen
// open after the press that changed them.
func updateSides(p *player.Player, b SideBoard, open *sidePicker) {
	oreScreens.Lock()
	sidesTurn++
	l := newOreLayout()
	l.prefix = "s" + strconv.Itoa(sidesTurn) + "_"
	oreScreens.Unlock()
	choose := func(run func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if run != nil {
				run(p, tx)
			}
			if b.Refresh != nil {
				updateSides(p, b.Refresh(p, tx), nil)
			}
		}
	}
	vars := map[string]any{"@picker": box("")}
	for i, s := range b.Sides {
		if len(s.Ports) == 0 {
			vars["@"+sideRegion(i)] = box("mc-side-ports", "children", []any{
				box("mc-port-none", "children", []any{txt("", i18n.T(i18n.Of(p), "ui.sides.none"))}),
			})
			continue
		}
		var ports []any
		for j, port := range s.Ports {
			var children []any
			if n := len(port.Choices); n > 0 {
				cur := port.Choices[min(max(port.Chosen, 0), n-1)]
				swatch := choiceNode("mc-port-swatch", cur)
				swatch["tip"] = cur.Tip
				children = append(children, l.press(swatch, choose(port.Choices[(port.Chosen+1)%n].Run)))
			}
			plus := box("mc-port-plus", "children", []any{txt("", "+")})
			at := sidePicker{i, j}
			if open != nil && *open == at {
				plus["class"] = "mc-port-plus on"
				l.press(plus, func(p *player.Player, _ *world.Tx) { updateSides(p, b, nil) })
				vars["@picker"] = pickerNode(l, i, port, choose)
			} else {
				l.press(plus, func(p *player.Player, _ *world.Tx) { updateSides(p, b, &at) })
			}
			children = append(children, txt("mc-port-label", port.Label), plus)
			ports = append(ports, box("mc-port", "children", children))
		}
		vars["@"+sideRegion(i)] = box("mc-side-ports", "children", ports)
	}
	withScreen(p, func(st *oreScreen) {
		st.gen++
		for k, v := range l.runs {
			st.runs[k] = v
		}
		st.form.SetVars(vars)
	})
}

// pickerNode is the panel of the choices of port, on the card of side: under
// the card, or over it for a card on the bottom row. choose makes the run of
// a choice from the choice's Run.
func pickerNode(l *oreLayout, side int, port SidePort, choose func(func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx)) map[string]any {
	var choices []any
	for k, c := range port.Choices {
		class := "mc-port-choice"
		if k == port.Chosen {
			class += " on"
		}
		n := box(class, "tip", c.Tip, "children", []any{choiceNode("mc-choice-swatch", c)})
		choices = append(choices, l.press(n, choose(c.Run)))
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
	return box("mc-port-picker", "style", map[string]any{"left": gu(x), "top": gu(y), "width": gu(w)}, "children", choices)
}

// sideRows are the ItemBoard rows of a SidesMenu shown with actions: the
// block beside each side and a slot for each port showing its choice. A port
// opens its choices in an ItemGrid; a choice shows the menu again from
// b.Refresh.
func sideRows(title string, b SideBoard, actions []Option) [][]BoardCell {
	again := func(p *player.Player, tx *world.Tx) {
		if b.Refresh != nil {
			SidesMenu(p, title, b.Refresh(p, tx), actions...)
		}
	}
	var rows [][]BoardCell
	for i, s := range b.Sides {
		row := []BoardCell{SlotCell(ItemSlot{Icon: s.Icon, Tip: s.Name + ": " + s.Tip, Selected: i == b.Focus}), Blank()}
		for _, port := range s.Ports {
			var shown PortChoice
			if port.Chosen >= 0 && port.Chosen < len(port.Choices) {
				shown = port.Choices[port.Chosen]
			}
			choices := port.Choices
			row = append(row, SlotCell(ItemSlot{
				Icon: shown.Icon,
				Tip:  fmt.Sprintf("%s: %s", port.Label, shown.Tip),
				Run: func(p *player.Player, tx *world.Tx) {
					var grid []ItemSlot
					for _, c := range choices {
						run := c.Run
						grid = append(grid, ItemSlot{Icon: c.Icon, Tip: c.Tip, Run: func(p *player.Player, tx *world.Tx) {
							if run != nil {
								run(p, tx)
							}
							again(p, tx)
						}})
					}
					ItemGrid(p, title, "", grid, Back(i18n.T(i18n.Of(p), "ui.sides.back"), again))
				},
			}))
		}
		rows = append(rows, row)
	}
	return rows
}
