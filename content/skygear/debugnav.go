package skygear

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// debugFind lists the blocks whose name contains Name around the operator,
// nearest first, searching shell by shell out to Radius (scenarios).
type debugFind struct {
	adminBase
	Sub    cmd.SubCommand    `cmd:"find"`
	Name   string            `cmd:"name"`
	Radius cmd.Optional[int] `cmd:"radius"`
}

const findDefaultRadius, findMaxRadius, findShown = 16, 48, 8

func (c debugFind) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	r, set := c.Radius.Load()
	if !set {
		r = findDefaultRadius
	}
	r = min(max(r, 1), findMaxRadius)
	centre := cube.PosFromVec3(p.Position())
	found := findBlocks(tx, centre, r, strings.ToLower(c.Name))
	if len(found) == 0 {
		o.Errorf("no %q within %d blocks", c.Name, r)
		return
	}
	for _, f := range found[:min(len(found), findShown)] {
		o.Printf("%s at %d %d %d (%.1f)", f.name, f.pos[0], f.pos[1], f.pos[2], f.dist)
	}
	if len(found) > findShown {
		o.Printf("%d more", len(found)-findShown)
	}
}

type foundBlock struct {
	pos  cube.Pos
	name string
	dist float64
}

// findBlocks are the blocks around centre whose name contains name, nearest
// first: each shell of the cube out to r is searched before the next, so the
// search stops at the first shell with a match and the one after it.
func findBlocks(tx *world.Tx, centre cube.Pos, r int, name string) []foundBlock {
	var out []foundBlock
	stopAt := r
	for d := 0; d <= stopAt; d++ {
		for x := -d; x <= d; x++ {
			for y := -d; y <= d; y++ {
				for z := -d; z <= d; z++ {
					if max(abs(x), abs(y), abs(z)) != d {
						continue
					}
					pos := centre.Add(cube.Pos{x, y, z})
					if pos.OutOfBounds(tx.Range()) {
						continue
					}
					n, _ := tx.Block(pos).EncodeBlock()
					if !strings.Contains(n, name) {
						continue
					}
					out = append(out, foundBlock{pos, n, math.Sqrt(float64(x*x + y*y + z*z))})
				}
			}
		}
		if len(out) > 0 && stopAt == r {
			stopAt = min(r, d+1)
		}
	}
	slices.SortFunc(out, func(a, b foundBlock) int { return cmp.Compare(a.dist, b.dist) })
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// debugGoto moves the operator to the nearest place to stand by the block at
// X Y Z and turns them to look at it (scenarios).
type debugGoto struct {
	adminBase
	Sub cmd.SubCommand `cmd:"goto"`
	X   int            `cmd:"x"`
	Y   int            `cmd:"y"`
	Z   int            `cmd:"z"`
}

func (c debugGoto) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	target := cube.Pos{c.X, c.Y, c.Z}
	stand, ok := standBy(tx, target)
	if !ok {
		o.Errorf("no place to stand by %d %d %d", c.X, c.Y, c.Z)
		return
	}
	feet := stand.Vec3Middle()
	eye := feet.Add(mgl64.Vec3{0, 1.62, 0})
	d := target.Vec3Centre().Sub(eye)
	yaw := mgl64.RadToDeg(math.Atan2(-d.X(), d.Z()))
	pitch := mgl64.RadToDeg(-math.Atan2(d.Y(), math.Hypot(d.X(), d.Z())))
	rot := p.Rotation()
	p.Move(mgl64.Vec3{}, yaw-rot.Yaw(), pitch-rot.Pitch())
	p.Teleport(feet)
	o.Printf("at %d %d %d looking at %d %d %d", stand[0], stand[1], stand[2], c.X, c.Y, c.Z)
}

// standBy is the nearest place by target where a player stands: two free
// blocks over a solid one, within three blocks across and two up or down,
// with a free line of blocks to target at eye height.
func standBy(tx *world.Tx, target cube.Pos) (cube.Pos, bool) {
	var best cube.Pos
	bestD := math.MaxFloat64
	for x := -3; x <= 3; x++ {
		for z := -3; z <= 3; z++ {
			if x == 0 && z == 0 {
				continue
			}
			for y := -2; y <= 1; y++ {
				pos := target.Add(cube.Pos{x, y, z})
				if !free(tx, pos) || !free(tx, pos.Side(cube.FaceUp)) || free(tx, pos.Side(cube.FaceDown)) {
					continue
				}
				d := float64(x*x+z*z) + 2*float64(y*y)
				if d < bestD {
					best, bestD = pos, d
				}
			}
		}
	}
	return best, bestD < math.MaxFloat64
}

// free reports whether a player's body fits in the block at pos.
func free(tx *world.Tx, pos cube.Pos) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	switch tx.Block(pos).(type) {
	case block.Air:
		return true
	}
	return len(tx.Block(pos).Model().BBox(pos, tx)) == 0
}
