// Package ui has small helpers to build forms whose buttons run closures.
package ui

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"log/slog"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
)

// Option is a menu button and the action it runs. Icon is an optional
// texture path shown on the button, e.g. "textures/items/emerald".
type Option struct {
	Label string
	Icon  string
	Run   func(p *player.Player, tx *world.Tx)
}

// pending remembers the last form sent to each player until it is answered,
// so a repeated request (the client repeats item use packets while the
// button is held) does not queue copies that reopen one after another.
var pending = struct {
	sync.Mutex
	m map[uuid.UUID]sentForm
}{m: map[uuid.UUID]sentForm{}}

type sentForm struct {
	title string
	at    time.Time
}

// duplicateWindow is how long an unanswered form blocks an identical one.
const duplicateWindow = 3 * time.Second

// send sends f unless a form with the same title was sent to p less than
// duplicateWindow ago and is not answered yet.
func send(p *player.Player, title string, f form.Form) {
	pending.Lock()
	if last, ok := pending.m[p.UUID()]; ok && last.title == title && time.Since(last.at) < duplicateWindow {
		pending.Unlock()
		return
	}
	pending.m[p.UUID()] = sentForm{title: title, at: time.Now()}
	pending.Unlock()
	p.SendForm(f)
}

// answered marks the form of the submitter answered (submitted or closed).
func answered(s form.Submitter) {
	if p, ok := s.(*player.Player); ok {
		pending.Lock()
		delete(pending.m, p.UUID())
		pending.Unlock()
	}
}

type menu struct{ opts []Option }

func (menu) Close(s form.Submitter, _ *world.Tx) { answered(s) }

func (m menu) Submit(s form.Submitter, pressed form.Button, tx *world.Tx) {
	answered(s)
	p := s.(*player.Player)
	defer recoverForm(p)
	for _, o := range m.opts {
		if o.Label == pressed.Text {
			// Headings and pads of grids have nothing to run.
			if o.Run != nil {
				o.Run(p, tx)
			}
			return
		}
	}
}

// recoverForm keeps a bug in a form callback from crashing the server: the
// panic is logged and the player is told the action failed.
func recoverForm(p *player.Player) {
	if r := recover(); r != nil {
		slog.Error("form callback panicked", "player", p.Name(), "panic", r, "stack", string(debug.Stack()))
		p.Message(i18n.T(i18n.Of(p), "ui.ui.recoverForm.1"))
	}
}

// Menu shows a menu form whose buttons run their options. The pressed
// button is found by its label (the client sends no index), so a label
// that repeats gets invisible "§r"s added until it is unique (all the empty
// upgrade slots of the drawer window ran the first one, 11th playtest).
func Menu(p *player.Player, title, body string, opts ...Option) {
	opts = uniqueLabels(opts)
	f := form.NewMenu(menu{opts: opts}, title).WithBody(body)
	for _, o := range opts {
		f = f.AddButton(form.NewButton(o.Label, o.Icon))
	}
	send(p, title, f)
}

// uniqueLabels returns opts with every label unique.
func uniqueLabels(opts []Option) []Option {
	seen := map[string]bool{}
	out := make([]Option, len(opts))
	for i, o := range opts {
		for seen[o.Label] {
			o.Label += "§r"
		}
		seen[o.Label] = true
		out[i] = o
	}
	return out
}

// Messages from form actions show as toasts: chat is hidden while forms
// are open, and a toast shows over them without interrupting the next form.

// Notify tells p about something that failed (not enough money, out of
// stock, ...): a toast only, as it matters just now.
func Notify(p *player.Player, format string, a ...any) {
	toast(p, i18n.T(i18n.Of(p), "ui.ui.Notify.1"), fmt.Sprintf(format, a...))
}

// Flash tells p the result of an action that matters later (a purchase, a
// sale, ...): a toast and the same line in chat, to read back.
func Flash(p *player.Player, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	toast(p, i18n.T(i18n.Of(p), "ui.ui.Flash.1"), msg)
	p.Message(msg)
}

