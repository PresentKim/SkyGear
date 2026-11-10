package conduit

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/content/skygear/core"
)

// ConduitPlate is the EnderIO-style connector plate on a side of a bundle that
// meets a block other than a conduit of its kind. It has no hitbox; clicking
// its square on the block it touches opens the side's settings. The render
// controllers decode
//
//	Variant     = side + 6*kinds + 48*mode + 192*extractChannel + 3072*insertChannel
//	MarkVariant = energyTier + 4*itemTier + 16*fluidTier
//
// with kinds the bit set of kinds with a connector (energy 1, item 2, fluid 4).

// conduitKinds' indices: energy 0, item 1, fluid 2 (see bundle.go).

var _ block.Activatable = Bundle{}

// PlateKey identifies the plate on a side of the bundle at pos.
type PlateKey struct {
	Pos  cube.Pos
	Side int // index in conduitFaces
}

// FaceIndex is the index of f in conduitFaces.
func FaceIndex(f cube.Face) int {
	for i, g := range Faces {
		if g == f {
			return i
		}
	}
	return 0
}

// plugsOf returns the connectors a bundle should have: for each kind of conduit
// in it and side where the kind meets a block that is not a conduit of its
// kind, the tier of the conduit. An item conduit also has its connector where
// the side is switched off (the plate is how it is switched on again).
func plugsOf(tx *world.Tx, pos cube.Pos, b Bundle) plugs {
	return b.plugsFrom(b.sides(tx, pos))
}

// plugs are the tiers of the connectors by side and kind (0 energy, 1 item,
// 2 fluid); 0 is no connector.
type plugs [6][3]int

// plugsFrom is plugsOf for the neighbour sides of a bundle.
func (b Bundle) plugsFrom(sides [6]bundleSide) (out plugs) {
	for side, s := range sides {
		other := s.block
		if b.Energy > 0 {
			if _, isConduit := energy.IsConduit(other); !isConduit && JoinsEnergy(other) {
				out[side][0] = b.Energy
			}
		}
		if b.Item > 0 && s.holdsItems {
			out[side][1] = b.Item
		}
		if b.Fluid > 0 {
			if o, isBundle := other.(Bundle); !isBundle || o.Fluid == 0 {
				if joinsFluid(other) {
					out[side][2] = b.Fluid
				}
			}
		}
	}
	return out
}

// PlateKinds are the kinds that have a connector on the side of the bundle.
func PlateKinds(tx *world.Tx, pos cube.Pos, b Bundle, side int) []int {
	var out []int
	plugs := plugsOf(tx, pos, b)
	for k := range 3 {
		if plugs[side][k] != 0 {
			out = append(out, k)
		}
	}
	return out
}

// plateLook returns the two numbers of the plate on a side of a bundle, and
// whether it needs a plate at all.
func plateLook(plugs plugs, b Bundle, side int) (variant, mark int32, any bool) {
	kinds := 0
	for k := range 3 {
		if plugs[side][k] != 0 {
			kinds |= 1 << k
		}
	}
	if kinds == 0 {
		return 0, 0, false
	}
	f := Faces[side]
	variant = int32(side + 6*kinds + 48*int(b.Modes[f]) + 192*b.Extract[f] + 3072*b.Insert[f])
	mark = int32(b.Energy + 4*b.Item + 16*b.Fluid)
	return variant, mark, true
}

type Plate struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
}

type plateData struct {
	key           PlateKey
	variant, mark int32
}

type plateConfig struct {
	key           PlateKey
	variant, mark int32
}

func (c plateConfig) Apply(data *world.EntityData) {
	data.Data = &plateData{key: c.key, variant: c.variant, mark: c.mark}
}

func (p *Plate) H() *world.EntityHandle { return p.handle }

func (p *Plate) Position() mgl64.Vec3 { return p.data.Pos }

func (p *Plate) Rotation() cube.Rotation { return p.data.Rot }

func (p *Plate) Variant() int32 { return p.data.Data.(*plateData).variant }

func (p *Plate) MarkVariant() int32 { return p.data.Data.(*plateData).mark }

// Tick removes plates that no longer belong to a bundle.
func (p *Plate) Tick(tx *world.Tx, current int64) {
	if !checkTurn(p.handle, current) {
		return
	}
	d := p.data.Data.(*plateData)
	s, ok := core.Of(tx)
	_, isBundle := tx.Block(d.key.Pos).(Bundle)
	if ok && isBundle && LocalOf(s).Plates[d.key] == p.handle {
		return
	}
	_ = p.Close()
}

func (p *Plate) Close() error {
	p.tx.RemoveEntity(p)
	return p.handle.Close()
}

// PlateType is the world.EntityType of ConduitPlate.
var PlateType plateType

type plateType struct{}

func (plateType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &plateData{}
	}
	return &Plate{tx: tx, handle: handle, data: data}
}

func (plateType) EncodeEntity() string { return "skygear:conduit_plate" }

// Transient: never saved with the world.
func (plateType) Transient() bool { return true }

func (plateType) BBox(world.Entity) cube.BBox { return cube.BBox{} }

func (plateType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &plateData{} }

