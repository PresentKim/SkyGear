package core

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

// The second dimension's hunting ground lies in the save's own world, far
// from the island, so the save's state stays in one world (dimension
// package). ArenaCentre is the middle of its ground: players stand at its Y.
var ArenaCentre = cube.Pos{20000, 64, 20000}

// ArenaRadius is how far from the centre the hunting ground reaches; a wall
// no one sees closes it there.
const ArenaRadius = 48

// arenaRegion is how far from the centre (on each axis) a position counts as
// in the hunting ground, the wall and a margin past it included.
const arenaRegion = ArenaRadius + 16

// InArena reports whether pos is in the hunting ground: what happens there is
// not the island's (no fairy visits, no building).
func InArena(pos mgl64.Vec3) bool {
	c := ArenaCentre.Vec3()
	return math.Abs(pos[0]-c[0]) <= arenaRegion && math.Abs(pos[2]-c[2]) <= arenaRegion
}
