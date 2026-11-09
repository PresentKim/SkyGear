// Package bridge is the bridging assist ("reach-around" placement, as the
// BridgingMod Java mod does it): a player who holds a block and uses it on
// the air places it in the first empty cell along their view that has a
// block to build off on its side away from the view, such as the gap in
// front of the block they stand on at an edge. A mark next to the crosshair
// (the HUD's Crosshair part) shows when such a cell is in view and whether
// the block goes up, down or to the side. Players turn it off with Setting.
package bridge

import (
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/platform/account"
)

// Setting is the player setting (account.Setting) that turns the assist on.
// It is on unless turned off.
const Setting = "bridge_assist"

// Reach is how far along the view a cell is looked for, in blocks.
const Reach = 5.0

// alignment is how closely a side must face along the view to be built off
// (the dot product of the view and the side's direction, as BridgingMod).
const alignment = 0.1

// placeInterval is the shortest time between two assisted placements, like
// the repeat of a held use button (4 ticks).
const placeInterval = 200 * time.Millisecond

// updateInterval limits how often moving updates the crosshair mark.
const updateInterval = 50 * time.Millisecond

// The marks of the HUD's Crosshair part: the direction the block goes from
// the block it is built off.
const (
	MarkUp   = "up"
	MarkDown = "down"
	MarkSide = "side"
)

// Target is a cell the assist places in: the block at Support is clicked on
// its Face (Support.Side(Face) is the cell) at ClickPos, relative to
// Support.
type Target struct {
	Support  cube.Pos
	Face     cube.Face
	ClickPos mgl64.Vec3
}

// Mark is the crosshair mark of the target.
func (t Target) Mark() string {
	switch t.Face {
	case cube.FaceUp:
		return MarkUp
	case cube.FaceDown:
		return MarkDown
	}
	return MarkSide
}

// Find returns the cell a player with eye position eye and view direction
// dir places held in: the first cell along the view, within Reach, that is
// empty, does not overlap the player's box and has a block to build off on
// one of its sides that face back towards the player.
func Find(tx *world.Tx, eye, dir mgl64.Vec3, body cube.BBox, held world.Block, sneaking bool) (Target, bool) {
	var (
		t     Target
		found bool
	)
	end := eye.Add(dir.Mul(Reach))
	trace.TraverseBlocks(eye, end, func(pos cube.Pos) bool {
		if !fillable(tx, pos, held) {
			return true
		}
		if pos.OutOfBounds(tx.Range()) || body.IntersectsWith(cube.Box(0, 0, 0, 1, 1, 1).Translate(pos.Vec3())) {
			return true
		}
		for _, f := range cube.Faces() {
			// f is the face of the support the block is built on: it faces
			// along the view, so the support is behind the cell, on the
			// side the view comes from.
			if dir.Dot(faceNormal(f)) < alignment {
				continue
			}
			support := pos.Side(f.Opposite())
			if !buildable(tx, support, f, held, sneaking) {
				continue
			}
			t, found = Target{Support: support, Face: f, ClickPos: clickPos(eye, dir, support, f)}, true
			return false
		}
		return true
	})
	return t, found
}

// fillable reports whether held may go in pos: air, or a block held replaces
// (grass, a flower).
func fillable(tx *world.Tx, pos cube.Pos, held world.Block) bool {
	b := tx.Block(pos)
	if _, air := b.(block.Air); air {
		return true
	}
	r, ok := b.(block.Replaceable)
	return ok && r.ReplaceableBy(held)
}

// buildable reports whether a block can be built on face of the block at
// pos: a solid face, not a liquid nor a block held would replace, and not a
// block that does something when used (a chest, a door) unless sneaking,
// so the assist never opens one.
func buildable(tx *world.Tx, pos cube.Pos, face cube.Face, held world.Block, sneaking bool) bool {
	b := tx.Block(pos)
	if _, air := b.(block.Air); air {
		return false
	}
	if _, liquid := b.(world.Liquid); liquid {
		return false
	}
	if r, ok := b.(block.Replaceable); ok && r.ReplaceableBy(held) {
		return false
	}
	if _, ok := b.(block.Activatable); ok && !sneaking {
		return false
	}
	return b.Model().FaceSolid(pos, face, tx)
}

