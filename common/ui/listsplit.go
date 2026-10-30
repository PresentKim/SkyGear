package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// listNameBytes is the width of the name field of a row's label, padded with
// invisible "§r"s (3 bytes) and spaces so the screen can cut the name and the
// information under it apart and draw them at different sizes
// (listRowParts). Padding must never split a colour code.
const listNameBytes = 60

// ListRowText formats the label of a ListRow: the name padded to
// listNameBytes, then the information line (with its own colour codes).
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
// the right. A row's Run shows the screen again with that row chosen: a press
// that shows nothing closes the screen.
func ListSplitMenu(p *player.Player, title string, sp ListSplit) {
	oreListSplit(p, title, sp)
}
