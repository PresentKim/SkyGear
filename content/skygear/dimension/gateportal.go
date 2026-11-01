package dimension

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The run's driver: an entity no one sees that follows the hunter (so its
// chunk always ticks) and drives the run every tick; it closes when the run
// ends. It is the old gate entity, whose look the pack still has.

// PortalLooks is how many looks (textures) the gate has; each hunt's gate
// takes one by the monster's place in the order.
const PortalLooks = 8

// GatePortal is the gate entity. Variant picks its texture in the render
// controller (the level minus one).
type GatePortal struct {
	tx     *world.Tx
	Handle *world.EntityHandle
	Data   *world.EntityData
}

type gatePortalData struct{ Level int }

type gatePortalConfig struct{ Level int }

func (c gatePortalConfig) Apply(data *world.EntityData) {
	data.Data = &gatePortalData{Level: c.Level}
}

func (g *GatePortal) H() *world.EntityHandle { return g.Handle }

func (g *GatePortal) Position() mgl64.Vec3 { return g.Data.Pos }

func (g *GatePortal) Rotation() cube.Rotation { return g.Data.Rot }

func (g *GatePortal) Variant() int32 { return int32(g.Data.Data.(*gatePortalData).Level - 1) }

// Invisible keeps the driver out of sight.
func (g *GatePortal) Invisible() bool { return true }

// Tick drives the run whose gate it is, and closes a gate that no run
// holds (left from an earlier session).
func (g *GatePortal) Tick(tx *world.Tx, _ int64) {
	s, ok := core.Of(tx)
	if !ok {
		_ = g.Close()
		return
	}
	if r := RunOf(s); r != nil && r.Portal == g.Handle {
		r.tick(s, tx)
		return
	}
	_ = g.Close()
}

// Close takes the gate out of the world before closing its handle (a handle
// still in a world cannot close, as when its chunk unloads).
func (g *GatePortal) Close() error {
	g.tx.RemoveEntity(g)
	return g.Handle.Close()
}

// GatePortalType is the world.EntityType of GatePortal.
var GatePortalType gatePortalType

type gatePortalType struct{}

func (gatePortalType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &gatePortalData{Level: 1}
	}
	return &GatePortal{tx: tx, Handle: handle, Data: data}
}

func (gatePortalType) EncodeEntity() string { return "skygear:gate_portal" }

func (gatePortalType) BBox(world.Entity) cube.BBox { return cube.BBox{} }

func (gatePortalType) DecodeNBT(_ map[string]any, d *world.EntityData) {
	d.Data = &gatePortalData{Level: 1}
}

func (gatePortalType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// dirYaw is the yaw of a direction (0 south, 90 west).
func dirYaw(dir mgl64.Vec3) float64 { return mgl64.RadToDeg(math.Atan2(-dir[0], dir[2])) }
