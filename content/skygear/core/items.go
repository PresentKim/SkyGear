package core

import (
	"bytes"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// GiveItem gives p an item, dropping what does not fit at pos.
func (s *Session) GiveItem(p *player.Player, tx *world.Tx, pos mgl64.Vec3, it item.Stack) {
	if n, _ := p.Inventory().AddItem(it); n < it.Count() {
		s.DropAt(tx, pos, it.Grow(-n))
	}
}

// CountPlain counts the plain items of it in an inventory.
func CountPlain(inv *inventory.Inventory, it world.Item) int {
	n, key := 0, naming.ItemKey(it)
	for _, st := range inv.Slots() {
		if !st.Empty() && Plain(st) && naming.ItemKey(st.Item()) == key {
			n += st.Count()
		}
	}
	return n
}

// TakePlain takes count plain items of it from an inventory if it holds
// that many.
func TakePlain(inv *inventory.Inventory, it world.Item, count int) bool {
	if CountPlain(inv, it) < count {
		return false
	}
	key := naming.ItemKey(it)
	for slot, st := range inv.Slots() {
		if count == 0 {
			break
		}
		if st.Empty() || !Plain(st) || naming.ItemKey(st.Item()) != key {
			continue
		}
		n := min(count, st.Count())
		_ = inv.SetItem(slot, st.Grow(-n))
		count -= n
	}
	return true
}

// Abs is the absolute value of v.
func Abs[T int | int32](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

// EncodeItem is the little endian NBT of one item of s, as saves keep
// items.
func EncodeItem(s item.Stack) []byte {
	var b bytes.Buffer
	_ = nbt.NewEncoderWithEncoding(&b, nbt.LittleEndian).Encode(item.WriteNBT(s.Grow(1-s.Count()), true))
	return b.Bytes()
}

// DecodeItem is the stack EncodeItem wrote (empty if b cannot be read).
func DecodeItem(b []byte) item.Stack {
	var m map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian).Decode(&m); err != nil {
		return item.Stack{}
	}
	return item.ReadNBT(m, nil)
}

// Plain reports whether a stack is a plain item: no added NBT (custom
// name, lore, values such as a time-limited item's expiry). Research,
// machine recipes and prices ask for plain items; anything else is a
// different item to them.
func Plain(st item.Stack) bool { return st.Comparable(item.NewStack(st.Item(), 1)) }
