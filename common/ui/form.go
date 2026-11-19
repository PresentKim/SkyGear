package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/formui"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invui"
)

// The screens show in the menu window of invui (menu.go). Only an input,
// which needs the client's text field, is a server form: a custom form the
// sgui pack draws (assets/resourcepack/sgui/ui/sg_form.json), routed by the
// title prefix "sg:in:".

// inkSwap is what the colour codes of text become on the grey window: light
// colours become dark ones.
var inkSwap = strings.NewReplacer("§f", "§0", "§7", "§8", "§8", "§7", "§e", "§1", "§6", "§4", "§a", "§2", "§b", "§3", "§c", "§4", "§d", "§5")

func ink(s string) string { return inkSwap.Replace(i18n.Plain(s)) }

// formScreen is what a player has open: a form or the menu window, and what
// counts the screens shown so a run that shows none is known.
type formScreen struct {
	form *formui.Form
	gen  int
	key  string
	// loading is when the loading screen shown came up, zero for any other
	// screen (CloseLoading).
	loading time.Time
	// menu reports that the screen shows in the menu window (openMenu), of
	// the player of h.
	menu bool
	h    *world.EntityHandle
}

// showing reports whether st shows a screen: a form or the menu window.
func (st *formScreen) showing() bool {
	return st != nil && (st.menu || st.form != nil && !st.form.Closed())
}

var formScreens = struct {
	sync.Mutex
	m map[uuid.UUID]*formScreen
}{m: map[uuid.UUID]*formScreen{}}

func formOpen(p *player.Player) bool {
	formScreens.Lock()
	defer formScreens.Unlock()
	return formScreens.m[p.UUID()].showing()
}

func formKey(p *player.Player) (string, bool) {
	formScreens.Lock()
	defer formScreens.Unlock()
	st := formScreens.m[p.UUID()]
	if !st.showing() {
		return "", false
	}
	return st.key, true
}

func forgetForm(id uuid.UUID) {
	formScreens.Lock()
	delete(formScreens.m, id)
	formScreens.Unlock()
	invui.Forget(id)
}

// openInput shows p the input in as a custom form. Submitting runs its submit
// button's run with the text, then Idle when that showed no other screen;
// closing it runs its back button's run, or the layout's close run, or Idle.
func openInput(p *player.Player, l *oreLayout, in inputData) {
	invui.Close(p)
	formScreens.Lock()
	st := formScreens.m[p.UUID()]
	if st == nil {
		st = &formScreen{}
		formScreens.m[p.UUID()] = st
	}
	st.gen++
	gen := st.gen
	st.key, st.menu, st.loading = l.screenKey, false, time.Time{}
	formScreens.Unlock()

	current := func() bool {
		formScreens.Lock()
		defer formScreens.Unlock()
		return st.gen == gen
	}
	h := p.H()
	run := func(f func(p *player.Player, tx *world.Tx), closed, idle bool) {
		h.Do(func(tx *world.Tx, e world.Entity) {
			p := e.(*player.Player)
			if !current() {
				return
			}
			if f != nil {
				func() {
					defer recoverForm(p)
					f(p, tx)
				}()
			}
			if idle && current() && Idle != nil {
				defer recoverForm(p)
				Idle(p, tx, "", closed)
			}
		})
	}
	cancelled := func() { run(l.close, true, l.close == nil) }
	for _, b := range in.Chrome.Bar {
		if back := l.runs[b.ID]; back != nil && b.Icon == "tex:textures/ui/sg/chevron_left" && l.close == nil {
			cancelled = func() { run(back, false, false) }
		}
	}
	title := "sg:in:" + i18n.Plain(in.Chrome.Title)
	submit := l.runs[in.Submit.ID]
	f, err := formui.OpenInput(p, formui.Input{Title: title, Label: ink(in.Label), Default: in.Value, Submit: in.Submit.Label}, func(value string) {
		formScreens.Lock()
		l.values[in.Field] = value
		formScreens.Unlock()
		run(submit, false, true)
	}, cancelled)
	formScreens.Lock()
	if st.gen == gen && err == nil {
		st.form = f
	}
	formScreens.Unlock()
}

// closeForm closes the form p has open without running its close run.
func closeForm(p *player.Player) {
	formScreens.Lock()
	defer formScreens.Unlock()
	if st := formScreens.m[p.UUID()]; st != nil && st.form != nil {
		_ = st.form.Close()
		st.form = nil
	}
}
