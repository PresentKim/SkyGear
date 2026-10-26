package dragonfactory

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// The gate: starting a wave opens a slowly turning round portal over the
// anchor, textured by the level and facing the player who opened it. The
// monsters come out of it every gatePortalEvery and walk out the way it faces
// before going for the players. It closes when the wave ends.

// gatePortalEvery is how many Fast calls (0.2 s each) pass between two
// monsters coming out of the gate.
const gatePortalEvery = 1

// gateExitTicks is how long a monster walks straight out of the gate.
const gateExitTicks = 25

// GatePortal is the gate entity over the anchor. Variant is the wave level
// minus one: the render controller picks the texture with it.
type GatePortal struct {
	handle *world.EntityHandle
	data   *world.EntityData
}

type gatePortalData struct{ level int }

type gatePortalConfig struct{ level int }

func (c gatePortalConfig) Apply(data *world.EntityData) {
	data.Data = &gatePortalData{level: c.level}
}

func (g *GatePortal) H() *world.EntityHandle  { return g.handle }
func (g *GatePortal) Position() mgl64.Vec3    { return g.data.Pos }
func (g *GatePortal) Rotation() cube.Rotation { return g.data.Rot }
func (g *GatePortal) Variant() int32          { return int32(g.data.Data.(*gatePortalData).level - 1) }

// Tick closes a gate that no wave holds (left from an earlier session).
func (g *GatePortal) Tick(tx *world.Tx, _ int64) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok || s.wave == nil || s.wave.portal != g.handle {
		_ = g.Close()
	}
}

func (g *GatePortal) Close() error { return g.handle.Close() }

// GatePortalType is the world.EntityType of GatePortal.
var GatePortalType gatePortalType

type gatePortalType struct{}

func (gatePortalType) Open(_ *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &gatePortalData{level: 1}
	}
	return &GatePortal{handle: handle, data: data}
}
func (gatePortalType) EncodeEntity() string        { return "dragonfactory:gate_portal" }
func (gatePortalType) BBox(world.Entity) cube.BBox { return cube.BBox{} }
func (gatePortalType) DecodeNBT(_ map[string]any, d *world.EntityData) {
	d.Data = &gatePortalData{level: 1}
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
func (s *session) openGatePortal(tx *world.Tx, w *waveState) {
	h := world.EntitySpawnOpts{Position: w.gate.Vec3Middle().Add(mgl64.Vec3{0, 1, 0}), Rotation: cube.Rotation{dirYaw(w.out), 0}}.New(GatePortalType, gatePortalConfig{level: w.level})
	tx.AddEntity(h)
	w.portal = h
}

func (s *session) closeGatePortal(tx *world.Tx, w *waveState) {
	if w.portal == nil {
		return
	}
	if e, ok := w.portal.Entity(tx); ok {
		tx.RemoveEntity(e)
	}
	_ = w.portal.Close()
	w.portal = nil
}

// emergeMonsters lets the next monster of the wave out of the gate, every
// gatePortalEvery calls (Fast).
func (s *session) emergeMonsters(tx *world.Tx) {
	w := s.wave
	if w == nil || len(w.pending) == 0 {
		return
	}
	if w.emerge++; w.emerge < gatePortalEvery {
		return
	}
	w.emerge = 0
	mob := w.pending[0]
	w.pending = w.pending[1:]
	side := mgl64.Vec3{w.out[2], 0, -w.out[0]}.Mul(s.rand.Float64()*1.2 - 0.6)
	at := w.gate.Vec3Middle().Add(mgl64.Vec3{0, 1, 0}).Add(w.out.Mul(0.6)).Add(side)
	h := world.EntitySpawnOpts{Position: at, Rotation: cube.Rotation{dirYaw(w.out), 0}}.New(monsterTypes[mob], monsterConfig{mob: mob, exit: w.out})
	w.left[h] = mob
	tx.AddEntity(h)
}
