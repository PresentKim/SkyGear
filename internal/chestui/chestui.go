// Package chestui shows virtual storages and menus as chest windows of any
// size. The windows are backed by an invisible client-only entity (the
// third_party/dragonfly patch), and the inventoryui resource pack reads the
// size from a prefix of the window title (like PocketMine's
// CustomSizedInvMenu).
//
// A Storage holds real items that players move in and out. A Menu is a
// Storage whose items are buttons: clicking one runs its action and nothing
// moves.
package chestui

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
)

// MaxRows is the number of rows a window shows before it scrolls.
const MaxRows = 6

// Title returns title with the prefix that makes the inventoryui resource
// pack show a window for size slots: §{rows}§{scroll} followed by nine §r.
func Title(size int, title string) string {
	rows := (size + 8) / 9
	scroll := 0
	if rows > MaxRows {
		rows, scroll = MaxRows, 1
	}
	if rows < 1 {
		rows = 1
	}
	return fmt.Sprintf("§%d§%d§r§r§r§r§r§r§r§r§r%s", rows, scroll, title)
}

// Storage is an inventory shown in chest windows. Changes to its inventory,
// by a viewer or by the server, are shown to every viewer.
type Storage struct {
	inv   *inventory.Inventory
	title string

	mu       sync.Mutex
	viewers  map[session.EntityContainerViewer]*world.EntityHandle
	onChange func(slot int, before, after item.Stack)
}

// NewStorage returns an empty storage with size slots. size must be at
// least 1; windows of more than 54 slots scroll.
func NewStorage(size int, title string) *Storage {
	s := &Storage{title: title, viewers: map[session.EntityContainerViewer]*world.EntityHandle{}}
	s.inv = inventory.New(size, s.slotChanged)
	return s
}

// Inventory returns the inventory of the storage. Its Handler may be set to
// validate moves; its SlotFunc must not be replaced (use OnChange).
func (s *Storage) Inventory() *inventory.Inventory { return s.inv }

// OnChange sets a function called after each slot change.
func (s *Storage) OnChange(f func(slot int, before, after item.Stack)) {
	s.mu.Lock()
	s.onChange = f
	s.mu.Unlock()
}

// SetTitle changes the title used by windows opened afterwards.
func (s *Storage) SetTitle(title string) {
	s.mu.Lock()
	s.title = title
	s.mu.Unlock()
}

// Open shows the storage to p. onClose, if not nil, is called in p's
// transaction when the window closes for any reason.
func (s *Storage) Open(p *player.Player, onClose func(p *player.Player, tx *world.Tx)) {
	v, h := p.ContainerViewer(), p.H()
	s.mu.Lock()
	title := s.title
	s.mu.Unlock()
	p.OpenEntityContainer(s.inv, Title(s.inv.Size(), title), func(tx *world.Tx) {
		s.mu.Lock()
		delete(s.viewers, v)
		s.mu.Unlock()
		if onClose == nil {
			return
		}
		if e, ok := h.Entity(tx); ok {
			onClose(e.(*player.Player), tx)
		}
	})
	s.mu.Lock()
	s.viewers[v] = h
	s.mu.Unlock()
}

// Viewers returns the handles of the players viewing the storage.
func (s *Storage) Viewers() []*world.EntityHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*world.EntityHandle, 0, len(s.viewers))
	for _, h := range s.viewers {
		out = append(out, h)
	}
	return out
}

// CloseAll closes the windows of every viewer. It may be called from any
// goroutine; each window is closed in its viewer's transaction.
func (s *Storage) CloseAll() {
	for _, h := range s.Viewers() {
		h.Do(func(tx *world.Tx, e world.Entity) {
			e.(*player.Player).CloseOpenedContainer()
		})
	}
}

func (s *Storage) slotChanged(slot int, before, after item.Stack) {
	s.mu.Lock()
	viewers := make([]session.EntityContainerViewer, 0, len(s.viewers))
	for v := range s.viewers {
		viewers = append(viewers, v)
	}
	f := s.onChange
	s.mu.Unlock()
	for _, v := range viewers {
		v.ViewSlotChange(s.inv, slot, after)
	}
	if f != nil {
		f(slot, before, after)
	}
}
