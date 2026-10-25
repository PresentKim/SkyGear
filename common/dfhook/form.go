package dfhook

import (
	"bytes"
	"reflect"
	"sync"
	"unsafe"

	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func init() {
	OnClientPacket(func(c *Conn, pk packet.Packet) bool {
		r, ok := pk.(*packet.ModalFormResponse)
		if !ok {
			return false
		}
		// "null" is a cancel too: Dragonfly would leave a menu's button
		// index at 0 and press the first button.
		if data, ok := r.ResponseData.Value(); ok && bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			r.ResponseData = protocol.Optional[[]byte]{}
		}
		// A response to a form that was dropped (more than 10 open) or
		// answered twice is ignored; Dragonfly would kick the player.
		return !formOpen(c.Session(), r.FormID)
	})
}

// formOpen reports whether s waits for a response to form id. It is true
// if s is unknown (Attach was not called).
func formOpen(s *session.Session, id uint32) bool {
	if s == nil {
		return true
	}
	h := unexported(fieldByName(reflect.ValueOf(s).Elem(), "handlers")).MapIndex(reflect.ValueOf(uint32(packet.IDModalFormResponse)))
	fh, ok := h.Interface().(*session.ModalFormResponseHandler)
	if !ok {
		return true
	}
	mu := field[sync.Mutex](fh, "mu")
	mu.Lock()
	defer mu.Unlock()
	_, ok = (*field[map[uint32]form.Form](fh, "forms"))[id]
	return ok
}

// unexported makes the value of an unexported field usable.
func unexported(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}
