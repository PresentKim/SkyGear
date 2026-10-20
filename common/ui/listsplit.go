package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// ListSplitToken starts the title of a form with a list of rows on the left
// and the detail of the chosen row on the right (ListSplitMenu).
const ListSplitToken = "§d§f§l§r"

// listRowToken and listRowOnToken mark a row of the list, the chosen one
// last. The actions are marked by their place in the grid on the right: the
// first is the full width one, the others alternate left and right column.
const (
	listRowToken     = "§0§o§r"
	listRowOnToken   = "§0§x§r"
	listPrimaryToken = "§0§0§r"
	listColAToken    = "§0§y§r"
	listColBToken    = "§0§z§r"
)

// listNameBytes is the width of the name field of a row's label, padded with
// invisible "§r"s (3 bytes) and spaces so the pack can cut the name and the
// information under it apart and draw them at different sizes
// (docs/tech/json-ui.md). Padding must never split a colour code: the client
// then shows the "r".
const listNameBytes = 60

// ListRowText formats the label of a ListRow: the name padded to
// listNameBytes, then the information line (with its own colour codes).
// ListSplitOptions puts the row's marker in front of it.
func ListRowText(name, info string) string {
	var b strings.Builder
	for _, r := range name {
		if b.Len()+utf8.RuneLen(r) > listNameBytes {
			break
		}
		b.WriteRune(r)
	}
	for ; listNameBytes-b.Len() >= 3; b.WriteString("§r") {
	}
	for b.Len() < listNameBytes {
		b.WriteByte(' ')
	}
	return b.String() + info
}

// ListRow is one row of a ListSplitMenu: an icon, a label of up to two lines
// and what pressing it does. Selected draws the row as the chosen one.
type ListRow struct {
	Label, Icon string
	Selected    bool
	Run         func(p *player.Player, tx *world.Tx)
}

// ListSplit is the content of a ListSplitMenu.
type ListSplit struct {
	// Rows are the rows of the list, in order.
	Rows []ListRow
	// Body is the description under the rows of the actions.
	Body string
	// Actions are the buttons on the right.
	Actions []Option
}

// ListSplitMenu shows a list on the left and the actions of the chosen row on
// the right. Pressing any row answers the form, so Run must show the form
// again with that row chosen, the chosen one included: with nothing sent back
// the client closes the window, so pressing the chosen row used to close the
// screen instead of keeping it.
func ListSplitMenu(p *player.Player, title string, sp ListSplit) {
	Menu(p, ListSplitToken+title, sp.Body, listSplitOptions(sp)...)
}

// listSplitOptions is the buttons of a ListSplitMenu: the rows and then the
// actions (the first is the full width one, the rest alternate columns).
func listSplitOptions(sp ListSplit) []Option {
	nop := func(*player.Player, *world.Tx) {}
	opts := make([]Option, 0, len(sp.Rows)+len(sp.Actions))
	for _, r := range sp.Rows {
		marker := listRowToken
		if r.Selected {
			marker = listRowOnToken
		}
		run := r.Run
		if run == nil {
			run = nop
		}
		opts = append(opts, Option{Label: marker + r.Label, Icon: r.Icon, Run: run})
	}
	for i, a := range sp.Actions {
		if marked(a.Label) {
			opts = append(opts, a)
			continue
		}
		token := listPrimaryToken
		switch {
		case i == 0:
			token = listPrimaryToken
		case i%2 == 1:
			token = listColAToken
		default:
			token = listColBToken
		}
		a.Label = token + a.Label
		opts = append(opts, a)
	}
	return opts
}