// ToastGap is how long a toast stays on screen, roughly. The client shows
// toasts one after another, so many quick ones (buying again and again)
// came minutes late (10th playtest). A player gets at most one toast per
// ToastGap: what comes in between is merged into the next one, which shows
// the latest message and how many were folded into it.
var ToastGap = 3 * time.Second

// toasts are the players' toast states.
var toasts = struct {
	sync.Mutex
	m map[uuid.UUID]*toastState
}{m: map[uuid.UUID]*toastState{}}

type toastState struct {
	// next is when the shown toast is gone and another may be sent.
	next time.Time
	// title and msg are the latest message held back, folded counts how
	// many others it stands for.
	title, msg string
	folded     int
	waiting    bool
}

// toast sends a toast now, or holds it back until the shown one is gone.
func toast(p *player.Player, title, msg string) {
	toasts.Lock()
	defer toasts.Unlock()
	st := toasts.m[p.UUID()]
	if st == nil {
		st = &toastState{}
		toasts.m[p.UUID()] = st
	}
	now := time.Now()
	if st.waiting && now.After(st.next.Add(ToastGap)) {
		// The held toast never went out (the player changed worlds before
		// it was due); start over.
		st.waiting, st.folded = false, 0
	}
	if !st.waiting && !now.Before(st.next) {
		st.next = now.Add(ToastGap)
		p.SendToast(title, msg)
		return
	}
	if st.waiting {
		st.folded++
	}
	st.title, st.msg = title, msg
	if st.waiting {
		return
	}
	st.waiting = true
	p.H().DoAfter(st.next.Sub(now), func(_ *world.Tx, e world.Entity) {
		toasts.Lock()
		title, msg, folded := st.title, st.msg, st.folded
		st.waiting, st.folded, st.next = false, 0, time.Now().Add(ToastGap)
		toasts.Unlock()
		if folded > 0 {
			msg += fmt.Sprintf(i18n.T(i18n.Of(p), "ui.ui.toast.1"), folded)
		}
		e.(*player.Player).SendToast(title, msg)
	})
}

// Button markers (valid colour codes, so a client without the theme shows
// nothing extra) give a button a special place in the themed forms:
//
//   - TabToken / TabOnToken: a tab in the tab strip at the top of the
//     window (TabOnToken is the selected one);
//   - BackToken: the back button left of the close button;
//   - ActionToken: a button under a table;
//   - CellToken / CellNumToken: a non-clickable text or number cell of a
//     table (TableMenu).
//
// The themed lists and grids skip marked buttons. Put marked buttons after
// the normal ones so grids keep no gaps.
const (
	TabToken     = "§0§1§r"
	TabOnToken   = "§0§2§r"
	BackToken    = "§0§3§r"
	ActionToken  = "§0§4§r"
	CellToken    = "§0§5§r"
	CellNumToken = "§0§6§r"
	TotalToken   = "§0§8§r"
	StoreToken   = "§0§9§r"
	UpgradeToken = "§0§a§r"
	InfoToken    = "§0§b§r"
	UtilToken    = "§0§c§r"
)

// Tab returns a tab button of a themed form.
func Tab(label, icon string, selected bool, run func(p *player.Player, tx *world.Tx)) Option {
	token := TabToken
	if selected {
		token = TabOnToken
	}
	return Option{Label: token + label, Icon: icon, Run: run}
}

// SubTitle is the title of a screen below another one: "parent · sub". Every
// screen that goes deeper than its menu names its parent this way.
func SubTitle(parent, sub string) string { return parent + " · " + sub }

