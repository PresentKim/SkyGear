package invui

import (
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/chestui"
)

// window is the menu window of a player: its storage, what a press on a
// slot runs and what closing it runs.
type window struct {
	st *chestui.Storage

	mu       sync.Mutex
	open     bool
	quiet    bool
	dropping bool
	press  func(p *player.Player, tx *world.Tx, slot int)
	closed func(p *player.Player, tx *world.Tx)
}

var windows = struct {
	sync.Mutex
	m map[uuid.UUID]*window
}{m: map[uuid.UUID]*window{}}

func windowOf(p *player.Player) *window {
	windows.Lock()
	defer windows.Unlock()
	w := windows.m[p.UUID()]
	if w == nil {
		w = &window{st: chestui.NewStorage(Size, Marker)}
		w.st.Inventory().Handle(handler{w})
		w.st.CloseWithClient()
		windows.m[p.UUID()] = w
	}
	return w
}

// Forget drops the window of the player id once it left.
func Forget(id uuid.UUID) {
	windows.Lock()
	w := windows.m[id]
	delete(windows.m, id)
	windows.Unlock()
	if w != nil {
		w.st.Discard()
	}
}

// Show puts items in the slots of p's menu window and opens it if it is not
// open. press runs when p presses a slot, closed when p closes the window.
// Slots past the end of items are emptied. It must run in p's transaction.
func Show(p *player.Player, items []item.Stack, press func(p *player.Player, tx *world.Tx, slot int), closed func(p *player.Player, tx *world.Tx)) {
	w := windowOf(p)
	w.mu.Lock()
	w.press, w.closed = press, closed
	open := w.open
	w.mu.Unlock()
	inv := w.st.Inventory()
	for i := 0; i < Size; i++ {
		var it item.Stack
		if i < len(items) {
			it = items[i]
		}
		if !it.Empty() && it.Count() < 2 && it.MaxCount() >= 2 {
			it = it.Grow(2 - it.Count())
		}
		if old, _ := inv.Item(i); !same(old, it) {
			_ = inv.SetItem(i, it)
		}
	}
	if open {
		return
	}
	w.mu.Lock()
	w.open, w.quiet = true, false
	w.mu.Unlock()
	w.st.Open(p, func(p *player.Player, tx *world.Tx) {
		w.mu.Lock()
		w.open = false
		quiet, closed := w.quiet, w.closed
		w.mu.Unlock()
		if !quiet && closed != nil && p != nil {
			closed(p, tx)
		}
	})
}

// Close closes p's menu window without running its closed function. It
// must run in p's transaction.
func Close(p *player.Player) {
	windows.Lock()
	w := windows.m[p.UUID()]
	windows.Unlock()
	if w == nil {
		return
	}
	w.mu.Lock()
	open := w.open
	w.quiet = true
	w.mu.Unlock()
	if open {
		chestui.Close(p)
	}
}

// Open reports whether p has the menu window open.
func Open(p *player.Player) bool {
	windows.Lock()
	w := windows.m[p.UUID()]
	windows.Unlock()
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.open
}

func same(a, b item.Stack) bool {
	if a.Empty() || b.Empty() {
		return a.Empty() == b.Empty()
	}
	return a.Comparable(b) && a.Count() == b.Count() && a.CustomName() == b.CustomName()
}

// handler keeps every item in place: dropping one of a slot's items (what a
// slot button sends) or taking it is a press of the slot. The drop is let
// through and the player's drop handler cancels the item it would throw
// (SwallowDrop): a drop the inventory handler cancels fails the request, and
// the client then draws no slot of the window. Show puts two of each item that
// stacks, so the slot the client drops one from before the answer stays
// filled and its button drawn.
type handler struct{ w *window }

func (h handler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) {
	ctx.Cancel()
	h.press(ctx, slot)
}

func (h handler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) {
	h.w.mu.Lock()
	h.w.dropping = true
	h.w.mu.Unlock()
	h.press(ctx, slot)
}

func (h handler) HandlePlace(ctx *inventory.Context, _ int, _ item.Stack) { ctx.Cancel() }

// SwallowDrop reports whether the item p is dropping is a press of p's menu
// window, which its drop handler cancels.
func SwallowDrop(p *player.Player) bool {
	windows.Lock()
	w := windows.m[p.UUID()]
	windows.Unlock()
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	d := w.dropping
	w.dropping = false
	return d
}

func (h handler) press(ctx *inventory.Context, slot int) {
	p, ok := ctx.Val().(*player.Player)
	if !ok {
		return
	}
	p.H().Do(func(tx *world.Tx, e world.Entity) {
		h.w.mu.Lock()
		press := h.w.press
		h.w.dropping = false
		h.w.mu.Unlock()
		if press != nil {
			press(e.(*player.Player), tx, slot)
		}
	})
}
