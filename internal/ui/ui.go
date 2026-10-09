// Package ui has small helpers to build forms whose buttons run closures.
package ui

import (
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

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
			o.Run(p, tx)
			return
		}
	}
}

// recoverForm keeps a bug in a form callback from crashing the server: the
// panic is logged and the player is told the action failed.
func recoverForm(p *player.Player) {
	if r := recover(); r != nil {
		slog.Error("form callback panicked", "player", p.Name(), "panic", r, "stack", string(debug.Stack()))
		p.Message("§c처리 중 오류가 발생했습니다.")
	}
}

// Menu shows a menu form whose buttons run their options. Labels must
// be unique within a menu.
func Menu(p *player.Player, title, body string, opts ...Option) {
	f := form.NewMenu(menu{opts: opts}, title).WithBody(body)
	for _, o := range opts {
		f = f.AddButton(form.NewButton(o.Label, o.Icon))
	}
	send(p, title, f)
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
)

// Tab returns a tab button of a themed form.
func Tab(label, icon string, selected bool, run func(p *player.Player, tx *world.Tx)) Option {
	token := TabToken
	if selected {
		token = TabOnToken
	}
	return Option{Label: token + label, Icon: icon, Run: run}
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

// Confirm asks a yes/no question as a themed menu (the vanilla modal form
// is not themed): the yes button runs f, "취소" or closing does nothing.
func Confirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	Menu(p, title, body,
		Option{Label: yes, Icon: "textures/ui/confirm", Run: f},
		Option{Label: "취소", Icon: "textures/ui/cancel", Run: func(*player.Player, *world.Tx) {}},
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
