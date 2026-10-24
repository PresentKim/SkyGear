package chestui

import (
	"maps"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/text/language"
)

// Localize, if set, gives an item of a window its name and lore in lang (a
// catalog language of i18n), or reports false to leave it as it is. Everyone
// who has a window of the same inventory open sees the same items, so the
// items hold one text; the window packets sent to each client carry the
// texts of that client's language instead.
var Localize func(lang string, it item.Stack) (name string, lore []string, ok bool)

// localizeWindow puts the texts of c's language into the contents of its
// window sent in pk.
func localizeWindow(c *dfhook.Conn, pk packet.Packet) {
	if Localize == nil {
		return
	}
	switch pk := pk.(type) {
	case *packet.InventoryContent:
		if w := openWindowOf(c, pk.WindowID); w != nil {
			lang := connLang(c)
			for slot := range pk.Content {
				localizeSlot(lang, w, slot, &pk.Content[slot].Stack)
			}
		}
	case *packet.InventorySlot:
		if w := openWindowOf(c, pk.WindowID); w != nil {
			localizeSlot(connLang(c), w, int(pk.Slot), &pk.NewItem.Stack)
		}
	}
}

// openWindowOf is c's window with the window ID, if it has one.
func openWindowOf(c *dfhook.Conn, windowID uint32) *window {
	cl := loadClient(c)
	if cl == nil {
		return nil
	}
	w := cl.win.Load()
	if w == nil || w.id() == 0 || uint32(w.id()) != windowID {
		return nil
	}
	return w
}

// id is the window ID of the window (0 until it is opened).
func (w *window) id() byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.windowID
}

// connLang is the catalog language of the client's language.
func connLang(c *dfhook.Conn) string {
	tag, _ := language.Parse(strings.Replace(c.ClientData().LanguageCode, "_", "-", 1))
	return i18n.Lang(tag)
}

// localizeSlot replaces the name and lore of the stack sent for a slot of w
// with Localize's for the item in that slot, when the stack sent is of that
// item.
func localizeSlot(lang string, w *window, slot int, st *protocol.ItemStack) {
	it, err := w.inv.Item(slot)
	if err != nil || it.Empty() {
		return
	}
	if rid, _, ok := world.ItemRuntimeID(it.Item()); !ok || rid != st.NetworkID {
		return
	}
	name, lore, ok := Localize(lang, it)
	if !ok {
		return
	}
	data := maps.Clone(st.NBTData)
	if data == nil {
		data = map[string]any{}
	}
	display := map[string]any{}
	if old, ok := data["display"].(map[string]any); ok {
		maps.Copy(display, old)
	}
	delete(display, "Name")
	delete(display, "Lore")
	if name != "" {
		display["Name"] = name
	}
	if len(lore) != 0 {
		display["Lore"] = lore
	}
	data["display"] = display
	st.NBTData = data
}
