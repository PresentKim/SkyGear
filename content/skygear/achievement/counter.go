package achievement

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/platform/account"
)

// GrantCollection gives the first-possession achievements to the players who
// hold the items (advancement.InventoryNames). It runs every second.
func GrantCollection(held []advancement.PlayerItems) {
	for _, ph := range held {
		for _, h := range Haves {
			if ph.Names[h.Item] {
				account.Grant(ph.P, "sg_have_"+h.Short)
			}
		}
	}
}

// Count adds to a counter of p.
func Count(p *player.Player, key string, n int64) {
	if p != nil {
		account.Count(p, key, n)
	}
}

// CountAll adds to a counter of every player in the world, for what the
// island's machines and plants do on their own.
func CountAll(tx *world.Tx, key string, n int64) {
	for e := range tx.Players() {
		account.Count(e.(*player.Player), key, n)
	}
}

// CountBreak counts a block p breaks: all blocks, logs, leaves, and what a
// hammer crushes.
func CountBreak(p *player.Player, b world.Block, hammer bool) {
	Count(p, "block_break", 1)
	switch b.(type) {
	case block.Log:
		Count(p, "log_break", 1)
	case block.Leaves:
		Count(p, "leaves_break", 1)
	}
	if !hammer {
		return
	}
	switch b.(type) {
	case block.Stone:
		Count(p, "hammer_stone", 1)
	case block.Cobblestone:
		Count(p, "hammer_cobble", 1)
	case block.Gravel:
		Count(p, "hammer_gravel", 1)
	case block.Sand:
		Count(p, "hammer_sand", 1)
	}
}

// Walked is the part of a block each player walked since the last whole
// block was counted.
var Walked = core.NewPlayers[float64]()

// CountWalk counts the blocks p walks to newPos: the horizontal distance of
// a move, kept as a fraction until it adds up to a block. A jump of more
// than a few blocks is a teleport, not walking.
func CountWalk(s *core.Session, p *player.Player, newPos mgl64.Vec3) {
	from := p.Position()
	d := math.Hypot(newPos[0]-from[0], newPos[2]-from[2])
	if d == 0 || d > 4 {
		return
	}
	w := Walked.Of(s)
	acc := w[p.H()] + d
	if whole := math.Floor(acc); whole >= 1 {
		Count(p, "walk", int64(whole))
		acc -= whole
	}
	w[p.H()] = acc
}
