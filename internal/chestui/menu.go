package chestui

import (
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Button is an item in a Menu that runs Run when clicked.
type Button struct {
	Item item.Stack
	Run  func(p *player.Player, tx *world.Tx)
}

// Menu is a chest window of buttons. Items in it cannot be taken or placed:
// taking one (clicking it) runs the button instead. A Menu may be shown to
// several players and changed while open.
type Menu struct {
	*Storage

	mu      sync.Mutex
	buttons map[int]Button
}

// NewMenu returns an empty menu with size slots.
func NewMenu(size int, title string) *Menu {
	m := &Menu{Storage: NewStorage(size, title), buttons: map[int]Button{}}
	m.inv.Handle(menuHandler{m: m})
	return m
}

// Set puts b in slot, replacing what was there. Viewers see it at once.
func (m *Menu) Set(slot int, b Button) {
	m.mu.Lock()
	m.buttons[slot] = b
	m.mu.Unlock()
	_ = m.inv.SetItem(slot, b.Item)
}

// Clear removes every button.
func (m *Menu) Clear() {
	m.mu.Lock()
	m.buttons = map[int]Button{}
	m.mu.Unlock()
	m.inv.Clear()
}

func (m *Menu) button(slot int) (Button, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.buttons[slot]
	return b, ok
}

type menuHandler struct{ m *Menu }

func (h menuHandler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	ctx.Cancel()
	b, ok := h.m.button(slot)
	if !ok || b.Run == nil {
		return
	}
	p, ok := ctx.Val().(*player.Player)
	if !ok {
		return
	}
	// Run after the item request is answered: the action may close this
	// window or open another one.
	p.H().Do(func(tx *world.Tx, e world.Entity) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("chestui: button panicked", "panic", r, "stack", string(debug.Stack()))
			}
		}()
		b.Run(e.(*player.Player), tx)
	})
}

func (menuHandler) HandlePlace(ctx *inventory.Context, _ int, _ item.Stack) { ctx.Cancel() }
func (menuHandler) HandleDrop(ctx *inventory.Context, _ int, _ item.Stack)  { ctx.Cancel() }

// formDelay is how long CloseThen waits after closing a window. A form sent
// while the client is still closing a container may not be shown.
const formDelay = 250 * time.Millisecond

// CloseThen closes p's chest window and runs f in p's transaction shortly
// after, for buttons that continue in a form.
func CloseThen(p *player.Player, f func(p *player.Player, tx *world.Tx)) {
	p.CloseOpenedContainer()
	p.H().DoAfter(formDelay, func(tx *world.Tx, e world.Entity) {
		f(e.(*player.Player), tx)
	})
}

// Lore is a convenience to give a button item a name and lore lines.
func Lore(it item.Stack, name string, lines ...string) item.Stack {
	return it.WithCustomName("§r" + name).WithLore(lines...)
}
