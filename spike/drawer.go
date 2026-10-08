package spike

import (
	"image"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
)

// Drawer is a minimal single item storage block used to verify hopper
// integration: it accepts items from a hopper above/beside it and gives
// items to a hopper below it, one at a time like vanilla containers.
type Drawer struct {
	// Item is the stored item type (count 1). Empty means unlocked and empty.
	Item  item.Stack
	Count int64
	Cap   int64
}

var drawerHash = block.NextHash()

func (Drawer) EncodeItem() (string, int16)           { return "dragonfactory:spike_drawer", 0 }
func (Drawer) EncodeBlock() (string, map[string]any) { return "dragonfactory:spike_drawer", nil }
func (Drawer) Hash() (uint64, uint64)                { return drawerHash, 0 }
func (Drawer) Model() world.BlockModel               { return model.Solid{} }
func (Drawer) Properties() customblock.Properties    { return Machine{}.Properties() }
func (Drawer) Name() string                          { return "Spike Drawer" }
func (Drawer) Texture() image.Image                  { return texture() }
func (Drawer) Category() category.Category           { return category.Items() }

// InsertItem implements block.HopperInsertable.
func (d Drawer) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	for slot, s := range h.Inventory(tx, pos).Slots() {
		if s.Empty() || d.Count >= d.Cap {
			continue
		}
		if !d.Item.Empty() && !d.Item.Comparable(s) {
			continue
		}
		if d.Item.Empty() {
			d.Item = s.Grow(1 - s.Count())
		}
		d.Count++
		_ = h.Inventory(tx, pos).SetItem(slot, s.Grow(-1))
		tx.SetBlock(pos, d, nil)
		return true
	}
	return false
}

// ExtractItem implements block.HopperExtractable.
func (d Drawer) ExtractItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	if d.Count == 0 {
		return false
	}
	hopperPos := pos.Side(cube.FaceDown)
	if _, err := h.Inventory(tx, hopperPos).AddItem(d.Item); err != nil {
		return false
	}
	d.Count--
	tx.SetBlock(pos, d, nil)
	return true
}

func (d Drawer) EncodeNBT() map[string]any {
	m := map[string]any{"id": "SpikeDrawer", "Count": d.Count, "Cap": d.Cap}
	if !d.Item.Empty() {
		m["Item"] = item.WriteNBT(d.Item, true)
	}
	return m
}

func (d Drawer) DecodeNBT(data map[string]any) any {
	d.Count, _ = data["Count"].(int64)
	d.Cap, _ = data["Cap"].(int64)
	if it, ok := data["Item"].(map[string]any); ok {
		d.Item = item.ReadNBT(it, nil)
	}
	return d
}

func init() {
	world.RegisterBlock(Drawer{})
	world.RegisterItem(Drawer{})
}
