package dragonfactory

import (
	"image/color"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

// Vein mining (like the VeinMiner mod): the iron hammer, mining a crushable
// block while sneaking, also breaks the blocks of the same kind connected to
// it face to face, up to veinLimit and what the hammer's durability allows
// (each block costs one). Only blocks the hammer crushes (hammerDrop) count.
// While a player sneaks with the iron hammer and looks at such a block, the
// outline of what it would break shows in particles only that player sees.

// veinLimit is the most blocks one vein mining breaks, the mined one
// included.
const veinLimit = 64

// veinOutlineColour is the colour of the outline particles.
var veinOutlineColour = color.RGBA{R: 255, G: 214, B: 64, A: 255}

// veinBlocks are the blocks of the same kind as the one at start connected
// to it face to face, start first, up to limit; none when the hammer does not
// crush it. canBuild filters the blocks a player may break.
func veinBlocks(tx *world.Tx, start cube.Pos, limit int, canBuild func(cube.Pos) bool) []cube.Pos {
	b := tx.Block(start)
	it, ok := b.(world.Item)
	if _, crushable := hammerDrop(b); !ok || !crushable {
		return nil
	}
	key := itemKey(it)
	same := func(pos cube.Pos) bool {
		other, ok := tx.Block(pos).(world.Item)
		return ok && itemKey(other) == key && canBuild(pos)
	}
	out := []cube.Pos{start}
	seen := map[cube.Pos]bool{start: true}
	for i := 0; i < len(out) && len(out) < limit; i++ {
		for _, f := range cube.Faces() {
			n := out[i].Side(f)
			if seen[n] || n.OutOfBounds(tx.Range()) {
				continue
			}
			seen[n] = true
			if same(n) {
				if out = append(out, n); len(out) >= limit {
					break
				}
			}
		}
	}
	return out
}

// veinMine breaks the blocks connected to the one at pos p mines with an
// iron hammer while sneaking (pos itself is broken by the game): each drops
// what the hammer makes of it and costs one durability.
func (s *session) veinMine(p *player.Player, tx *world.Tx, pos cube.Pos) {
	held, off := p.HeldItems()
	if !isIronHammer(held) || !p.Sneaking() {
		return
	}
	// one durability is left for the block the game breaks itself
	left := held.Durability() - 1
	blocks := veinBlocks(tx, pos, veinLimit, s.canBuild)
	if len(blocks) < 2 || left <= 0 {
		return
	}
	broken := 0
	for _, at := range blocks[1:] {
		if broken >= left {
			break
		}
		b := tx.Block(at)
		drop, ok := hammerDrop(b)
		if !ok {
			continue
		}
		countBreak(p, b, true)
		tx.SetBlock(at, nil, nil)
		tx.AddParticle(at.Vec3Centre(), particle.BlockBreak{Block: b})
		s.dropAt(tx, at.Vec3Centre(), drop)
		broken++
	}
	if broken > 0 {
		p.SetHeldItems(held.Damage(broken), off)
	}
}

// veinEdges are the edges of the outline of a set of blocks: an edge is on
// the outline when one or three of the four cells around it are in the set,
// or two across from each other. Each edge is its start corner and the axis
// it runs along.
func veinEdges(blocks []cube.Pos) [][2]mgl64.Vec3 {
	in := map[cube.Pos]bool{}
	for _, b := range blocks {
		in[b] = true
	}
	type edge struct {
		at   cube.Pos
		axis int
	}
	seen := map[edge]bool{}
	var out [][2]mgl64.Vec3
	for _, b := range blocks {
		for axis := 0; axis < 3; axis++ {
			u, v := (axis+1)%3, (axis+2)%3
			for du := 0; du <= 1; du++ {
				for dv := 0; dv <= 1; dv++ {
					at := b
					at[u] += du
					at[v] += dv
					e := edge{at, axis}
					if seen[e] {
						continue
					}
					seen[e] = true
					// the four cells around the edge
					var cells [4]bool
					for i := 0; i < 4; i++ {
						c := at
						c[u] -= 1 - i&1
						c[v] -= 1 - i>>1
						cells[i] = in[c]
					}
					n := 0
					for _, c := range cells {
						if c {
							n++
						}
					}
					diagonal := n == 2 && cells[0] == cells[3]
					if n == 1 || n == 3 || diagonal {
						start := mgl64.Vec3{float64(at[0]), float64(at[1]), float64(at[2])}
						end := start
						end[axis]++
						out = append(out, [2]mgl64.Vec3{start, end})
					}
				}
			}
		}
	}
	return out
}

// veinOutlinePoints is how many particles one edge gets, and veinOutlineMax
// the most particles one outline sends.
const (
	veinOutlinePoints = 4
	veinOutlineMax    = 480
)

// showVeinOutline shows p, sneaking with the iron hammer, the outline of the
// blocks it would break mining the block at pos.
func (s *session) showVeinOutline(p *player.Player, tx *world.Tx, pos cube.Pos) {
	if held, _ := p.HeldItems(); !isIronHammer(held) || !p.Sneaking() || !s.canBuild(pos) {
		return
	}
	blocks := veinBlocks(tx, pos, veinLimit, s.canBuild)
	if len(blocks) == 0 {
		return
	}
	edges := veinEdges(blocks)
	step := 1
	if n := len(edges) * veinOutlinePoints; n > veinOutlineMax {
		step = (n + veinOutlineMax - 1) / veinOutlineMax
	}
	dust := particle.Dust{Colour: veinOutlineColour}
	i := 0
	for _, e := range edges {
		for k := 0; k < veinOutlinePoints; k++ {
			if i++; i%step != 0 {
				continue
			}
			t := (float64(k) + 0.5) / veinOutlinePoints
			p.ShowParticle(e[0].Add(e[1].Sub(e[0]).Mul(t)), dust)
		}
	}
}
