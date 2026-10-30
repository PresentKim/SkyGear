// Package ui builds the menus as server-driven DDUI screens drawn on the
// game's Ore UI (oreui.go): their buttons run closures.
package ui

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/internal/slottext"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func init() {
	sghook.OnClose(func(c *sghook.Conn) {
		if id, err := uuid.Parse(c.IdentityData().Identity); err == nil {
			forget(id)
		}
	})
	sghook.OnServerPacket(plainText)
}

// plainText puts the fallback texts in place of the client spans
// (i18n.ClientKey) of what only the Ore UI page translates, should a span end
// up in chat or a title.
func plainText(_ *sghook.Conn, pk packet.Packet) bool {
	switch pk := pk.(type) {
	case *packet.Text:
		pk.Message = i18n.Plain(pk.Message)
		for i, s := range pk.Parameters {
			pk.Parameters[i] = i18n.Plain(s)
		}
	case *packet.SetTitle:
		pk.Text = i18n.Plain(pk.Text)
	}
	return false
}

// forget drops what is kept about the player with id once it quit.
func forget(id uuid.UUID) {
	toasts.Lock()
	delete(toasts.m, id)
	toasts.Unlock()
	oreScreens.Lock()
	delete(oreScreens.m, id)
	oreScreens.Unlock()
}

// Option is a menu button and the action it runs. Icon is an optional
// texture path shown on the button, e.g. "textures/items/emerald".
type Option struct {
	Label string
	Icon  string
	Run   func(p *player.Player, tx *world.Tx)
	// toggle draws the option as a row with a switch, on when on (Toggle).
	toggle, on bool
	// row is how a list draws an option that is no button (Heading,
	// Divider); "" is a button.
	row string
}

// Heading is a list's heading over the rows after it, not pressable.
func Heading(label string) Option { return Option{Label: label, row: "heading"} }

// Divider is a line across a list between two groups of rows.
func Divider() Option { return Option{row: "divider"} }

// Toggle is a menu option drawn as a row with an Ore UI switch on its right,
// on when on. Pressing the row runs run, which should change the setting and
// show the menu again.
func Toggle(label string, on bool, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: label, Run: run, toggle: true, on: on}
}

// FormOpen reports whether p has a DDUI screen open.
func FormOpen(p *player.Player) bool { return oreOpen(p) }

// Idle, if set, runs when p closed a DDUI screen (closed: Esc or the close
// button, title "") or pressed something on it that showed no other screen.
// It runs with the screen still open: a screen it shows replaces that one in
// place, and the screen closes if it shows none. When the client closed the
// screen itself, it runs after the screen is gone.
var Idle func(p *player.Player, tx *world.Tx, title string, closed bool)

// recoverForm keeps a bug in a screen's callback from crashing the server:
// the panic is logged and the player is told the action failed.
func recoverForm(p *player.Player) {
	if r := recover(); r != nil {
		slog.Error("form callback panicked", "player", p.Name(), "panic", r, "stack", string(debug.Stack()))
		Tell(p, "", i18n.T(i18n.Of(p), "ui.form.error"))
	}
}

// InLobby reports whether p is in the lobby. The platform sets it.
var InLobby func(p *player.Player) bool

// Menu shows the body text and a list button per option. Tab, Back, Help and
// BarPager options go over the window and to its control bar.
func Menu(p *player.Player, title, body string, opts ...Option) {
	oreMenu(p, title, body, opts)
}

// Messages from screen actions show as toasts: chat is hidden while a screen
// is open, and a toast shows over it without interrupting the next screen.

// Notify tells p about something that failed (not enough money, out of
// stock, ...): a toast only, as it matters just now.
func Notify(p *player.Player, format string, a ...any) {
	toast(p, i18n.T(i18n.Of(p), "ui.notify.title"), fmt.Sprintf(format, a...))
}

// Flash tells p the result of an action that matters later (a purchase, a
// sale, ...): a toast and the same line in chat, to read back (no chat in the
// lobby).
func Flash(p *player.Player, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	toast(p, i18n.T(i18n.Of(p), "ui.flash.title"), msg)
	if InLobby == nil || !InLobby(p) {
		p.Message(msg)
	}
}

