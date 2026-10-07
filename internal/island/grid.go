// Package island maps island indices to world positions.
package island

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

// Grid lays islands out on a square grid away from the lobby at the origin.
type Grid struct {
	// Spacing is the distance between island centres.
	Spacing int
	// Radius is the protected radius around an island centre.
	Radius int
	// Height is the Y level of the OneBlock.
	Height int
}

const columns = 1000

// Centre returns the OneBlock position of the island with index i.
func (g Grid) Centre(i int) cube.Pos {
	col, row := i%columns, i/columns
	return cube.Pos{(col + 1) * g.Spacing, g.Height, (row + 1) * g.Spacing}
}

// IndexAt returns the island index whose protected area contains pos.
func (g Grid) IndexAt(pos cube.Pos) (int, bool) {
	col := floorDiv(pos.X()+g.Spacing/2, g.Spacing) - 1
	row := floorDiv(pos.Z()+g.Spacing/2, g.Spacing) - 1
	if col < 0 || row < 0 || col >= columns {
		return 0, false
	}
	i := row*columns + col
	c := g.Centre(i)
	if abs(pos.X()-c.X()) > g.Radius || abs(pos.Z()-c.Z()) > g.Radius {
		return 0, false
	}
	return i, true
}

// Spawn returns the standing position on top of the OneBlock of island i.
func (g Grid) Spawn(i int) mgl64.Vec3 {
	return g.Centre(i).Vec3Middle().Add(mgl64.Vec3{0, 1, 0})
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
