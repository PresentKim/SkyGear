package wave

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The gate: starting a wave opens a slowly turning round portal over the
// anchor, textured by the level and facing the player who opened it. The
// monsters come out of it every gatePortalEvery and walk out the way it faces
// before going for the players. It closes when the wave ends.

// GatePortalEvery is how many Fast calls (0.2 s each) pass between two
// monsters coming out of the gate.
const GatePortalEvery = 1

// GateExitTicks is how long a monster walks straight out of the gate.
const GateExitTicks = 25

// GatePortal is the gate entity over the anchor. Variant is the wave level
// minus one: the render controller picks the texture with it.
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

// Tick closes a gate that no wave holds (left from an earlier session).
func (g *GatePortal) Tick(tx *world.Tx, _ int64) {
	s, ok := core.Of(tx)
	if !ok || LocalOf(s).Wave == nil || LocalOf(s).Wave.Portal != g.Handle {
		_ = g.Close()
	}
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

// gateFacing is the way out of the gate at anchor towards a player at from:
// the nearest of the four directions.
func gateFacing(anchor cube.Pos, from mgl64.Vec3) mgl64.Vec3 {
	d := from.Sub(anchor.Vec3Centre())
	if math.Abs(d[0]) > math.Abs(d[2]) {
		return mgl64.Vec3{math.Copysign(1, d[0]), 0, 0}
	}
	if d[2] == 0 {
		return mgl64.Vec3{0, 0, 1}
	}
	return mgl64.Vec3{0, 0, math.Copysign(1, d[2])}
}

// dirYaw is the yaw of a direction (0 south, 90 west).
func dirYaw(dir mgl64.Vec3) float64 { return mgl64.RadToDeg(math.Atan2(-dir[0], dir[2])) }

// openGatePortal opens the gate of a wave over its anchor, facing out.
func openGatePortal(s *core.Session, tx *world.Tx, w *State) {
	h := world.EntitySpawnOpts{Position: w.Gate.Vec3Middle().Add(mgl64.Vec3{0, 1, 0}), Rotation: cube.Rotation{dirYaw(w.Out), 0}}.New(GatePortalType, gatePortalConfig{Level: w.Level})
	tx.AddEntity(h)
	w.Portal = h
}

func closeGatePortal(s *core.Session, tx *world.Tx, w *State) {
	if w.Portal == nil {
		return
	}
	if e, ok := w.Portal.Entity(tx); ok {
		tx.RemoveEntity(e)
	}
	_ = w.Portal.Close()
	w.Portal = nil
}

// returnToGate takes back a monster of the wave that fell into the void: it
// comes out of the gate again, unhurt by the fall and with the health it had.
func returnToGate(s *core.Session, tx *world.Tx, m *Monster) {
	w := LocalOf(s).Wave
	d := m.Md()
	d.vel, d.onGround = mgl64.Vec3{}, false
	d.exit, d.ExitTicks = w.Out, GateExitTicks
	side := mgl64.Vec3{w.Out[2], 0, -w.Out[0]}.Mul(s.Rand.Float64()*1.2 - 0.6)
	m.Data.Pos = w.Gate.Vec3Middle().Add(mgl64.Vec3{0, 1, 0}).Add(w.Out.Mul(0.6)).Add(side)
	m.Data.Rot = cube.Rotation{dirYaw(w.Out), 0}
	for _, v := range tx.Viewers(m.Data.Pos) {
		v.ViewEntityTeleport(m, m.Data.Pos)
	}
}

// EmergeMonsters lets the next monster of the wave out of the gate, every
// gatePortalEvery calls (Fast).
func EmergeMonsters(s *core.Session, tx *world.Tx) {
	w := LocalOf(s).Wave
	if w == nil || len(w.Pending) == 0 {
		return
	}
	if w.emerge++; w.emerge < GatePortalEvery {
		return
	}
	w.emerge = 0
	mob := w.Pending[0]
	w.Pending = w.Pending[1:]
	side := mgl64.Vec3{w.Out[2], 0, -w.Out[0]}.Mul(s.Rand.Float64()*1.2 - 0.6)
	at := w.Gate.Vec3Middle().Add(mgl64.Vec3{0, 1, 0}).Add(w.Out.Mul(0.6)).Add(side)
	h := world.EntitySpawnOpts{Position: at, Rotation: cube.Rotation{dirYaw(w.Out), 0}}.New(MobTypes[mob], MonsterConfig{Mob: mob, exit: w.Out})
	w.Left[h] = mob
	tx.AddEntity(h)
}