// faceNormal is the unit vector out of face f.
func faceNormal(f cube.Face) mgl64.Vec3 {
	return cube.Pos{}.Side(f).Vec3()
}

// clickPos is where the view crosses the plane of the support's face,
// relative to the support and kept on the face, so a slab or stairs goes on
// the half the player aims at; the face's centre if the view runs along the
// plane.
func clickPos(eye, dir mgl64.Vec3, support cube.Pos, face cube.Face) mgl64.Vec3 {
	n := faceNormal(face)
	centre := support.Vec3Centre().Add(n.Mul(0.5))
	click := centre
	if d := dir.Dot(n); math.Abs(d) > 1e-6 {
		if k := centre.Sub(eye).Dot(n) / d; k > 0 {
			click = eye.Add(dir.Mul(k))
		}
	}
	rel := click.Sub(support.Vec3())
	for i := range 3 {
		rel[i] = min(max(rel[i], 0), 1)
	}
	axis := int(face.Axis())
	rel[axis] = centre.Sub(support.Vec3())[axis]
	return rel
}

// heldBlock is the block p holds in held, if the assist is on for p and p
// may build.
func heldBlock(p *player.Player, held item.Stack) (world.Block, bool) {
	if held.Empty() || !p.GameMode().AllowsEditing() || !account.On(p, Setting) {
		return nil, false
	}
	b, ok := held.Item().(world.Block)
	return b, ok
}

// target is the cell p, at pos with rotation rot, places held in.
func target(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, held item.Stack) (Target, bool) {
	b, ok := heldBlock(p, held)
	if !ok {
		return Target{}, false
	}
	eye := pos.Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	body := p.H().Type().BBox(p).Translate(pos)
	return Find(tx, eye, rot.Vec3(), body, b, p.Sneaking())
}

// Place places p's held block in the cell the assist finds, the way using
// it on the support's face would (the game's building rules and protection
// apply), when p used the held item on the air. It reports whether the use
// was the assist's: the caller cancels the item's own use then.
func Place(s *core.Session, p *player.Player) bool {
	held, _ := p.HeldItems()
	t, ok := target(p, p.Tx(), p.Position(), p.Rotation(), held)
	if !ok {
		return false
	}
	now := time.Now()
	if now.Sub(lastPlace.Of(s)[p.H()]) >= placeInterval {
		lastPlace.Of(s)[p.H()] = now
		p.UseItemOnBlock(t.Support, t.Face, t.ClickPos)
	}
	return true
}

// Update works out p's crosshair mark for a player at pos with rotation rot
// holding held, unless it was updated less than updateInterval ago (force:
// always), and reports whether it changed: the caller sends the HUD again
// then. A player who looks at a block gets no mark: using the held block
// places it on that block.
func Update(s *core.Session, p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, held item.Stack, force bool) bool {
	now := time.Now()
	if !force && now.Sub(lastUpdate.Of(s)[p.H()]) < updateInterval {
		return false
	}
	lastUpdate.Of(s)[p.H()] = now
	mark := ""
	if _, _, looking := look.LookedAt(p, tx, pos, rot); !looking {
		if t, ok := target(p, tx, pos, rot, held); ok {
			mark = t.Mark()
		}
	}
	marks := marks.Of(s)
	if marks[p.H()] == mark {
		return false
	}
	if mark == "" {
		delete(marks, p.H())
	} else {
		marks[p.H()] = mark
	}
	return true
}

// Mark is p's crosshair mark (MarkUp, MarkDown, MarkSide), or "" for none.
func Mark(s *core.Session, p *player.Player) string {
	return marks.Of(s)[p.H()]
}

var (
	// marks are the crosshair marks players were last given.
	marks = core.NewPlayers[string]()
	// lastUpdate is when each player's mark was last worked out.
	lastUpdate = core.NewPlayers[time.Time]()
	// lastPlace is when each player last placed a block with the assist.
	lastPlace = core.NewPlayers[time.Time]()
)
