package dimension

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// The hunting ground: a round grassland of core.ArenaRadius around
// core.ArenaCentre with a path around its rim and barrier blocks past it.
// It is laid once per world and looks the same at every entry, since no one
// can build in it.

const (
	// gateOffset is how far from the centre the gate stands, the players'
	// start the same way on the other side.
	gateOffset = 12
	// wallHeight is how high the barrier around the ground goes.
	wallHeight = 4
)

// arenaGate and arenaStart are where the gate and the hunter's start are.
func arenaGate() mgl64.Vec3 {
	return core.ArenaCentre.Vec3Middle().Add(mgl64.Vec3{-gateOffset, 0, 0})
}

func arenaStart() mgl64.Vec3 {
	return core.ArenaCentre.Vec3Middle().Add(mgl64.Vec3{gateOffset, 0, 0})
}

// buildArena lays the hunting ground unless it is already there.
func buildArena(tx *world.Tx) {
	c := core.ArenaCentre
	floor := c.Y() - 1
	if _, ok := tx.Block(cube.Pos{c.X(), floor, c.Z()}).(block.Grass); ok {
		return
	}
	r := core.ArenaRadius
	for dx := -r - 1; dx <= r+1; dx++ {
		for dz := -r - 1; dz <= r+1; dz++ {
			d := math.Hypot(float64(dx), float64(dz))
			x, z := c.X()+dx, c.Z()+dz
			switch {
			case d < float64(r-1):
				tx.SetBlock(cube.Pos{x, floor - 1, z}, block.Dirt{}, nil)
				tx.SetBlock(cube.Pos{x, floor, z}, block.Grass{}, nil)
				if deco := decoration(x, z); deco != nil {
					tx.SetBlock(cube.Pos{x, c.Y(), z}, deco, nil)
				}
			case d < float64(r):
				tx.SetBlock(cube.Pos{x, floor - 1, z}, block.Dirt{}, nil)
				tx.SetBlock(cube.Pos{x, floor, z}, block.DirtPath{}, nil)
			case d < float64(r+1):
				tx.SetBlock(cube.Pos{x, floor, z}, block.Dirt{}, nil)
				for y := range wallHeight {
					tx.SetBlock(cube.Pos{x, c.Y() + y, z}, block.Barrier{}, nil)
				}
			}
		}
	}
}

// decoration is the plant on the grass at (x, z): short grass and a few
// flowers, the same on every build, none near the gate or the start.
func decoration(x, z int) world.Block {
	for _, at := range []mgl64.Vec3{arenaGate(), arenaStart()} {
		if math.Hypot(float64(x)-at[0], float64(z)-at[2]) < 4 {
			return nil
		}
	}
	h := uint32(x)*73856093 ^ uint32(z)*19349663
	h ^= h >> 13
	switch h % 100 {
	case 0:
		return block.Flower{Type: block.Dandelion()}
	case 1:
		return block.Flower{Type: block.Cornflower()}
	case 2, 3, 4, 5, 6, 7:
		return block.ShortGrass{}
	}
	return nil
}

// insideArena keeps pos inside the ground, margin blocks from the rim.
func insideArena(pos mgl64.Vec3, margin float64) mgl64.Vec3 {
	c := core.ArenaCentre.Vec3Middle()
	off := mgl64.Vec3{pos[0] - c[0], 0, pos[2] - c[2]}
	limit := float64(core.ArenaRadius) - margin
	if l := off.Len(); l > limit {
		off = off.Mul(limit / l)
	}
	return mgl64.Vec3{c[0] + off[0], pos[1], c[2] + off[2]}
}
