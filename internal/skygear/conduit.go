package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Conduits carry energy (common/energy), items (common/itemnet) and
// fluids (common/fluidnet). Every conduit in the world is part of a
// ConduitBundle (bundle.go); the conduit items only go into bundles.

// conduitFaces are the six sides in the bit order of a connection set.
var conduitFaces = [6]cube.Face{cube.FaceDown, cube.FaceUp, cube.FaceNorth, cube.FaceSouth, cube.FaceWest, cube.FaceEast}

// conduitTiers is how many tiers conduits have.
const conduitTiers = 3

// conduitCore is the box conduit item blocks are pointed at with.
var conduitCore = cube.Box(5.0/16, 5.0/16, 5.0/16, 11.0/16, 11.0/16, 11.0/16)

// conduitModel is the shape of a conduit block: the space its cores take
// (bundleSelection); no side of it is solid. conn is unused: the display
// entity draws the arms.
type conduitModel struct{ conn int }

func (m conduitModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{bundleSelection}
}

func (conduitModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return false }