// Pager returns the buttons that turn pages of a list shown page of pages (both
// from 0 and 1 respectively: page is the index): "previous" when there is an
// earlier page and "next" when there is a later one, each naming the page it
// goes to. show shows that page again. Every paged list of the game uses it,
// so they read and sit the same (docs/tech/ui-style.md).
func Pager(p *player.Player, page, pages int, show func(p *player.Player, tx *world.Tx, page int)) []Option {
	lang := i18n.Of(p)
	var out []Option
	if page > 0 {
		out = append(out, Option{Label: fmt.Sprintf(i18n.T(lang, "ui.page.prev"), page, pages), Icon: "textures/ui/df/arrow_left",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page-1) }})
	}
	if page < pages-1 {
		out = append(out, Option{Label: fmt.Sprintf(i18n.T(lang, "ui.page.next"), page+2, pages), Icon: "textures/ui/df/arrow_right",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page+1) }})
	}
	return out
}

// Back returns the back button of a themed form, drawn left of the close
// button.
func Back(label string, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: BackToken + label, Icon: "textures/ui/arrow_left", Run: run}
}

// CardToken starts the title of a menu shown as a grid of icon-and-text
// cards, three per row (assets/resourcepack/dragonfactory/ui).
const CardToken = "§d§f§c§r"

// CardMenu shows a menu as a grid of cards with the icon above the label,
// so many options fit on one screen. Labels should be short (two lines).
func CardMenu(p *player.Player, title, body string, opts ...Option) {
	Menu(p, CardToken+title, body, opts...)
}

// ShopToken starts the title of a compact card grid: smaller cards, more
// per row, for long product lists.
const ShopToken = "§d§f§s§r"

// ShopMenu sends a compact card grid form.
func ShopMenu(p *player.Player, title, body string, opts ...Option) {
	Menu(p, ShopToken+title, body, opts...)
}

// TableToken starts the title of a table form (TableMenu).
const TableToken = "§d§f§t§r"

// Cell is one cell of a table row: text, an optional icon, and whether it
// is a number (drawn right-aligned) or a header.
type Cell struct {
	Text, Icon string
	Num, Head  bool
}

// TableMenu sends a form that shows rows of cells as a table (columns
// cells per row, header row first), then total, then the action buttons.
// Cells are not buttons: only the actions answer.
func TableMenu(p *player.Player, title, body string, rows [][]Cell, total string, actions ...Option) {
	var opts []Option
	for _, row := range rows {
		for _, c := range row {
			token, text := CellToken, c.Text
			if c.Num {
				token = CellNumToken
			}
			if c.Head {
				text = "§7" + text
			}
			opts = append(opts, Option{Label: token + text, Icon: c.Icon, Run: func(*player.Player, *world.Tx) {}})
		}
	}
	if total != "" {
		opts = append(opts, Option{Label: TotalToken + total, Run: func(*player.Player, *world.Tx) {}})
	}
	for _, a := range actions {
		a.Label = ActionToken + a.Label
		opts = append(opts, a)
	}
	Menu(p, TableToken+title, body, opts...)
}

// SlotToken starts the title of a slot form (SlotMenu).
const SlotToken = "§d§f§i§r"

// Slot is one slot of a SlotMenu: an icon (ItemIcon or a texture path), the
// text at its bottom right (a count, or empty) and what pressing it does.
type Slot struct {
	Icon, Text string
	Run        func(p *player.Player, tx *world.Tx)
}

// SlotMenu sends an inventory-like form (df_form.json slot_form): a
// storage slot that only shows (icon and count) with info text on its
// right, then two rows of pressable slots: left (e.g. storage upgrades) and
// right (utilities), then action buttons.
func SlotMenu(p *player.Player, title string, store Slot, info string, left, right []Slot, actions ...Option) {
	nop := func(*player.Player, *world.Tx) {}
	opts := []Option{
		{Label: StoreToken + store.Text, Icon: store.Icon, Run: nop},
		{Label: InfoToken + info, Run: nop},
	}
	add := func(token string, slots []Slot) {
		for _, sl := range slots {
			run := sl.Run
			if run == nil {
				run = nop
			}
			opts = append(opts, Option{Label: token + sl.Text, Icon: sl.Icon, Run: run})
		}
	}
	add(UpgradeToken, left)
	add(UtilToken, right)
	for _, a := range actions {
		a.Label = ActionToken + a.Label
		opts = append(opts, a)
	}
	Menu(p, SlotToken+title, "", opts...)
}

