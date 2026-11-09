// Package elevator is the wool elevator: a block of wool and an ender pearl
// that carries a player straight up or down to the next elevator of its
// colour, as the elevators of OpenBlocks and the Elevator mods do. Jumping
// on one goes up, sneaking on one goes down; the floors between do not
// matter.
package elevator

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Range is how many blocks up or down an elevator looks for the next one.
const Range = 32

// Cooldown is the time after a ride before the player rides again, so one
// jump or sneak does not count twice.
const Cooldown = 300 * time.Millisecond

var hashes = func() map[uint8]uint64 {
	m := map[uint8]uint64{}
	for _, c := range item.Colours() {
		m[c.Uint8()] = block.NextHash()
	}
	return m
}()

func init() {
	for _, c := range item.Colours() {
		world.RegisterBlock(Elevator{Colour: c})
		world.RegisterItem(Elevator{Colour: c})
	}
}

// Elevator is the elevator block of a wool Colour. It leads only to
// elevators of the same colour, so one column can hold several lines.
type Elevator struct{ Colour item.Colour }

func (e Elevator) EncodeItem() (string, int16) { return "skygear:elevator_" + e.Colour.String(), 0 }

func (e Elevator) EncodeBlock() (string, map[string]any) {
	return "skygear:elevator_" + e.Colour.String(), nil
}

func (e Elevator) Hash() (uint64, uint64) { return hashes[e.Colour.Uint8()], 0 }

func (Elevator) Model() world.BlockModel { return model.Solid{} }

func (e Elevator) Name() string { return i18n.T(i18n.Default, "name.Elevator."+e.Colour.String()) }

func (Elevator) Category() category.Category { return category.Construction() }

func (e Elevator) Properties() customblock.Properties {
	name := "skygear_elevator_" + e.Colour.String()
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":  blockmodel.SolidMaterial(name + "_side"),
		"up": blockmodel.SolidMaterial(name + "_up"),
	}}
}

// BreakInfo is that of wool: soft, mined faster with shears.
func (e Elevator) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.8, BlastResistance: 0.8,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.EffectiveFor(item.TypeShears),
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(e, 1)} },
	}
}

// lastRide is when each player last rode an elevator.
var lastRide = core.NewPlayers[time.Time]()

// Up carries p to the next elevator above the one it stands on. It reports
// whether p rode.
func Up(s *core.Session, p *player.Player, tx *world.Tx) bool { return ride(s, p, tx, 1) }

// Down carries p to the next elevator below the one it stands on. It
// reports whether p rode.
func Down(s *core.Session, p *player.Player, tx *world.Tx) bool { return ride(s, p, tx, -1) }

func ride(s *core.Session, p *player.Player, tx *world.Tx, dir int) bool {
	from, e, ok := Under(tx, p.Position())
	if !ok {
		return false
	}
	to, ok := Next(tx, from, e.Colour, dir)
	if !ok {
		return false
	}
	rides, h := lastRide.Of(s), p.H()
	if now := time.Now(); now.Sub(rides[h]) < Cooldown {
		return false
	} else {
		rides[h] = now
	}
	pos := p.Position()
	dst := mgl64.Vec3{pos.X(), float64(to.Y() + 1), pos.Z()}
	tx.PlaySound(pos, sound.Teleport{})
	p.Teleport(dst)
	tx.PlaySound(dst, sound.Teleport{})
	return true
}

// Under is the elevator a player with feet at pos stands on.
func Under(tx *world.Tx, pos mgl64.Vec3) (cube.Pos, Elevator, bool) {
	at := cube.PosFromVec3(pos.Sub(mgl64.Vec3{0, 0.2, 0}))
	e, ok := tx.Block(at).(Elevator)
	return at, e, ok
}

// Next is the nearest elevator of colour c above (dir 1) or below (dir -1)
// from, within Range, with room for a player on top.
func Next(tx *world.Tx, from cube.Pos, c item.Colour, dir int) (cube.Pos, bool) {
	r := tx.Range()
	for d := 1; d <= Range; d++ {
		pos := from.Add(cube.Pos{0, dir * d, 0})
		if pos.OutOfBounds(r) {
			break
		}
		if e, ok := tx.Block(pos).(Elevator); ok && e.Colour == c && roomAbove(tx, pos) {
			return pos, true
		}
	}
	return cube.Pos{}, false
}

// roomAbove reports whether the two blocks above pos let a player stand.
func roomAbove(tx *world.Tx, pos cube.Pos) bool {
	for i := 1; i <= 2; i++ {
		above := pos.Add(cube.Pos{0, i, 0})
		if len(tx.Block(above).Model().BBox(above, tx)) > 0 {
			return false
		}
	}
	return true
}

// ResetCooldown lets p ride again at once.
func ResetCooldown(s *core.Session, p *player.Player) { delete(lastRide.Of(s), p.H()) }
