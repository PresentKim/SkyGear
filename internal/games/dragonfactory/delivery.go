package dragonfactory

import (
	"image"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// DeliveryBox takes items for the main mission: from a hopper, from the
// held stack when used, or the whole inventory when used while sneaking.
type DeliveryBox struct{}

var deliveryHash = block.NextHash()
var deliveryTexture = frameTexture(rgb(70, 45, 20), rgb(150, 105, 55), rgb(240, 200, 60), 4)

func (DeliveryBox) EncodeItem() (string, int16)           { return "dragonfactory:delivery_box", 0 }
func (DeliveryBox) EncodeBlock() (string, map[string]any) { return "dragonfactory:delivery_box", nil }
func (DeliveryBox) Hash() (uint64, uint64)                { return deliveryHash, 0 }
func (DeliveryBox) Model() world.BlockModel               { return model.Solid{} }
func (DeliveryBox) Name() string                          { return "납품함" }
func (DeliveryBox) Category() category.Category           { return category.Items() }
func (DeliveryBox) Geometry() []byte                      { return nil }
func (DeliveryBox) Texture() image.Image                  { return deliveryTexture }
func (DeliveryBox) Textures() map[string]image.Image {
	return map[string]image.Image{"dragonfactory_delivery_box": deliveryTexture}
}
func (DeliveryBox) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*": customblock.NewMaterial("dragonfactory_delivery_box", customblock.OpaqueRenderMethod()),
	}}
}
func (d DeliveryBox) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2, BlastResistance: 2,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(t item.Tool) bool { return t.ToolType() == item.TypeAxe },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(d, 1)} },
	}
}

// Activate delivers the held stack, or the inventory when sneaking.
func (DeliveryBox) Activate(_ cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	if p.Sneaking() {
		if s.debounced(p) {
			return true
		}
		if n := s.deliverInventory(p, tx); n > 0 {
			p.Messagef("§a%d개를 납품했습니다.", n)
		} else {
			p.SendTip("§7납품할 아이템이 없습니다.")
		}
		return true
	}
	held, off := p.HeldItems()
	if held.Empty() {
		if !s.debounced(p) {
			s.showMission(p)
		}
		return true
	}
	if n := s.deliver(tx, p, held.Item(), held.Count()); n > 0 {
		p.SetHeldItems(held.Grow(-n), off)
		p.SendTip("§a" + itemName(held) + " " + formatInt(int64(n)) + "개 납품")
	} else {
		p.SendTip("§7이 아이템은 지금 미션에 필요하지 않습니다.")
	}
	return true
}

// InsertItem implements block.HopperInsertable: one needed item per transfer.
func (DeliveryBox) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	inv := h.Inventory(tx, pos.Side(h.Facing.Opposite()))
	for slot, st := range inv.Slots() {
		if st.Empty() {
			continue
		}
		if s.deliver(tx, nil, st.Item(), 1) == 1 {
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
