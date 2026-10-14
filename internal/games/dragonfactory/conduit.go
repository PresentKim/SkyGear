package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Conduits (docs/design/06-power-and-conduits.md): cables that carry energy
// (internal/energy) and items (internal/itemnet). Every conduit in the world
// is a ConduitBundle (bundle.go): one block holding at most one conduit of
// each kind. EnergyConduit and ItemConduit are the items that are placed into
// a bundle. The block itself is invisible and has a single state: a display
// entity (conduitvisual.go) draws the conduits, with a core for each kind and
// an arm towards every side the kind connects to, so no combination of kinds,
// tiers and sides multiplies the block states (docs/tech/conduits.md).

// conduitFaces are the six sides in the bit order of a connection set.
var conduitFaces = [6]cube.Face{cube.FaceDown, cube.FaceUp, cube.FaceNorth, cube.FaceSouth, cube.FaceWest, cube.FaceEast}

// conduitConnState is the one block state of the conduits of earlier versions
// (oldconduits.go): their connected sides as a two digit number.
const conduitConnState = "dragonfactory:conn"

// conduitTiers is how many tiers conduits have.
const conduitTiers = 3

func conduitConnValue(conn int) string { return fmt.Sprintf("%02d", conn) }

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
