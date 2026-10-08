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

// CardToken starts the title of a menu shown as a grid of icon-and-text
// cards, three per row (assets/resourcepack/dragonfactory/ui).
const CardToken = "§d§f§c§r"

// CardMenu shows a menu as a grid of cards with the icon above the label,
// so many options fit on one screen. Labels should be short (two lines).
func CardMenu(p *player.Player, title, body string, opts ...Option) {
	Menu(p, CardToken+title, body, opts...)
}

type confirm struct {
	Yes, No form.Button
	run     func(p *player.Player, tx *world.Tx)
}

func (confirm) Close(s form.Submitter, _ *world.Tx) { answered(s) }

func (c confirm) Submit(s form.Submitter, pressed form.Button, tx *world.Tx) {
	answered(s)
	p := s.(*player.Player)
	defer recoverForm(p)
	if pressed == c.Yes {
		c.run(p, tx)
	}
}

// Confirm asks a yes/no question and runs f on yes.
func Confirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	send(p, title, form.NewModal(confirm{Yes: form.NewButton(yes, ""), No: form.NewButton("취소", ""), run: f}, title).WithBody(body))
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
