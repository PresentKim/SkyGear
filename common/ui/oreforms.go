package ui

import (
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// DDUI screens for ListSplitMenu, Input, ItemGrid(Sections) and SlotMenu.
// They share the common window size, so moving between them updates the open
// screen in place.

// dListRow is a row of the list of a list-split screen.
type dListRow struct {
	ID       string `json:"id"`
	Icon     string `json:"icon"`
	Name     string `json:"name"`
	Info     string `json:"info"`
	Selected bool   `json:"selected"`
}

// listSplitData is the data of the listsplit screen.
type listSplitData struct {
	Chrome  dChrome    `json:"chrome"`
	Rows    []dListRow `json:"rows"`
	Body    string     `json:"body"`
	Actions []dButton  `json:"actions"`
}

// inputData is the data of the input screen; Field is the id of the input
// box and Submit the button that sends its text.
type inputData struct {
	Chrome dChrome `json:"chrome"`
	Label  string  `json:"label"`
	Field  string  `json:"field"`
	Value  string  `json:"value"`
	Submit dButton `json:"submit"`
}

// gridData is the data of the grid screen.
type gridData struct {
	Chrome   dChrome    `json:"chrome"`
	Body     string     `json:"body"`
	Sections []dSlotRow `json:"sections"`
	Items    []dItem    `json:"items"`
}

// slotMenuData is the data of the slotmenu screen.
type slotMenuData struct {
	Chrome dChrome `json:"chrome"`
	Store  dSlot   `json:"store"`
	Info   string  `json:"info"`
	Left   []dSlot `json:"left"`
	Right  []dSlot `json:"right"`
	Items  []dItem `json:"items"`
}

// oreListSplit is a ListSplitMenu: the rows in a list on the left, and on the
// right the body and the actions.
func oreListSplit(p *player.Player, title string, sp ListSplit) {
	l := newOreLayout()
	chrome, actions := l.chromeOf(title, sp.Actions)
	rows := make([]dListRow, 0, len(sp.Rows))
	for _, r := range sp.Rows {
		name, info := listRowParts(r.Label)
		rows = append(rows, dListRow{ID: l.id(r.Run), Icon: oreIcon(r.Icon), Name: name, Info: info, Selected: r.Selected})
	}
	showScreen(p, l, "listsplit", listSplitData{Chrome: chrome, Rows: rows, Body: sp.Body, Actions: l.buttonsOf(actions)})
}

// listRowParts cuts a ListRowText label into the name (the padding taken
// off) and the information line. A label without the name field is all name.
func listRowParts(label string) (name, info string) {
	if len(label) <= listNameBytes {
		return label, ""
	}
	name, info = label[:listNameBytes], label[listNameBytes:]
	name = strings.TrimRight(name, " ")
	for strings.HasSuffix(name, "§r") {
		name = strings.TrimSuffix(name, "§r")
	}
	return name, info
}

// oreInput is an Input: the label over an input box with the text def, and a
// button that runs f with the text (Enter presses it too). With back, a back
// button in the control bar shows the screen it came from.
func oreInput(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string), back func(p *player.Player, tx *world.Tx)) {
	lang := i18n.Of(p)
	l := newOreLayout()
	var opts []Option
	if back != nil {
		opts = append(opts, Back(i18n.T(lang, "ui.input.back"), back))
	}
	chrome, _ := l.chromeOf(title, opts)
	submit := i18n.T(lang, "ui.input.ok")
	if strings.Contains(title, CreateToken) {
		submit = i18n.T(lang, "ui.input.create")
	}
	const field = "text"
	l.values[field] = def
	ok := l.buttonOf(Option{Label: submit, Icon: "textures/ui/confirm", Run: func(p *player.Player, tx *world.Tx) {
		f(p, tx, l.value(field))
	}})
	showScreen(p, l, "input", inputData{Chrome: chrome, Label: label, Field: field, Value: def, Submit: ok})
}

// oreItemGrid is an ItemGrid or ItemGridSections: the body text, the headed
// sections of slots, and the actions (tabs and the control bar's buttons in
// their places, the rest as list rows).
func oreItemGrid(p *player.Player, title, body string, sections []GridSection, actions []Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, actions)
	rows := make([]SlotRow, 0, len(sections))
	for _, sec := range sections {
		rows = append(rows, SlotRow{Heading: sec.Heading, Slots: sec.Slots})
	}
	showScreen(p, l, "grid", gridData{Chrome: chrome, Body: body, Sections: l.rowsOf(rows), Items: l.itemsOf(rest)})
}

// oreSlotMenu is a SlotMenu: the storage slot with the info text beside it,
// under them the left slots and the right slots, and the actions as list
// rows. A slot's text is its tooltip; the storage slot's is its count.
func oreSlotMenu(p *player.Player, title string, store Slot, info string, left, right []Slot, actions []Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, actions)
	slots := func(list []Slot) []dSlot {
		out := make([]dSlot, 0, len(list))
		for _, sl := range list {
			out = append(out, l.slotOf(ItemSlot{Icon: sl.Icon, Tip: sl.Text, Run: sl.Run}))
		}
		return out
	}
	showScreen(p, l, "slotmenu", slotMenuData{
		Chrome: chrome,
		Store:  l.slotOf(ItemSlot{Icon: store.Icon, Count: store.Text}),
		Info:   info,
		Left:   slots(left),
		Right:  slots(right),
		Items:  l.itemsOf(rest),
	})
}
