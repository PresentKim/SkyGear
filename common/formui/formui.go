// Package formui opens classic server forms (ModalFormRequest) that the sgui
// pack's JSON UI draws, and takes their answers before the session does.
//
// Connections must pass sghook (sghook.Wrap).
package formui

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Button is a button of an action form: its text and an image path (a
// texture path or an item's runtime id times 65536, "" for none).
type Button struct {
	Text, Image string
}

// Menu is an action form.
type Menu struct {
	Title, Body string
	Buttons     []Button
}

// Input is a custom form with one text field and the label of its submit
// button.
type Input struct {
	Title, Label, Default, Submit string
}

// Form is a form open on one connection.
type Form struct {
	conn *sghook.Conn
	id   uint32
	data []byte
	busy int

	mu     sync.Mutex
	closed bool
	answer func(data []byte)
	cancel func()
}

var (
	formsMu sync.Mutex
	forms   = map[*sghook.Conn]*Form{}
	formIDs atomic.Uint32
)

func init() {
	formIDs.Store(1 << 30)
	sghook.OnClientPacket(func(c *sghook.Conn, pk packet.Packet) bool {
		r, ok := pk.(*packet.ModalFormResponse)
		if !ok || r.FormID < 1<<30 {
			return false
		}
		formsMu.Lock()
		f := forms[c]
		if f == nil || f.id != r.FormID {
			formsMu.Unlock()
			return true
		}
		if reason, ok := r.CancelReason.Value(); ok && reason == packet.ModalFormCancelReasonUserBusy && f.busy < busyRetries {
			f.busy++
			formsMu.Unlock()
			time.AfterFunc(busyWait, f.resend)
			return true
		}
		delete(forms, c)
		formsMu.Unlock()
		f.mu.Lock()
		was := f.closed
		f.closed = true
		f.mu.Unlock()
		if was {
			return true
		}
		data, ok := r.ResponseData.Value()
		if !ok || string(data) == "null" || len(data) == 0 {
			f.cancel()
			return true
		}
		f.answer(data)
		return true
	})
	sghook.OnClose(func(c *sghook.Conn) {
		formsMu.Lock()
		f := forms[c]
		delete(forms, c)
		formsMu.Unlock()
		if f != nil {
			f.mu.Lock()
			f.closed = true
			f.mu.Unlock()
		}
	})
}

// OpenMenu shows p the action form m. pressed runs with the index of the
// button pressed, cancelled when p closed the form. Both run on the
// connection's read goroutine. A form of this package p has open is replaced
// and answers nothing more.
func OpenMenu(p *player.Player, m Menu, pressed func(i int), cancelled func()) (*Form, error) {
	buttons := make([]map[string]any, 0, len(m.Buttons))
	for _, b := range m.Buttons {
		btn := map[string]any{"text": b.Text}
		if b.Image != "" {
			btn["image"] = map[string]any{"type": "path", "data": b.Image}
		}
		buttons = append(buttons, btn)
	}
	data, err := json.Marshal(map[string]any{"type": "form", "title": m.Title, "content": m.Body, "buttons": buttons})
	if err != nil {
		return nil, err
	}
	return open(p, data, func(b []byte) {
		var i int
		if json.Unmarshal(b, &i) == nil && i >= 0 && i < len(m.Buttons) {
			pressed(i)
			return
		}
		cancelled()
	}, cancelled)
}

// OpenInput shows p the custom form in. submitted runs with the text p sent,
// cancelled when p closed the form.
func OpenInput(p *player.Player, in Input, submitted func(value string), cancelled func()) (*Form, error) {
	data, err := json.Marshal(map[string]any{"type": "custom_form", "title": in.Title, "submit": in.Submit, "content": []map[string]any{
		{"type": "input", "text": in.Label, "placeholder": "", "default": in.Default},
	}})
	if err != nil {
		return nil, err
	}
	return open(p, data, func(b []byte) {
		var values []any
		if json.Unmarshal(b, &values) == nil && len(values) > 0 {
			if s, ok := values[0].(string); ok {
				submitted(s)
				return
			}
		}
		cancelled()
	}, cancelled)
}

func open(p *player.Player, data []byte, answer func([]byte), cancel func()) (*Form, error) {
	c := sghook.Of(p)
	if c == nil {
		return nil, fmt.Errorf("formui: %v has no hooked connection", p.Name())
	}
	f := &Form{conn: c, id: formIDs.Add(1), data: data, answer: answer, cancel: cancel}
	formsMu.Lock()
	old := forms[c]
	forms[c] = f
	formsMu.Unlock()
	if old != nil {
		old.mu.Lock()
		wasOpen := !old.closed
		old.closed = true
		old.mu.Unlock()
		if wasOpen {
			_ = c.WritePacket(&packet.ClientBoundCloseForm{})
		}
	}
	return f, c.WritePacket(&packet.ModalFormRequest{FormID: f.id, FormData: data})
}

// A client turns a form down as busy while it is still closing another
// screen, such as the chest window a press in the menu window came from. The
// form is sent again after busyWait, up to busyRetries times.
const (
	busyWait    = 100 * time.Millisecond
	busyRetries = 20
)

// resend sends f again if it is still the open form of its connection.
func (f *Form) resend() {
	formsMu.Lock()
	open := forms[f.conn] == f
	formsMu.Unlock()
	if open && !f.Closed() {
		_ = f.conn.WritePacket(&packet.ModalFormRequest{FormID: f.id, FormData: f.data})
	}
}

// Close closes the form on the client. Neither of its callbacks runs.
func (f *Form) Close() error {
	f.mu.Lock()
	was := f.closed
	f.closed = true
	f.mu.Unlock()
	formsMu.Lock()
	if forms[f.conn] == f {
		delete(forms, f.conn)
	}
	formsMu.Unlock()
	if was {
		return nil
	}
	return f.conn.WritePacket(&packet.ClientBoundCloseForm{})
}

// Closed reports whether the form was answered, closed or replaced.
func (f *Form) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}
