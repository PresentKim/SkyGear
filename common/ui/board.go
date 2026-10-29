package ui

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// BoardColumns is how many cells a row of an ItemBoard holds.
const BoardColumns = 11

// barPagerToken marks a small icon button of the control bar: a page arrow
// (BarPager) or a BarButton.
const barPagerToken = "§0§j§r"

// cellKind is what a BoardCell draws.
type cellKind int

const (
	cellBlank cellKind = iota
	cellSlot
	cellFrame
	cellArrow
	cellText
	cellMini
)

// BoardCell is one cell of an ItemBoard row.
type BoardCell struct {
	kind cellKind
	slot ItemSlot
	text string
}

// Blank is an empty cell.
func Blank() BoardCell { return BoardCell{} }

// SlotCell is an item slot; a Selected slot is drawn lit.
func SlotCell(sl ItemSlot) BoardCell { return BoardCell{kind: cellSlot, slot: sl} }

// Frame is an empty slot frame (the empty places of a crafting grid).
func Frame() BoardCell { return BoardCell{kind: cellFrame} }

// Arrow is the arrow from the ingredients to the result.
func Arrow() BoardCell { return BoardCell{kind: cellArrow} }

// MiniCell is a small icon without a slot frame, with its tooltip and Run.
func MiniCell(sl ItemSlot) BoardCell { return BoardCell{kind: cellMini, slot: sl} }

// TextCell is a line of small text that runs over the blank cells after it.
func TextCell(s string) BoardCell { return BoardCell{kind: cellText, text: s} }

// ItemBoard shows the body text and rows of cells, BoardColumns to a row,
// drawn like a recipe viewer: slots, empty frames, arrows and lines of text.
// A row shorter than BoardColumns is filled with blank cells; a longer one is
// cut.
func ItemBoard(p *player.Player, title, body string, rows [][]BoardCell, actions ...Option) {
	oreBoard(p, title, body, rows, actions)
}

// BarPager returns the page arrows drawn in the control bar left of the help
// button, for a list shown page of pages (page from 0): the arrow to the page
// before when there is one and the arrow to the page after, their tooltip
// naming the page they go to. show shows that page again.
func BarPager(p *player.Player, page, pages int, show func(p *player.Player, tx *world.Tx, page int)) []Option {
	lang := i18n.Of(p)
	var out []Option
	if page > 0 {
		out = append(out, Option{Label: barPagerToken + fmt.Sprintf(i18n.T(lang, "ui.page.prev"), page, pages), Icon: "textures/ui/sg/arrow_left",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page-1) }})
	}
	if page < pages-1 {
		out = append(out, Option{Label: barPagerToken + fmt.Sprintf(i18n.T(lang, "ui.page.next"), page+2, pages), Icon: "textures/ui/sg/arrow_right",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page+1) }})
	}
	return out
}

// BarButton is a small icon button in the control bar, where BarPager draws
// its arrows: label is its tooltip.
func BarButton(label, icon string, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: barPagerToken + label, Icon: icon, Run: run}
}

// SlotOf returns the slot of a slot cell.
func (c BoardCell) SlotOf() (ItemSlot, bool) { return c.slot, c.kind == cellSlot }

// IsArrow reports whether the cell is an arrow.
func (c BoardCell) IsArrow() bool { return c.kind == cellArrow }

// IsHeader reports whether the cell starts a header row: a small icon or a
// line of text.
func (c BoardCell) IsHeader() bool { return c.kind == cellMini || c.kind == cellText }

// IsBlank reports whether the cell draws nothing.
func (c BoardCell) IsBlank() bool { return c.kind == cellBlank }

// IsText reports whether the cell is a line of text (it runs over the blank
// cells after it).
func (c BoardCell) IsText() bool { return c.kind == cellText }
