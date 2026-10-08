package instance

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// solid reports whether a player would collide with the block at pos.
func solid(tx *world.Tx, pos cube.Pos) bool {
	return len(tx.Block(pos).Model().BBox(pos, tx)) > 0
}

// fits reports whether a player standing with its feet in pos is free.
func fits(tx *world.Tx, pos cube.Pos) bool {
	return !solid(tx, pos) && !solid(tx, pos.Side(cube.FaceUp))
}

// SafeSpot returns pos if a player fits there, otherwise the nearest spot
// on top of the obstruction (up to 4 blocks higher), then beside it on
// solid ground (up to 2 blocks away), so a player entering a save never
// ends up inside blocks placed where it stood.
func SafeSpot(tx *world.Tx, pos mgl64.Vec3) mgl64.Vec3 {
	feet := cube.PosFromVec3(pos)
	if fits(tx, feet) {
		return pos
	}
	for dy := 1; dy <= 4; dy++ {
		if p := feet.Add(cube.Pos{0, dy, 0}); fits(tx, p) {
			return mgl64.Vec3{pos.X(), float64(p.Y()), pos.Z()}
		}
	}
	for r := 1; r <= 2; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if max(abs(dx), abs(dz)) != r {
					continue
				}
				for dy := 0; dy <= 1; dy++ {
					p := feet.Add(cube.Pos{dx, dy, dz})
					if fits(tx, p) && solid(tx, p.Side(cube.FaceDown)) {
						return p.Vec3Middle()
					}
				}
			}
		}
	}
	// Boxed in: go up until there is room.
	for p := feet.Add(cube.Pos{0, 5, 0}); p.Y() < tx.Range()[1]; p = p.Side(cube.FaceUp) {
		if fits(tx, p) {
			return mgl64.Vec3{pos.X(), float64(p.Y()), pos.Z()}
		}
	}
	return pos
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