// Tell writes msg in p's chat. The lobby has no chat: there it is a toast
// titled title (ui.tell.title when empty) instead.
func Tell(p *player.Player, title, msg string) {
	if InLobby != nil && InLobby(p) {
		if title == "" {
			title = i18n.T(i18n.Of(p), "ui.tell.title")
		}
		toast(p, title, msg)
		return
	}
	p.Message(msg)
}

// ToastGap is roughly how long a toast stays on screen. The client queues
// toasts, so a player gets at most one per ToastGap: those in between fold
// into the next, which shows the latest message and how many it stands for.
var ToastGap = 3 * time.Second

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

// Toast shows p a toast titled title, like Notify and Flash do: at most one
// a ToastGap, the ones in between folded into the next.
func Toast(p *player.Player, title, msg string) { toast(p, title, msg) }

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
			msg += fmt.Sprintf(i18n.T(i18n.Of(p), "ui.toast.more"), folded)
		}
		e.(*player.Player).SendToast(title, msg)
	})
}

// Button markers in front of a label give the button its place on a screen
// (oreLayout.chrome) and are taken off the text (untoken):
//
//   - TabToken / TabOnToken: a tab over the window (TabOnToken is the
//     selected one);
//   - BackToken: the back button of the control bar;
//   - HelpToken: the "?" button of the control bar (Help).
const (
	TabToken   = "§0§1§r"
	TabOnToken = "§0§2§r"
	BackToken  = "§0§3§r"
	HelpToken  = "§0§p§r"
)

// Tab returns a tab button.
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
// goes to. show shows that page again.
func Pager(p *player.Player, page, pages int, show func(p *player.Player, tx *world.Tx, page int)) []Option {
	lang := i18n.Of(p)
	var out []Option
	if page > 0 {
		out = append(out, Option{Label: fmt.Sprintf(i18n.T(lang, "ui.page.prev"), page, pages), Icon: "textures/ui/sg/arrow_left",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page-1) }})
	}
	if page < pages-1 {
		out = append(out, Option{Label: fmt.Sprintf(i18n.T(lang, "ui.page.next"), page+2, pages), Icon: "textures/ui/sg/arrow_right",
			Run: func(p *player.Player, tx *world.Tx) { show(p, tx, page+1) }})
	}
	return out
}

// Back returns the back button, drawn in the control bar left of the close
// button.
func Back(label string, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: BackToken + label, Icon: "textures/ui/sg/chevron_left", Run: run}
}

// CardMenu shows a menu as a grid of cards with the icon above the label,
// four to a row, so many options fit on one screen. Labels should be short
// (two lines). An empty title draws none.
func CardMenu(p *player.Player, title, body string, opts ...Option) {
	oreCardMenu(p, title, body, opts)
}

// Slot is one slot of a SlotMenu: an icon (ItemIcon or a texture path), the
// text at its bottom right (a count, or empty) and what pressing it does.
type Slot struct {
	Icon, Text string
	Run        func(p *player.Player, tx *world.Tx)
}

// SlotMenu shows an inventory-like screen: a storage slot that only shows
// (icon and count) with info text on its right, then two rows of pressable
// slots: left (e.g. storage upgrades) and right (utilities), then action
// buttons.
func SlotMenu(p *player.Player, title string, store Slot, info string, left, right []Slot, actions ...Option) {
	oreSlotMenu(p, title, store, info, left, right, actions)
}

// ItemSlot is one item slot: an icon (ItemIcon or a texture path), the count
// at its bottom right, the tooltip shown on hover (or the first tap on touch)
// and what pressing it does. A Selected slot cannot be pressed (its tooltip
// still shows): it is the one already shown. A slot without Run cannot be
// pressed either.
type ItemSlot struct {
	Icon, Count, Tip string
	Run              func(p *player.Player, tx *world.Tx)
	// Selected draws the slot as the chosen one, which cannot be pressed.
	Selected bool
}