// Item slot forms (ItemDetail, ItemGrid). A slot is a button whose text holds
// a 9 byte marker, one byte saying which side its tooltip grows to, its
// count in 10 bytes (starts with a colour code, padded with '~') and then
// its tooltip; the pack cuts them apart
// (assets/resourcepack/dragonfactory/ui/df_form.json, docs/tech/json-ui.md).
const (
	// DetailToken starts the title of an item detail form (ItemDetail).
	DetailToken = "§d§f§g§r"
	// WrapToken starts the title of an item slot form in a wrapping grid
	// (ItemGrid).
	WrapToken = "§d§f§h§r"
	// ColumnsToken starts the title of a form with two columns of slots
	// (ColumnsMenu).
	ColumnsToken = "§d§f§j§r"

	// colsATok and colsAmTok mark the full slots and mini buttons of the left
	// column of a ColumnsMenu, colsBTok and colsBmTok those of the right one.
	colsATok  = "§0§k§r"
	colsAmTok = "§0§l§r"
	colsBTok  = "§0§m§r"
	colsBmTok = "§0§n§r"

	slotAToken = "§0§d§r"
	slotBToken = "§0§e§r"
	slotWToken = "§0§f§r"
	head1Token = "§0§g§r"
	head2Token = "§0§h§r"
	// HelpToken marks the "?" button in the title bar (Help); padToken marks
	// an empty cell that fills a row of a sectioned grid.
	HelpToken    = "§0§p§r"
	padToken     = "§0§q§r"
	detailMarker = "§0§i§r"

	// slotCountBytes is the width of the count field of a slot.
	slotCountBytes = 10
	// tipLeftSlots is how many slots from the left of a row have their
	// tooltip grow to the right; the later ones grow to the left (the pane
	// clips a tooltip at its edges).
	tipLeftSlots = 5
	// gridColumns is how many slots a wrapping grid holds per row, and
	// gridTipLeft how many of them from the left grow their tooltip right.
	gridColumns = 11
	gridTipLeft = 6
)

// ItemSlot is one slot of an item slot form: an icon (ItemIcon or a texture
// path), the count at its bottom right (starts with a colour code, at most
// slotCountBytes bytes), the tooltip shown on hover (or the first tap on
// touch) and what pressing it does. Run must show a form again: any press
// answers the form.
type ItemSlot struct {
	Icon, Count, Tip string
	Run              func(p *player.Player, tx *world.Tx)
	// Mini makes a ColumnsMenu slot a small button without a slot frame, drawn
	// beside the slot before it.
	Mini bool
}

// SlotRow is a heading with a row of slots (at most 8 fit).
type SlotRow struct {
	Heading string
	Slots   []ItemSlot
}