func (plateType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// syncPlates makes the plates of the bundle at pos what its plugs say.
// Unless verify is set, a plate whose look is as remembered is taken as alive.
func syncPlates(s *core.Session, tx *world.Tx, pos cube.Pos, b Bundle, plugs plugs, verify bool) {
	for side := range Faces {
		key := PlateKey{pos, side}
		variant, mark, need := plateLook(plugs, b, side)
		if h, ok := LocalOf(s).Plates[key]; ok {
			if !verify && need && LocalOf(s).looks[h] == [2]int32{variant, mark} {
				continue
			}
			delete(LocalOf(s).looks, h)
			if e, alive := h.Entity(tx); alive {
				d := e.(*Plate).data.Data.(*plateData)
				if need && d.variant == variant && d.mark == mark {
					continue
				}
				_ = e.Close()
			}
			delete(LocalOf(s).Plates, key)
		}
		if !need {
			continue
		}
		// The plate stands at the foot of the block like the display entity.
		h := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(pos[0]) + 0.5, float64(pos[1]), float64(pos[2]) + 0.5}}.
			New(PlateType, plateConfig{key: key, variant: variant, mark: mark})
		LocalOf(s).Plates[key] = h
		LocalOf(s).looks[h] = [2]int32{variant, mark}
		tx.AddEntity(h)
	}
}

// removePlates removes the plates of the bundle at pos.
func removePlates(s *core.Session, tx *world.Tx, pos cube.Pos) {
	plates := LocalOf(s).Plates
	for side := range Faces {
		k := PlateKey{pos, side}
		h, ok := plates[k]
		if !ok {
			continue
		}
		delete(plates, k)
		delete(LocalOf(s).looks, h)
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
}

// PlateHit reports whether a click at clickPos (in blocks, relative to the
// block that was clicked) on face of the block at clicked is on the plate of a
// bundle beside it, and returns that bundle and the side its plate is on.
func PlateHit(tx *world.Tx, clicked cube.Pos, face cube.Face, clickPos mgl64.Vec3) (cube.Pos, Bundle, int, bool) {
	at := clicked.Side(face)
	b, ok := tx.Block(at).(Bundle)
	if !ok {
		return at, b, 0, false
	}
	side := FaceIndex(face.Opposite())
	if len(PlateKinds(tx, at, b, side)) == 0 {
		return at, b, 0, false
	}
	// the plate is 10 pixels square round the middle of the face
	skip := map[cube.Axis]int{cube.X: 0, cube.Y: 1, cube.Z: 2}[face.Axis()]
	for a, c := range [3]float64{clickPos[0], clickPos[1], clickPos[2]} {
		if a == skip {
			continue
		}
		if d := (c - 0.5) * 16; d < -5.5 || d > 5.5 {
			return at, b, side, false
		}
	}
	return at, b, side, true
}

func SideName(f cube.Face) string {
	return map[cube.Face]string{
		cube.FaceUp: "up", cube.FaceDown: "down", cube.FaceNorth: "north", cube.FaceSouth: "south", cube.FaceWest: "west", cube.FaceEast: "east",
	}[f]
}

// fitsBundle reports whether the stack is a conduit of a kind the bundle has
// not got yet.
func fitsBundle(st item.Stack, b Bundle) bool {
	switch st.Item().(type) {
	case EnergyConduit:
		return b.Energy == 0
	case ItemConduit:
		return b.Item == 0
	case FluidConduit:
		return b.Fluid == 0
	case NetConduit:
		return b.Net == 0
	}
	return false
}

// NextPlateSide finds the side with a connector plate that comes after the one
// the player was last shown by clicking a side without a plate, and remembers
// it, so clicking a conduit again opens the next target block.
func NextPlateSide(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b Bundle) (int, bool) {
	last, ok := LocalOf(s).plateTurn[p.UUID()]
	if !ok {
		last = -1
	}
	for i := 1; i <= 6; i++ {
		side := (last + i + 6) % 6
		if len(PlateKinds(tx, pos, b, side)) > 0 {
			LocalOf(s).plateTurn[p.UUID()] = side
			return side, true
		}
	}
	return 0, false
}

// UsePlate opens the settings of the connector plate the player clicked on the
// block beside it, unless the held conduit goes into that conduit, and reports
// whether it took the click.
func UsePlate(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, held item.Stack) bool {
	at, b, side, ok := PlateHit(tx, pos, face, clickPos)
	if !ok || fitsBundle(held, b) {
		return false
	}
	// One click sends several use events; only the first opens a form.
	if !s.Debounced(p) {
		ShowPlateForm(s, p, tx, at, side)
	}
	return true
}

// Activate opens the settings of a conduit that has a connector plate: of the
// clicked side, or the next side with one. A held conduit that goes into the
// bundle and a sneaking player place blocks instead. Being Activatable also
// tells the client not to predict placing the held block on a bundle.
func (b Bundle) Activate(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := core.Of(tx)
	if !ok || !ok2 || p.Sneaking() {
		return false
	}
	held, _ := p.HeldItems()
	if fitsBundle(held, b) {
		return false
	}
	if len(PlateKinds(tx, pos, b, FaceIndex(face))) > 0 {
		if !s.Debounced(p) {
			ShowPlateForm(s, p, tx, pos, FaceIndex(face))
		}
		return true
	}
	if bundleHasPlate(tx, pos, b) {
		if !s.Debounced(p) {
			if side, found := NextPlateSide(s, p, tx, pos, b); found {
				ShowPlateForm(s, p, tx, pos, side)
			}
		}
		return true
	}
	return false
}

// bundleHasPlate reports whether any side of the bundle has a connector plate.
func bundleHasPlate(tx *world.Tx, pos cube.Pos, b Bundle) bool {
	for side := range Faces {
		if len(PlateKinds(tx, pos, b, side)) > 0 {
			return true
		}
	}
	return false
}
