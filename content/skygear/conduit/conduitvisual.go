package conduit

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// ConduitVisual draws a conduit bundle so the bundle block needs a single
// state: the render controller decodes the look from two variants. Each kind
// has a code, 0 for none, else 1 + (tier-1)*64 + sides (bit i is
// conduitFaces[i]); Variant = energy + 193*item, MarkVariant = fluid, small
// enough for the client's float32 molang. It removes itself when its bundle
// or session is gone (entities are saved with the world).

const Codes = 193

// Code is the code of a conduit of a tier connecting to sides.
func Code(tier, sides int) int32 {
	if tier <= 0 {
		return 0
	}
	return int32(1 + (min(tier, Tiers)-1)*64 + sides&63)
}

// Look returns the Variant and MarkVariant of a bundle.
func Look(b Bundle) (variant, mark int32) {
	return Code(b.Energy, b.EConn) + Codes*Code(b.Item, b.IConn), Code(b.Fluid, b.FConn)
}

type Visual struct {
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

func (v *Visual) H() *world.EntityHandle { return v.handle }

func (v *Visual) Position() mgl64.Vec3 { return v.data.Pos }

func (v *Visual) Rotation() cube.Rotation { return v.data.Rot }

// Variant and MarkVariant are sent to the client as the entity's variants.
func (v *Visual) Variant() int32 { return v.data.Data.(*visualData).variant }

func (v *Visual) MarkVariant() int32 { return v.data.Data.(*visualData).mark }

// Tick removes visuals that no longer belong to a bundle.
func (v *Visual) Tick(tx *world.Tx, _ int64) {
	d := v.data.Data.(*visualData)
	s, ok := core.Of(tx)
	_, isBundle := tx.Block(d.owner).(Bundle)
	if ok && isBundle && ownsVisual(s, d.owner, v.handle) {
		return
	}
	_ = v.Close()
}

func (v *Visual) Close() error {
	v.tx.RemoveEntity(v)
	return v.handle.Close()
}

// VisualType is the world.EntityType of ConduitVisual. Its identifier matches
// the client entity in assets/resourcepack/skygear.
var VisualType visualType

type visualType struct{}

func (visualType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &visualData{}
	}
	return &Visual{tx: tx, handle: handle, data: data}
}

func (visualType) EncodeEntity() string { return "skygear:conduit_visual" }

func (visualType) BBox(world.Entity) cube.BBox { return cube.BBox{} }

func (visualType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &visualData{} }

func (visualType) EncodeNBT(*world.EntityData) map[string]any { return nil }

func ownsVisual(s *core.Session, pos cube.Pos, h *world.EntityHandle) bool {
	return LocalOf(s).Visuals[pos] == h
}

// showConduit makes sure the bundle at pos has a display entity that shows
// what it holds now: one that is out of date is made again.
func showConduit(s *core.Session, tx *world.Tx, pos cube.Pos, b Bundle) {
	showConduitWith(s, tx, pos, b, plugsOf(tx, pos, b))
}

// showConduitWith is showConduit for a bundle whose plugs are known.
func showConduitWith(s *core.Session, tx *world.Tx, pos cube.Pos, b Bundle, plugs map[plugKey]int) {
	syncPlates(s, tx, pos, b, plugs)
	variant, mark := Look(b)
	if h, ok := LocalOf(s).Visuals[pos]; ok {
		if e, ok := h.Entity(tx); ok {
			if d := e.(*Visual).data.Data.(*visualData); d.variant == variant && d.mark == mark {
				return
			}
			delete(LocalOf(s).Visuals, pos)
			_ = e.Close()
		} else {
			delete(LocalOf(s).Visuals, pos)
		}
	}
	h := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(pos[0]) + 0.5, float64(pos[1]), float64(pos[2]) + 0.5}}.
		New(VisualType, visualConfig{owner: pos, variant: variant, mark: mark})
	LocalOf(s).Visuals[pos] = h
	tx.AddEntity(h)
}

// RemoveVisual removes the display entity and the plates of the bundle at pos.
func RemoveVisual(s *core.Session, tx *world.Tx, pos cube.Pos) {
	removePlates(s, tx, pos)
	h, ok := LocalOf(s).Visuals[pos]
	if !ok {
		return
	}
	delete(LocalOf(s).Visuals, pos)
	if e, ok := h.Entity(tx); ok {
		_ = e.Close()
	}
}

// RemoveVisuals removes every display entity before the world is saved.
func RemoveVisuals(s *core.Session, tx *world.Tx) {
	for pos := range LocalOf(s).Visuals {
		RemoveVisual(s, tx, pos)
	}
	for k := range LocalOf(s).Plates {
		removePlates(s, tx, k.Pos)
	}
}
