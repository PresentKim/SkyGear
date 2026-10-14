package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Conduits (docs/design/06-power-and-conduits.md): cables that carry energy
// (energy.go in internal/energy) and, later, items. A conduit connects by
// itself to the conduits of its kind and the blocks that take part next to
// it, and shows an arm towards each connected side: its block state is the
// set of connected sides (conduitConnState, 0 to 63, bit i is
// conduitFaces[i]), and the pack has a geometry for every set
// (models/blocks/conduit.geo.json, made by tools/conduitgeo).

// conduitFaces are the six sides in the bit order of a connection set.
var conduitFaces = [6]cube.Face{cube.FaceDown, cube.FaceUp, cube.FaceNorth, cube.FaceSouth, cube.FaceWest, cube.FaceEast}

// conduitConnState is the one block state of a conduit: its connected sides
// as a two digit number.
const conduitConnState = "dragonfactory:conn"

// conduitTiers is how many tiers conduits have.
const conduitTiers = 3

func conduitConnValue(conn int) string { return fmt.Sprintf("%02d", conn) }

func conduitStates() map[string][]any {
	var values []any
	for conn := range 64 {
		values = append(values, conduitConnValue(conn))
	}
	return map[string][]any{conduitConnState: values}
}

// conduitPermutations is one geometry per non-empty set of sides.
func conduitPermutations(textures map[string]customblock.Material) []customblock.Permutation {
	var out []customblock.Permutation
	for conn := 1; conn < 64; conn++ {
		out = append(out, customblock.Permutation{
			Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", conduitConnState, conduitConnValue(conn)),
			Properties: customblock.Properties{Geometry: conduitGeometry(conn), Textures: textures, SelectionBox: conduitCore},
		})
	}
	return out
}

func conduitGeometry(conn int) string {
	return "geometry.dragonfactory_conduit_" + conduitConnValue(conn)
}

// conduitCore is the box of the core in block units: what the client lets
// the player point at (a little bigger than drawn, so a thin cable can be
// targeted).
var conduitCore = cube.Box(5.0/16, 5.0/16, 5.0/16, 11.0/16, 11.0/16, 11.0/16)

// conduitModel is the shape of a conduit: the core and an arm to each
// connected side; no side of it is solid.
type conduitModel struct{ conn int }

var conduitArmBoxes = [6]cube.BBox{
	cube.Box(6.0/16, 0, 6.0/16, 10.0/16, 6.0/16, 10.0/16),
	cube.Box(6.0/16, 10.0/16, 6.0/16, 10.0/16, 1, 10.0/16),
	cube.Box(6.0/16, 6.0/16, 0, 10.0/16, 10.0/16, 6.0/16),
	cube.Box(6.0/16, 6.0/16, 10.0/16, 10.0/16, 10.0/16, 1),
	cube.Box(0, 6.0/16, 6.0/16, 6.0/16, 10.0/16, 10.0/16),
	cube.Box(10.0/16, 6.0/16, 6.0/16, 1, 10.0/16, 10.0/16),
}

func (m conduitModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	boxes := []cube.BBox{cube.Box(6.0/16, 6.0/16, 6.0/16, 10.0/16, 10.0/16, 10.0/16)}
	for i, b := range conduitArmBoxes {
		if m.conn&(1<<i) != 0 {
			boxes = append(boxes, b)
		}
	}
	return boxes
}

func (conduitModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return false }

// connectionsOf returns the set of sides of pos whose neighbour takes part
// according to joins (given the face of pos the neighbour is on).
func connectionsOf(tx *world.Tx, pos cube.Pos, joins func(f cube.Face, at cube.Pos, b world.Block) bool) int {
	conn := 0
	for i, f := range conduitFaces {
		at := pos.Side(f)
		if joins(f, at, tx.Block(at)) {
			conn |= 1 << i
		}
	}
	return conn
}

// EnergyConduit is an energy cable of a tier (1 to 3); Conn is the set of
// sides it shows an arm to.
type EnergyConduit struct {
	Tier, Conn int
}

