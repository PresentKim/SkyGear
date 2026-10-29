// Package oreui opens server-driven DDUI screens that the oreui pack's
// renderer (hbui/oui-ddui.js) draws in Ore UI from a layout tree.
//
// The vanilla root minecraft:custom_form reads data store "minecraft",
// property custom_form_data_<instance id>; the renderer reads these extra
// fields instead of drawing the vanilla form:
//
//	custom_form_data_1.oui        JSON {"form": nonce, "root": layout, "settings": [Setting]}
//	custom_form_data_1.oui_event  JSON written by the client: {"seq", "id", "kind", "value"}
//	oui_vars_1_<n>.v              JSON {"f": nonce, "s": seq, "p": patch}: changed
//	                              variables, rotating over varsSlots properties
//
// Connections must pass sghook (sghook.Wrap), which takes the client's DDUI
// packets the session does not handle.
package oreui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	store    = "minecraft"
	screen   = "minecraft:custom_form"
	instance = 1
	property = "custom_form_data_1"
	// varsSlots is how many properties the variable patches rotate through.
	// The client keeps only the newest value of a property in a frame, so two
	// patches to one property lose the first; the renderer reads every slot
	// and applies the patches in seq order.
	varsSlots = 8
)

// Event is one interaction the renderer reports: Kind "click" with the ID of
// the pressed element and the text of the screen's input boxes by their IDs
// (Inputs), "change" with an input's Value, or "key" with ID "close" for Esc.
type Event struct {
	Seq    int               `json:"seq"`
	ID     string            `json:"id"`
	Kind   string            `json:"kind"`
	Value  any               `json:"value"`
	Inputs map[string]string `json:"inputs"`
}

// Form is a DDUI screen open on one connection.
type Form struct {
	conn *sghook.Conn
	// id is the form's own ID: the client reports the closing of a replaced
	// form after the next one opened, which must not close the next one. The
	// data instance (custom_form_data_1) stays the same for all.
	id uint32

	mu      sync.Mutex
	closed  bool
	onEvent func(Event)
	onClose func()
	pending map[string]any
	seq     int
	// nonce tells this form's layout and patches from those of a form before
	// it in the same properties.
	nonce string
}

// Setting is a client setting the renderer keeps at Value while the page is
// open (SetClientSettings): ID as the game's settings screen names it
// (video.ui_profile), Kind "bool", "option", "number" or "string".
type Setting struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

var (
	settingsMu sync.Mutex
	settings   = map[*sghook.Conn][]Setting{}
)

// SetClientSettings makes the renderer of p's client keep s: it writes each
// setting when it differs and again whenever the player changes it. The
// settings go with the layout of the next screen opened; they stay in the
// client's options after p leaves.
func SetClientSettings(p *player.Player, s []Setting) {
	c := sghook.Of(p)
	if c == nil {
		return
	}
	settingsMu.Lock()
	settings[c] = s
	settingsMu.Unlock()
}

func settingsOf(c *sghook.Conn) []Setting {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return settings[c]
}

var (
	formsMu sync.Mutex
	forms   = map[*sghook.Conn]*Form{}
	// updateCount numbers every data store write: the client applies a change
	// only when its count is above the last one of the property.
	updateCount atomic.Uint32
	formIDs     atomic.Uint32
)

func init() {
	sghook.OnClientPacket(func(c *sghook.Conn, pk packet.Packet) bool {
		switch pk := pk.(type) {
		case *packet.ServerBoundDataStore:
			f := formOf(c)
			if f == nil || pk.Update.Property != property || f.onEvent == nil {
				return true
			}
			if pk.Update.Path == "oui_event" && pk.Update.ControlType == protocol.DataStoreControlString {
				var ev Event
				if json.Unmarshal([]byte(pk.Update.StringValue), &ev) == nil {
					f.onEvent(ev)
				}
				return true
			}
			// Other writes come from the vanilla custom_form root under the
			// hidden form: its close button, which the client may press for
			// the back button of a phone. That is Esc.
			slog.Info("ore ui vanilla form write", "player", c.IdentityData().DisplayName, "path", pk.Update.Path, "type", pk.Update.ControlType, "string", pk.Update.StringValue, "bool", pk.Update.BoolValue)
			if strings.Contains(strings.ToLower(pk.Update.Path), "close") {
				f.onEvent(Event{ID: "close", Kind: "key"})
			}
			return true
		case *packet.ServerBoundDataDrivenScreenClosed:
			formsMu.Lock()
			f := forms[c]
			if f != nil && pk.FormID == f.id {
				delete(forms, c)
			}
			formsMu.Unlock()
			if f != nil && pk.FormID == f.id {
				// The reason tells how the client closed it (a phone's back
				// button, the client's own menu, ...).
				slog.Info("ore ui screen closed by client", "player", c.IdentityData().DisplayName, "reason", pk.CloseReason)
				f.closedByClient()
			}
			return true
		}
		return false
	})
	sghook.OnClose(func(c *sghook.Conn) {
		formsMu.Lock()
		f := forms[c]
		delete(forms, c)
		formsMu.Unlock()
		settingsMu.Lock()
		delete(settings, c)
		settingsMu.Unlock()
		if f != nil {
			f.closedByClient()
		}
	})
}

