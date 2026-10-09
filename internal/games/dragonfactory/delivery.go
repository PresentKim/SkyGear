package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// DeliveryBox takes items for the main mission from a hopper. Using it opens
// the mission form, whose button submits the needed items from the
// inventory.
type DeliveryBox struct{}

var deliveryHash = block.NextHash()

func (DeliveryBox) EncodeItem() (string, int16)           { return "dragonfactory:delivery_box", 0 }
func (DeliveryBox) EncodeBlock() (string, map[string]any) { return "dragonfactory:delivery_box", nil }
func (DeliveryBox) Hash() (uint64, uint64)                { return deliveryHash, 0 }
func (DeliveryBox) Model() world.BlockModel               { return model.Solid{} }
func (DeliveryBox) Name() string                          { return "납품함" }
func (DeliveryBox) Category() category.Category           { return category.Items() }
func (DeliveryBox) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.dragonfactory_delivery_box", Textures: map[string]customblock.Material{
		"*":   alphaMaterial("dragonfactory_delivery_box"),
		"lid": alphaMaterial("dragonfactory_delivery_box_lid"),
	}}
}
func (d DeliveryBox) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1, BlastResistance: 2,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(d, 1)} },
	}
}

// Activate opens the mission form; items are submitted with its button.
func (DeliveryBox) Activate(_ cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	if !s.debounced(p) {
		s.showMission(p)
	}
	return true
}

// InsertItem implements block.HopperInsertable: one item per transfer, for
// the main mission or else the running urgent mission (paid to the host).
func (DeliveryBox) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	inv := h.Inventory(tx, pos.Side(h.Facing.Opposite()))
	for slot, st := range inv.Slots() {
		if st.Empty() || !plain(st) {
			continue
		}
		if s.deliver(tx, nil, st.Item(), 1) == 1 || s.deliverUrgent(tx, nil, st.Item(), 1) == 1 {
			_ = inv.SetItem(slot, st.Grow(-1))
			return true
		}
	}
	return false
}

func init() {
	world.RegisterBlock(DeliveryBox{})
	world.RegisterItem(DeliveryBox{})
}
