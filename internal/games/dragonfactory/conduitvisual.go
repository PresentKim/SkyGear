package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// ConduitVisual draws a conduit bundle so the bundle block needs a single
// state: the render controller decodes the look from two variants. Each kind
// has a code, 0 for none, else 1 + (tier-1)*64 + sides (bit i is
// conduitFaces[i]); Variant = energy + 193*item, MarkVariant = fluid, small
// enough for the client's float32 molang. It removes itself when its bundle
// or session is gone (entities are saved with the world).

const conduitCodes = 193

// conduitCode is the code of a conduit of a tier connecting to sides.
func conduitCode(tier, sides int) int32 {
	if tier <= 0 {
		return 0
	}
	return int32(1 + (min(tier, conduitTiers)-1)*64 + sides&63)
}

// conduitLook returns the Variant and MarkVariant of a bundle.
func conduitLook(b ConduitBundle) (variant, mark int32) {
	return conduitCode(b.Energy, b.EConn) + conduitCodes*conduitCode(b.Item, b.IConn), conduitCode(b.Fluid, b.FConn)
}

type ConduitVisual struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
}

type visualData struct {
	owner         cube.Pos
	variant, mark int32
}

// visualConfig spawns a ConduitVisual for the bundle at owner.
type visualConfig struct {
	owner         cube.Pos
	variant, mark int32
}

func (c visualConfig) Apply(data *world.EntityData) {
	data.Data = &visualData{owner: c.owner, variant: c.variant, mark: c.mark}
}

func (v *ConduitVisual) H() *world.EntityHandle  { return v.handle }
func (v *ConduitVisual) Position() mgl64.Vec3    { return v.data.Pos }
func (v *ConduitVisual) Rotation() cube.Rotation { return v.data.Rot }

// Variant and MarkVariant are sent to the client as the entity's variants.
func (v *ConduitVisual) Variant() int32     { return v.data.Data.(*visualData).variant }
func (v *ConduitVisual) MarkVariant() int32 { return v.data.Data.(*visualData).mark }

// Tick removes visuals that no longer belong to a bundle.
func (v *ConduitVisual) Tick(tx *world.Tx, _ int64) {
	d := v.data.Data.(*visualData)
	s, ok := activeGame.sessionOf(tx.World())
	_, isBundle := tx.Block(d.owner).(ConduitBundle)
	if ok && isBundle && s.ownsVisual(d.owner, v.handle) {
		return
	}
	_ = v.Close()
}

func (v *ConduitVisual) Close() error {
	v.tx.RemoveEntity(v)
	return v.handle.Close()
}

// VisualType is the world.EntityType of ConduitVisual. Its identifier matches
// the client entity in assets/resourcepack/dragonfactory.
var VisualType visualType

type visualType struct{}

func (visualType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &visualData{}
	}
	return &ConduitVisual{tx: tx, handle: handle, data: data}
}
func (visualType) EncodeEntity() string                            { return "dragonfactory:conduit_visual" }
func (visualType) BBox(world.Entity) cube.BBox                     { return cube.BBox{} }
func (visualType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &visualData{} }
func (visualType) EncodeNBT(*world.EntityData) map[string]any      { return nil }

func (s *session) ownsVisual(pos cube.Pos, h *world.EntityHandle) bool {
	return s.visuals[pos] == h
}

// showConduit makes sure the bundle at pos has a display entity that shows
// what it holds now: one that is out of date is made again.
func (s *session) showConduit(tx *world.Tx, pos cube.Pos, b ConduitBundle) {
	s.showConduitWith(tx, pos, b, plugsOf(tx, pos, b))
}

// showConduitWith is showConduit for a bundle whose plugs are known.
func (s *session) showConduitWith(tx *world.Tx, pos cube.Pos, b ConduitBundle, plugs map[plugKey]int) {
	if s.visuals == nil {
		s.visuals = map[cube.Pos]*world.EntityHandle{}
	}
	s.syncPlates(tx, pos, b, plugs)
	variant, mark := conduitLook(b)
	if h, ok := s.visuals[pos]; ok {
		if e, ok := h.Entity(tx); ok {
			if d := e.(*ConduitVisual).data.Data.(*visualData); d.variant == variant && d.mark == mark {
				return
			}
			delete(s.visuals, pos)
			_ = e.Close()
		} else {
			delete(s.visuals, pos)
		}
	}
	h := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(pos[0]) + 0.5, float64(pos[1]), float64(pos[2]) + 0.5}}.
		New(VisualType, visualConfig{owner: pos, variant: variant, mark: mark})
	s.visuals[pos] = h
	tx.AddEntity(h)
}

// removeVisual removes the display entity and the plates of the bundle at pos.
func (s *session) removeVisual(tx *world.Tx, pos cube.Pos) {
	s.removePlates(tx, pos)
	h, ok := s.visuals[pos]
	if !ok {
		return
	}
	delete(s.visuals, pos)
	if e, ok := h.Entity(tx); ok {
		_ = e.Close()
	}
}

// removeVisuals removes every display entity before the world is saved.
func (s *session) removeVisuals(tx *world.Tx) {
	for pos := range s.visuals {
		s.removeVisual(tx, pos)
	}
	for k := range s.plates {
		s.removePlates(tx, k.pos)
	}
}