// padSlotCount cuts s to at most n bytes (whole characters) and fills it
// with '~' up to n bytes; '~' in s is dropped.
func padSlotCount(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r == '~' {
			continue
		}
		if b.Len()+utf8.RuneLen(r) > n {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + strings.Repeat("~", n-b.Len())
}

// CountText formats an item count for a slot's bottom right: whole up to
// 9999, then in thousands ("12k") or millions ("3m").
func CountText(n int) string {
	switch {
	case n < 10000:
		return strconv.Itoa(n)
	case n < 1000000:
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.Itoa(min(n/1000000, 999)) + "m"
}

// slotLabel is the button text of a slot of the kind marker; tipRight says
// the tooltip grows to the left (a slot on the right side). A count is kept
// a text for the pack: it starts with a colour code ("§f" if it has none).
func slotLabel(marker string, tipRight bool, sl ItemSlot) string {
	count := sl.Count
	if count != "" && !strings.HasPrefix(count, "§") {
		count = "§f" + count
	}
	align := "l"
	if tipRight {
		align = "r"
	}
	return marker + align + padSlotCount(count, slotCountBytes) + sl.Tip
}

// slotOption is the option of a slot (nil Run does nothing).
func slotOption(marker string, tipRight bool, sl ItemSlot) Option {
	run := sl.Run
	if run == nil {
		run = func(*player.Player, *world.Tx) {}
	}
	return Option{Label: slotLabel(marker, tipRight, sl), Icon: sl.Icon, Run: run}
}

// marked reports whether a label starts with one of the button markers.
func marked(label string) bool { return strings.HasPrefix(label, "§0§") }

// actionOptions marks the buttons under the slots (a Back button already
// carries its marker).
func actionOptions(actions []Option) []Option {
	out := make([]Option, len(actions))
	for i, a := range actions {
		if !marked(a.Label) {
			a.Label = ActionToken + a.Label
		}
		out[i] = a
	}
	return out
}

// Detail is the card of an ItemDetail form: a big item, the name and status
// text beside it (the action buttons come right under them) and the body
// text under the card.
type Detail struct {
	Icon, Name, Status, Body string
}

// ItemDetail sends a two column form: on the left the Detail card with the
// action buttons under its name and the body text under the card, on the
// right up to two headed rows of item slots (rows[0] above rows[1]). The
// actions come first among the buttons, so their indexes do not depend on
// the slots.
func ItemDetail(p *player.Player, title string, d Detail, rows []SlotRow, actions ...Option) {
	opts := actionOptions(actions)
	name := d.Name
	if d.Status != "" {
		name += "\n" + d.Status
	}
	opts = append(opts, Option{Label: detailMarker + name, Icon: d.Icon, Run: func(*player.Player, *world.Tx) {}})
	kinds := [2][2]string{{head1Token, slotAToken}, {head2Token, slotBToken}}
	for i, row := range rows {
		if i >= len(kinds) {
			break
		}
		if row.Heading != "" {
			opts = append(opts, Option{Label: kinds[i][0] + row.Heading, Run: func(*player.Player, *world.Tx) {}})
		}
		for j, sl := range row.Slots {
			opts = append(opts, slotOption(kinds[i][1], j >= tipLeftSlots, sl))
		}
	}
	Menu(p, DetailToken+title, d.Body, opts...)
}

// ColumnsMenu sends a form with the body text on top and under it two columns
// side by side, each a heading with a row of slots (left, right) and, after
// a slot, small buttons without a frame (ItemSlot.Mini). The tooltips of the left column grow to the right and those
// of the right column to the left. Any press answers the form: Run must show a
// form again.
func ColumnsMenu(p *player.Player, title, body string, left, right SlotRow, actions ...Option) {
	opts := actionOptions(actions)
	// full slot and mini button markers of each column
	kinds := [2][2]string{{colsATok, colsAmTok}, {colsBTok, colsBmTok}}
	heads := [2]string{head1Token, head2Token}
	for i, row := range []SlotRow{left, right} {
		if row.Heading != "" {
			opts = append(opts, Option{Label: heads[i] + row.Heading, Run: func(*player.Player, *world.Tx) {}})
		}
		for _, sl := range row.Slots {
			marker := kinds[i][0]
			if sl.Mini {
				marker = kinds[i][1]
			}
			opts = append(opts, slotOption(marker, i == 1, sl))
		}
	}
	Menu(p, ColumnsToken+title, body, opts...)
}

// ItemGrid sends a form with the body text, then the item slots in a grid
// that wraps (11 per row, scrolling) and the action buttons. The slots come
// first among the buttons: the hidden action buttons would otherwise take
// the first cells of the grid and push the slots to the right.
func ItemGrid(p *player.Player, title, body string, slots []ItemSlot, actions ...Option) {
	opts := make([]Option, 0, len(slots)+len(actions))
	for i, sl := range slots {
		opts = append(opts, slotOption(slotWToken, i%gridColumns >= gridTipLeft, sl))
	}
	opts = append(opts, actionOptions(actions)...)
	Menu(p, WrapToken+title, body, opts...)
}

// Help is the "?" button left of the back button in the title bar: the tip
// shows when the mouse is over it, pressing it runs run (which shows the
// form again with the text open, for touch).
func Help(tip string, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: HelpToken + tip, Run: run}
}

// GridSection is a headed part of ItemGridSections.
type GridSection struct {
	Heading string
	Slots   []ItemSlot
}

// ItemGridSections is ItemGrid with headed sections one after the other:
// every section starts on a new row with its heading in the first cell, and
// its slots wrap in rows after it. Empty cells (pad buttons) bring the
// section to a whole number of rows, so Go decides where the rows break.
func ItemGridSections(p *player.Player, title, body string, sections []GridSection, actions ...Option) {
	opts := sectionOptions(sections)
	opts = append(opts, actionOptions(actions)...)
	Menu(p, WrapToken+title, body, opts...)
}

// sectionOptions is the buttons of the sections: heading, pads, slots, pads.
func sectionOptions(sections []GridSection) []Option {
	var opts []Option
	pad := func(n int) {
		for range n {
			opts = append(opts, Option{Label: padToken})
		}
	}
	for _, sec := range sections {
		opts = append(opts, Option{Label: head1Token + sec.Heading})
		pad(gridColumns - 1)
		for _, sl := range sec.Slots {
			opts = append(opts, slotOption(slotWToken, false, sl))
		}
		pad((gridColumns - len(sec.Slots)%gridColumns) % gridColumns)
	}
	return opts
}

// Confirm asks a yes/no question as a themed menu (the vanilla modal form
// is not themed): the yes button runs f, "취소" or closing does nothing.
func Confirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	Menu(p, title, body,
		Option{Label: yes, Icon: "textures/ui/confirm", Run: f},
		Option{Label: i18n.T(i18n.Of(p), "ui.ui.Confirm.1"), Icon: "textures/ui/cancel", Run: func(*player.Player, *world.Tx) {}},
	)
}

