package spike

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// ItemHolder is a custom entity that does nothing but hold an item in its
// main hand. The client renders the held item at the "rightItem" bone of the
// entity's geometry (resource pack), so unlike an item entity it does not
// spin or bob. It is used to display the content of a drawer.
type ItemHolder struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
}

type holderData struct{ item item.Stack }

// HolderConfig spawns an ItemHolder showing Item.
type HolderConfig struct{ Item item.Stack }

func (c HolderConfig) Apply(data *world.EntityData) { data.Data = &holderData{item: c.Item} }

// NewItemHolder creates an ItemHolder at pos facing yaw (degrees).
func NewItemHolder(it item.Stack, pos mgl64.Vec3, yaw float64) *world.EntityHandle {
	return world.EntitySpawnOpts{Position: pos, Rotation: cube.Rotation{yaw, 0}}.New(ItemHolderType, HolderConfig{Item: it})
}

func (h *ItemHolder) H() *world.EntityHandle  { return h.handle }
func (h *ItemHolder) Position() mgl64.Vec3    { return h.data.Pos }
func (h *ItemHolder) Rotation() cube.Rotation { return h.data.Rot }

// HeldItems implements item.Carrier, which makes viewers send the item.
func (h *ItemHolder) HeldItems() (item.Stack, item.Stack) {
	return h.data.Data.(*holderData).item, item.Stack{}
}

// SetItem changes the displayed item.
func (h *ItemHolder) SetItem(it item.Stack) {
	h.data.Data.(*holderData).item = it
	for _, v := range h.tx.Viewers(h.data.Pos) {
		v.ViewEntityItems(h)
	}
}

func (h *ItemHolder) Close() error {
	h.tx.RemoveEntity(h)
	return h.handle.Close()
}

// ItemHolderType is the world.EntityType of ItemHolder.
var ItemHolderType holderType

type holderType struct{}

func (holderType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &holderData{}
	}
	return &ItemHolder{tx: tx, handle: handle, data: data}
}
func (holderType) EncodeEntity() string                            { return "dragonfactory:item_holder" }
func (holderType) BBox(world.Entity) cube.BBox                     { return cube.BBox{} }
func (holderType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &holderData{} }

// EncodeNBT returns nil: holders are not saved, drawers re-create them.
func (holderType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// Entities is the default entity registry plus ItemHolderType.
var Entities = entity.DefaultRegistry.Config().New(append(entity.DefaultRegistry.Types(), ItemHolderType))
