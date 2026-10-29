package skygear

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Hooks of the counter achievements for what a player does with a block
// or on the move.

// countBreak counts a block p breaks: all blocks, logs, leaves, and what a
// hammer crushes.
func countBreak(p *player.Player, b world.Block, hammer bool) {
	count(p, "block_break", 1)
	switch b.(type) {
	case block.Log:
		count(p, "log_break", 1)
	case block.Leaves:
		count(p, "leaves_break", 1)
	}
	if !hammer {
		return
	}
	switch b.(type) {
	case block.Stone:
		count(p, "hammer_stone", 1)
	case block.Cobblestone:
		count(p, "hammer_cobble", 1)
	case block.Gravel:
		count(p, "hammer_gravel", 1)
	case block.Sand:
		count(p, "hammer_sand", 1)
	}
}

// countWalk counts the blocks p walks to newPos: the horizontal distance of
// a move, kept as a fraction until it adds up to a block. A jump of more
// than a few blocks is a teleport, not walking.
func (s *session) countWalk(p *player.Player, newPos mgl64.Vec3) {
	from := p.Position()
	d := math.Hypot(newPos[0]-from[0], newPos[2]-from[2])
	if d == 0 || d > 4 {
		return
	}
	if s.walked == nil {
		s.walked = map[*world.EntityHandle]float64{}
	}
	acc := s.walked[p.H()] + d
	if whole := math.Floor(acc); whole >= 1 {
		count(p, "walk", int64(whole))
		acc -= whole
	}
	s.walked[p.H()] = acc
}