type textInput struct {
	Value form.Input
	run   func(p *player.Player, tx *world.Tx, value string)
}

func (textInput) Close(s form.Submitter, _ *world.Tx) { answered(s) }

func (t textInput) Submit(s form.Submitter, tx *world.Tx) {
	answered(s)
	p := s.(*player.Player)
	defer recoverForm(p)
	t.run(p, tx, t.Value.Value())
}

// Input asks for one line of text.
func Input(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string)) {
	send(p, title, form.New(textInput{Value: form.NewInput(label, def, ""), run: f}, title))
}

type amount struct {
	Value form.Slider
	run   func(p *player.Player, tx *world.Tx, n int)
}

func (amount) Close(s form.Submitter, _ *world.Tx) { answered(s) }

func (a amount) Submit(s form.Submitter, tx *world.Tx) {
	answered(s)
	p := s.(*player.Player)
	defer recoverForm(p)
	a.run(p, tx, int(a.Value.Value()))
}

// Amount asks for a number from 1 to max with a slider starting at def.
func Amount(p *player.Player, title, label string, max, def int, f func(p *player.Player, tx *world.Tx, n int)) {
	send(p, title, form.New(amount{Value: form.NewSlider(label, 1, float64(max), 1, float64(min(max, def))), run: f}, title))
}

// ItemIcon returns a button icon that the resource pack theme draws as the
// item itself, with the client's item renderer, instead of a texture path:
// the item's runtime ID times 65536 (the renderer's "#item_id_aux"; the
// metadata is not used). It returns "" for an unregistered item. Blocks
// show as blocks and no icon texture is needed (docs/tech/json-ui.md).
func ItemIcon(it world.Item) string {
	rid, _, ok := world.ItemRuntimeID(it)
	if !ok {
		return ""
	}
	return strconv.FormatInt(int64(rid)*65536, 10)
}
