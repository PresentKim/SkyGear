package invui

import (
	"slices"
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

	mu     sync.Mutex
	open   bool
	quiet  bool
	layouts []string
	// pressed are the screen's slots of the window's slots (Place).
	pressed map[int]int
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
		w = &window{st: chestui.NewStorage(WindowSize, Marker)}
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
// open, or opens it again when the window does not draw the layout of the
// key in items (see Title). items are the screen's slots: the window holds
// the ones its layout uses at their places (Place), every other slot of the
// layout's region emptied and the other regions kept. press runs with the
// screen's slot when p presses one, closed when p closes the window. It must
// run in p's transaction.
func Show(p *player.Player, items []item.Stack, press func(p *player.Player, tx *world.Tx, slot int), closed func(p *player.Player, tx *world.Tx)) {
	layout := ""
	if len(items) > SlotKey {
		layout = LayoutOf(items[SlotKey].CustomName())
	}
	w := windowOf(p)
	w.mu.Lock()
	open, moved := w.open, w.open && !slices.Contains(w.layouts, layout)
	w.mu.Unlock()
	if moved {
		Close(p)
		open = false
	}
	slots := map[int]item.Stack{}
	first, end := Region(layout)
	for i := first; i < end; i++ {
		slots[i] = item.Stack{}
	}
	pressed := map[int]int{}
	for v, it := range items {
		if at, ok := Place(layout, v); ok {
			slots[at], pressed[at] = it, v
		}
	}
	w.mu.Lock()
	w.press, w.closed, w.pressed = press, closed, pressed
	if !open {
		w.layouts = LayoutsFor(layout)
	}
	layouts := w.layouts
	w.mu.Unlock()
	inv := w.st.Inventory()
	w.st.Batch(func() {
		for at, it := range slots {
			if old, _ := inv.Item(at); !same(old, it) {
				_ = inv.SetItem(at, it)
			}
		}
	})
	if open {
		return
	}
	w.mu.Lock()
	w.open, w.quiet = true, false
	w.mu.Unlock()
	w.st.SetTitle(Title(layouts))
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

// handler keeps every item in place: taking a slot's item (the quick move a
// slot button sends) or dropping it is a press of the slot.
type handler struct{ w *window }

func (h handler) HandleTake(ctx *inventory.Context, slot int, _ item.Stack) { h.press(ctx, slot) }
func (h handler) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack) { h.press(ctx, slot) }
func (h handler) HandlePlace(ctx *inventory.Context, _ int, _ item.Stack)   { ctx.Cancel() }

func (h handler) press(ctx *inventory.Context, slot int) {
	ctx.Cancel()
	p, ok := ctx.Val().(*player.Player)
	if !ok {
		return
	}
	p.H().Do(func(tx *world.Tx, e world.Entity) {
		h.w.mu.Lock()
		press := h.w.press
		v, ok := h.w.pressed[slot]
		h.w.mu.Unlock()
		if press != nil && ok {
			press(e.(*player.Player), tx, v)
		}
	})
}
