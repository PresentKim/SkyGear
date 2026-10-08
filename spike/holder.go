package spike

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
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
	return NewItemHolderOfType(ItemHolderType, it, pos, yaw)
}

// NewItemHolderOfType creates an ItemHolder of a specific type.
func NewItemHolderOfType(t world.EntityType, it item.Stack, pos mgl64.Vec3, yaw float64) *world.EntityHandle {
	return world.EntitySpawnOpts{Position: pos, Rotation: cube.Rotation{yaw, 0}}.New(t, HolderConfig{Item: it})
}

// Display depths: how far in front of the block centre the holder stands so
// the held item sits on the face. Main hand items (flat sprites) and off hand
// cube blocks are drawn at different distances from the entity. Confirmed on
// a real client (docs/tech/drawer.md): between board columns 2 and 3.
const (
	MainHandDepth = 0.05
	OffHandDepth  = 0.025
)

// HolderPosition returns where a holder showing it on the face of the block
// at pos stands, and its yaw.
func HolderPosition(pos cube.Pos, face cube.Direction, it item.Stack, mainDepth, offDepth float64) (mgl64.Vec3, float64) {
	depth := mainDepth
	if IsCubeItem(it) {
		depth = offDepth
	}
	off := cube.Pos{}.Side(face.Face()).Vec3().Mul(depth)
	return pos.Vec3Centre().Add(off), yawOf(face)
}

// yawOf returns the entity yaw that looks out of the given block face.
func yawOf(face cube.Direction) float64 {
	switch face {
	case cube.North:
		return 180
	case cube.East:
		return -90
	case cube.West:
		return 90
	}
	return 0
}

func (h *ItemHolder) H() *world.EntityHandle  { return h.handle }
func (h *ItemHolder) Position() mgl64.Vec3    { return h.data.Pos }
func (h *ItemHolder) Rotation() cube.Rotation { return h.data.Rot }

// HeldItems implements item.Carrier, which makes viewers send the item.
// Full cube blocks are held in the off hand and everything else in the main
// hand: the client renders them differently, so each hand bone of the
// geometry carries its own transform (resource pack item_holder.geo.json).
func (h *ItemHolder) HeldItems() (item.Stack, item.Stack) {
	it := h.data.Data.(*holderData).item
	if IsCubeItem(it) {
		return item.Stack{}, it
	}
	return it, item.Stack{}
}

// IsCubeItem reports if it is a block item that the client renders as a cube.
func IsCubeItem(it item.Stack) bool {
	b, ok := it.Item().(world.Block)
	if !ok {
		return false
	}
	_, solid := b.Model().(model.Solid)
	return solid
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
var ItemHolderType = holderType{id: "dragonfactory:item_holder"}

type holderType struct{ id string }

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
var Entities = entity.DefaultRegistry.Config().New(entityTypes())

func entityTypes() []world.EntityType {
	return append(entity.DefaultRegistry.Types(), ItemHolderType)
}
