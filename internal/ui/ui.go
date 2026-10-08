// Package ui has small helpers to build forms whose buttons run closures.
package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
)

// Option is a menu button and the action it runs.
type Option struct {
	Label string
	Run   func(p *player.Player, tx *world.Tx)
}

type menu struct{ opts []Option }

func (m menu) Submit(s form.Submitter, pressed form.Button, tx *world.Tx) {
	for _, o := range m.opts {
		if o.Label == pressed.Text {
			o.Run(s.(*player.Player), tx)
			return
		}
	}
}

// Menu shows a menu form whose buttons run their options. Labels must
// be unique within a menu.
func Menu(p *player.Player, title, body string, opts ...Option) {
	f := form.NewMenu(menu{opts: opts}, title).WithBody(body)
	for _, o := range opts {
		f = f.AddButton(form.NewButton(o.Label, ""))
	}
	p.SendForm(f)
}

type confirm struct {
	Yes, No form.Button
	run     func(p *player.Player, tx *world.Tx)
}

func (c confirm) Submit(s form.Submitter, pressed form.Button, tx *world.Tx) {
	if pressed == c.Yes {
		c.run(s.(*player.Player), tx)
	}
}

// Confirm asks a yes/no question and runs f on yes.
func Confirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	p.SendForm(form.NewModal(confirm{Yes: form.NewButton(yes, ""), No: form.NewButton("취소", ""), run: f}, title).WithBody(body))
}

type textInput struct {
	Value form.Input
	run   func(p *player.Player, tx *world.Tx, value string)
}

func (t textInput) Submit(s form.Submitter, tx *world.Tx) {
	t.run(s.(*player.Player), tx, t.Value.Value())
}

// Input asks for one line of text.
func Input(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string)) {
	p.SendForm(form.New(textInput{Value: form.NewInput(label, def, ""), run: f}, title))
}