// SplitCells is how many slots a page of a SplitMenu grid holds (8 columns,
// 6 rows); the page arrows sit under it.
const SplitCells = 48

// Split is the content of a SplitMenu.
type Split struct {
	// Tabs are the tab buttons over the grid (Tab).
	Tabs []Option
	// Slots are the cells of the grid, at most SplitCells; the rest of the
	// grid shows empty frames. One may be Selected.
	Slots []ItemSlot
	// Page and Pages are the page shown (from 0) and how many there are; Turn
	// shows another page. The arrows under the grid are always drawn: one
	// with no page to go to is dimmed and cannot be pressed.
	Page, Pages int
	Turn        func(p *player.Player, tx *world.Tx, page int)
	// Detail, Rows and Actions are what the right side shows for the chosen
	// slot: the card with the body text under it, the headed rows of slots
	// and the action buttons (Help and Back go to the control bar).
	Detail  Detail
	Rows    []SlotRow
	Actions []Option
}

// SlotRow is a heading with a row of slots.
type SlotRow struct {
	Heading string
	Slots   []ItemSlot
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

// Detail is the card of a SplitMenu's chosen slot: a big item, the name and
// status text beside it (the action buttons come right under them) and the
// body text under the card.
type Detail struct {
	Icon, Name, Status, Body string
}

// ItemGrid shows the body text, then the item slots in a grid that wraps in
// the window's width (scrolling) and the action buttons.
func ItemGrid(p *player.Player, title, body string, slots []ItemSlot, actions ...Option) {
	oreItemGrid(p, title, body, []GridSection{{Slots: slots}}, actions)
}

// Help is the "?" button of the control bar: the tip shows when the mouse is
// over it (the first tap on touch), pressing it runs run.
func Help(tip string, run func(p *player.Player, tx *world.Tx)) Option {
	return Option{Label: HelpToken + tip, Run: run}
}

// GridSection is a headed part of ItemGridSections.
type GridSection struct {
	Heading string
	Slots   []ItemSlot
}

// ItemGridSections is ItemGrid with headed sections one after the other:
// every section starts under its heading and its slots wrap after it.
func ItemGridSections(p *player.Player, title, body string, sections []GridSection, actions ...Option) {
	oreItemGrid(p, title, body, sections, actions)
}

// Confirm asks a yes/no question: the yes button runs f, "취소" or closing
// does nothing.
func Confirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	Menu(p, title, body,
		Option{Label: yes, Icon: "textures/ui/confirm", Run: f},
		Option{Label: i18n.T(i18n.Of(p), "ui.confirm.cancel"), Icon: "textures/ui/cancel", Run: func(*player.Player, *world.Tx) {}},
	)
}

// CreateToken in an Input title labels its submit button "생성하기" instead
// of "확인": a save slot is created there.
const CreateToken = "§d§f§e§r"

// Input asks for one line of text. label is the text over the input box; pass
// "" to ask under the title alone. back, if not nil, is the screen shown again
// when the player presses the back button of the control bar.
func Input(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string), back ...func(p *player.Player, tx *world.Tx)) {
	var b func(p *player.Player, tx *world.Tx)
	if len(back) > 0 {
		b = back[0]
	}
	oreInput(p, title, label, def, f, b)
}

// ItemIcon returns an icon that the screens draw as the item itself, with the
// client's item renderer, instead of a texture path: the item's runtime ID
// times 65536 (the renderer's "aux" value; the metadata is not used). It
// returns "" for an unregistered item.
func ItemIcon(it world.Item) string {
	aux, ok := slottext.ItemAux(it)
	if !ok {
		return ""
	}
	return strconv.FormatInt(aux, 10)
}

// SplitMenu shows the grid of Slots on the left with the page arrows under
// it, the Detail of the chosen slot on the right and the tabs over the
// window. A slot's Run shows the screen again with that slot chosen.
func SplitMenu(p *player.Player, title string, sp Split) {
	oreSplit(p, title, sp)
}