var energyConduitHashes = func() [conduitTiers + 1]uint64 {
	var h [conduitTiers + 1]uint64
	for t := 1; t <= conduitTiers; t++ {
		h[t] = block.NextHash()
	}
	return h
}()

func (c EnergyConduit) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:energy_conduit_%d", c.Tier), 0
}
func (c EnergyConduit) EncodeBlock() (string, map[string]any) {
	return fmt.Sprintf("dragonfactory:energy_conduit_%d", c.Tier), map[string]any{conduitConnState: conduitConnValue(c.Conn)}
}
func (c EnergyConduit) Hash() (uint64, uint64)  { return energyConduitHashes[c.Tier], uint64(c.Conn) }
func (c EnergyConduit) Model() world.BlockModel { return conduitModel{c.conn()} }
func (c EnergyConduit) conn() int               { return c.Conn }
func (c EnergyConduit) Name() string {
	return i18n.T(i18n.Default, fmt.Sprintf("name.EnergyConduit.%d", c.Tier))
}
func (EnergyConduit) Category() category.Category { return category.Construction() }

func (c EnergyConduit) materials() map[string]customblock.Material {
	return map[string]customblock.Material{"*": alphaMaterial(fmt.Sprintf("dragonfactory_conduit_energy_%d", c.Tier))}
}

func (c EnergyConduit) Properties() customblock.Properties {
	return customblock.Properties{Geometry: conduitGeometry(0), Textures: c.materials(), SelectionBox: conduitCore}
}

// States and Permutations implement block.Permutable.
func (EnergyConduit) States() map[string][]any { return conduitStates() }
func (c EnergyConduit) Permutations() []customblock.Permutation {
	return conduitPermutations(c.materials())
}

func (c EnergyConduit) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 3,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			return []item.Stack{item.NewStack(EnergyConduit{Tier: c.Tier}, 1)}
		},
	}
}

// EncodeNBT and DecodeNBT make the conduit a block entity: only those get
// Tick calls from the world. It stores nothing.
func (EnergyConduit) EncodeNBT() map[string]any      { return map[string]any{"id": "DragonFactoryConduit"} }
func (c EnergyConduit) DecodeNBT(map[string]any) any { return c }

// EnergyTier implements energy.Conduit.
func (c EnergyConduit) EnergyTier() int { return c.Tier }

// joinsEnergy reports whether a conduit connects to b: another energy conduit
// or a block that gives or takes energy.
func joinsEnergy(b world.Block) bool {
	if _, ok := b.(energy.Conduit); ok {
		return true
	}
	src, snk := energy.RolesOf(b)
	return src != nil || snk != nil
}

// refresh sets the block to show the sides it connects to.
func (c EnergyConduit) refresh(pos cube.Pos, tx *world.Tx) {
	if conn := connectionsOf(tx, pos, func(_ cube.Face, _ cube.Pos, b world.Block) bool { return joinsEnergy(b) }); conn != c.Conn {
		c.Conn = conn
		tx.SetBlock(pos, c, nil)
	}
}

// NeighbourUpdateTick keeps the arms up to date.
func (c EnergyConduit) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) { c.refresh(pos, tx) }

// Tick registers the conduit with the grid every few ticks (and, for one
// just placed, shows its arms).
func (c EnergyConduit) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	if currentTick%4 != 0 {
		return
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	c.refresh(pos, tx)
	s.energyGrid().AddConduit(pos)
}

// energyGrid is the energy grid of the island, made when first used.
func (s *session) energyGrid() *energy.Grid {
	if s.grid == nil {
		s.grid = energy.NewGrid()
	}
	return s.grid
}

func init() {
	for t := 1; t <= conduitTiers; t++ {
		for conn := range 64 {
			world.RegisterBlock(EnergyConduit{Tier: t, Conn: conn})
		}
		world.RegisterItem(EnergyConduit{Tier: t})
	}
}
