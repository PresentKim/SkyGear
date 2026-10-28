package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// ItemHolder is the display entity of drawers: it holds
// an item so the client renders it flat on the drawer face, and/or shows a
// name tag. Holders belong to a drawer position and remove themselves when
// that drawer is gone or the session does not know them (e.g. after a
// reload, since every entity is saved with the world).
type ItemHolder struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
}

type holderData struct {
	item  item.Stack
	owner cube.Pos
	pose  holderPose
}

// holderConfig spawns an ItemHolder for the drawer at owner.
type holderConfig struct {
	item  item.Stack
	owner cube.Pos
	pose  holderPose
}

func (c holderConfig) Apply(data *world.EntityData) {
	data.Data = &holderData{item: c.item, owner: c.owner, pose: c.pose}
}

// holderPose is a hand-equipped item pose tried with /sgdebug holderpose,
// sent as the variant (rotation) and mark variant (scale, offset) that
// animations/item_holder.animation.json decodes. Zero is the pack's own pose.
type holderPose struct {
	variant, mark int32
}

// newHolderPose packs rotations in degrees (5 degree steps, -180 to 180), a
// scale in percent (1 to 255) and offsets in pixels (-16 to 15). Molang
// numbers are 32-bit floats, so both values stay below 2^24.
func newHolderPose(rx, ry, rz, scale, dx, dy, dz int) holderPose {
	angle := func(d int) int32 { return int32((max(-180, min(180, d)) + 180 + 2) / 5) }
	offset := func(px int) int32 { return int32(max(-16, min(15, px)) + 16) }
	return holderPose{
		variant: angle(rx) + 73*angle(ry) + 73*73*angle(rz),
		mark:    int32(max(1, min(255, scale))) + 256*offset(dx) + 256*32*offset(dy) + 256*32*32*offset(dz),
	}
}

// Display depths in front of the block centre, confirmed on a real client:
// main hand items and off hand cube blocks differ.
const (
	mainHandDepth = 0.05
	offHandDepth  = 0.025
)

func (h *ItemHolder) H() *world.EntityHandle  { return h.handle }
func (h *ItemHolder) Position() mgl64.Vec3    { return h.data.Pos }
func (h *ItemHolder) Rotation() cube.Rotation { return h.data.Rot }
func (h *ItemHolder) NameTag() string         { return h.data.Name }
func (h *ItemHolder) AlwaysShowNameTag() bool { return h.data.AlwaysShowNameTag }
func (h *ItemHolder) Variant() int32          { return h.data.Data.(*holderData).pose.variant }
func (h *ItemHolder) MarkVariant() int32      { return h.data.Data.(*holderData).pose.mark }

// HeldItems implements item.Carrier: cube blocks in the off hand, everything
// else in the main hand (each hand bone has its own transform).
func (h *ItemHolder) HeldItems() (item.Stack, item.Stack) {
	it := h.data.Data.(*holderData).item
	if isCubeItem(it) {
		return item.Stack{}, it
	}
	return it, item.Stack{}
}

// Tick removes holders that no longer belong to a drawer display.
func (h *ItemHolder) Tick(tx *world.Tx, _ int64) {
	d := h.data.Data.(*holderData)
	s, ok := activeGame.sessionOf(tx.World())
	_, isDrawer := tx.Block(d.owner).(Drawer)
	if ok && isDrawer && s.ownsDisplay(d.owner, h.handle) {
		return
	}
	if ok && !isDrawer && s.ownsDisplay(d.owner, h.handle) {
		// The drawer was removed by other means: drop its count text too.
		s.removeDisplay(tx, d.owner)
		return
	}
	_ = h.Close()
}

func (h *ItemHolder) Close() error {
	h.tx.RemoveEntity(h)
	return h.handle.Close()
}

func isCubeItem(it item.Stack) bool {
	b, ok := it.Item().(world.Block)
	if !ok {
		return false
	}
	_, solid := b.Model().(model.Solid)
	return solid
}

// holderPosition returns where a holder showing it on the face of the block
// at pos stands, and its yaw.
func holderPosition(pos cube.Pos, face cube.Direction, it item.Stack) (mgl64.Vec3, float64) {
	depth := mainHandDepth
	if isCubeItem(it) {
		depth = offHandDepth
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

// HolderType is the world.EntityType of ItemHolder. Its identifier matches
// the client entity in assets/resourcepack/skygear.
var HolderType holderType

type holderType struct{}

func (holderType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &holderData{}
	}
	return &ItemHolder{tx: tx, handle: handle, data: data}
}
func (holderType) EncodeEntity() string                            { return "skygear:item_holder" }
func (holderType) BBox(world.Entity) cube.BBox                     { return cube.BBox{} }
func (holderType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &holderData{} }
func (holderType) EncodeNBT(*world.EntityData) map[string]any      { return nil }

// EntityTypes returns the default entity types plus the game's own.
func EntityTypes() []world.EntityType {
	return append(append(entity.DefaultRegistry.Types(), HolderType, FairyType, VisualType, PlateType, GatePortalType), MonsterTypes()...)
}
