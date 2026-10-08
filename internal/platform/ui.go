package platform

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
)

// option is a menu button and the action it runs.
type option struct {
	label string
	run   func(p *player.Player, tx *world.Tx)
}

type menu struct{ opts []option }

func (m menu) Submit(s form.Submitter, pressed form.Button, tx *world.Tx) {
	for _, o := range m.opts {
		if o.label == pressed.Text {
			o.run(s.(*player.Player), tx)
			return
		}
	}
}

// sendMenu shows a menu form whose buttons run their options. Labels must
// be unique within a menu.
func sendMenu(p *player.Player, title, body string, opts ...option) {
	f := form.NewMenu(menu{opts: opts}, title).WithBody(body)
	for _, o := range opts {
		f = f.AddButton(form.NewButton(o.label, ""))
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

// sendConfirm asks a yes/no question and runs f on yes.
func sendConfirm(p *player.Player, title, body, yes string, f func(p *player.Player, tx *world.Tx)) {
	p.SendForm(form.NewModal(confirm{Yes: form.NewButton(yes, ""), No: form.NewButton("취소", ""), run: f}, title).WithBody(body))
}

type textInput struct {
	Value form.Input
	run   func(p *player.Player, tx *world.Tx, value string)
}

func (t textInput) Submit(s form.Submitter, tx *world.Tx) {
	t.run(s.(*player.Player), tx, t.Value.Value())
}

// sendInput asks for one line of text.
func sendInput(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string)) {
	p.SendForm(form.New(textInput{Value: form.NewInput(label, def, ""), run: f}, title))
}