func formOf(c *sghook.Conn) *Form {
	formsMu.Lock()
	defer formsMu.Unlock()
	return forms[c]
}

// Open shows p a new screen with layout, replacing a screen of this package
// p has open (whose onClose does not run). onEvent runs on the connection's
// read goroutine for each event, onClose when the player closes the screen or
// leaves.
func Open(p *player.Player, layout any, onEvent func(Event), onClose func()) (*Form, error) {
	c := sghook.Of(p)
	if c == nil {
		return nil, fmt.Errorf("oreui: %v has no hooked connection", p.Name())
	}
	f := &Form{conn: c, id: formIDs.Add(1), onEvent: onEvent, onClose: onClose, nonce: fmt.Sprint(time.Now().UnixNano())}
	formsMu.Lock()
	if old := forms[c]; old != nil {
		old.mu.Lock()
		old.closed = true
		old.mu.Unlock()
	}
	forms[c] = f
	formsMu.Unlock()
	if err := f.SetLayout(layout); err != nil {
		return nil, err
	}
	return f, c.WritePacket(&packet.ClientBoundDataDrivenUIShowScreen{ScreenID: screen, FormID: f.id, DataInstanceID: protocol.Option(uint32(instance))})
}

// SetLayout replaces the layout tree of the screen. It resends the whole form
// property with the vanilla fields the custom_form root reads.
func (f *Form) SetLayout(layout any) error {
	b, err := json.Marshal(map[string]any{"form": f.nonce, "root": layout, "settings": settingsOf(f.conn)})
	if err != nil {
		return err
	}
	return f.change(property, protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeMap, MapValue: []protocol.DataStoreMapEntry{
		{Key: "title", Value: str("")},
		{Key: "closeButton", Value: protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeMap, MapValue: []protocol.DataStoreMapEntry{
			{Key: "label", Value: str("")},
			{Key: "visible", Value: protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeBool, BoolValue: true}},
		}}},
		{Key: "layout", Value: protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeList}},
		{Key: "oui", Value: str(string(b))},
		{Key: "oui_event", Value: str("")},
	}})
}

// SetVars sends changed variables of the layout (its bindings) in small
// properties of their own, so frequent updates do not resend the layout.
// Calls within 10ms are merged into one send.
func (f *Form) SetVars(vars map[string]any) {
	f.mu.Lock()
	first := f.pending == nil
	if first {
		f.pending = map[string]any{}
	}
	for k, v := range vars {
		f.pending[k] = v
	}
	f.mu.Unlock()
	if first {
		time.AfterFunc(10*time.Millisecond, f.flushVars)
	}
}

func (f *Form) flushVars() {
	f.mu.Lock()
	patch := f.pending
	f.pending = nil
	f.seq++
	seq := f.seq
	f.mu.Unlock()
	b, err := json.Marshal(map[string]any{"f": f.nonce, "s": seq, "p": patch})
	if err != nil {
		return
	}
	_ = f.change(fmt.Sprintf("oui_vars_%d_%d", instance, seq%varsSlots), protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeMap,
		MapValue: []protocol.DataStoreMapEntry{{Key: "v", Value: str(string(b))}}})
}

// Close closes the screen on the client. onClose does not run.
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
	return f.conn.WritePacket(&packet.ClientBoundDataDrivenUICloseScreen{FormID: protocol.Option(f.id)})
}

// Closed reports whether the form was closed by either side or replaced.
func (f *Form) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func (f *Form) closedByClient() {
	f.mu.Lock()
	was := f.closed
	f.closed = true
	f.mu.Unlock()
	if !was && f.onClose != nil {
		f.onClose()
	}
}

func (f *Form) change(prop string, v protocol.DataStorePropertyValue) error {
	if f.Closed() {
		return nil
	}
	return f.conn.WritePacket(&packet.ClientBoundDataStore{Updates: []protocol.DataStoreChangeEntry{{
		ChangeType: protocol.DataStoreChangeTypeChange,
		Change:     protocol.DataStoreChange{DataStoreName: store, Property: prop, UpdateCount: updateCount.Add(1), NewValue: v},
	}}})
}

func str(s string) protocol.DataStorePropertyValue {
	return protocol.DataStorePropertyValue{Type: protocol.DataStorePropertyTypeString, StringValue: s}
}
