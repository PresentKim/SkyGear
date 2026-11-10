// Package bridge is the bridging assist ("reach-around" placement, as the
// BridgingMod Java mod does it): a sneaking player who holds a block and
// uses it on the air places it in the first empty cell along their view that
// has a block to build off on its side away from the view, such as the gap
// in front of the block they stand on at an edge. Without sneaking, a use on
// the air does nothing, as in the vanilla game. While sneaking with a block,
// the block is shown in the cell to that player only, with green stained
// glass on its liquid layer (as the hammer's vein preview), and a mark next
// to the crosshair (the HUD's Crosshair part) shows whether the block goes
// up, down or to the side. One press of the use button places one block,
// however long it is held. Players turn the assist off with Setting.
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
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/platform/account"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Setting is the player setting (account.Setting) that turns the assist on.
// It is on unless turned off.
const Setting = "bridge_assist"

// Reach is how far along the view a cell is looked for, in blocks.
const Reach = 5.0

// alignment is how closely a side must face along the view to be built off
// (the dot product of the view and the side's direction, as BridgingMod).
const alignment = 0.1

// pressGap is how long the use button must go without a use for the next
// use to start a new press. The client repeats the use of a held button
// every few ticks (about 200ms) and sometimes sends one click twice, so uses
// closer together than this are one press, and one press places one block.
// Two deliberate clicks are rarely closer than this.
const pressGap = 300 * time.Millisecond

// updateInterval limits how often moving updates the preview and the
// crosshair mark.
const updateInterval = 100 * time.Millisecond

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

// Cell is the cell the block goes in.
func (t Target) Cell() cube.Pos {
	return t.Support.Side(t.Face)
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
// one of its sides that face back towards the player. The player sneaks, so
// a block that does something when used (a chest) is built off too.
func Find(tx *world.Tx, eye, dir mgl64.Vec3, body cube.BBox, held world.Block) (Target, bool) {
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
			if !buildable(tx, support, f, held) {
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
// pos: a solid face, not a liquid nor a block held would replace. A block
// that does something when used (a chest) is fine: the player sneaks, so
// the use places the block and never opens it.
func buildable(tx *world.Tx, pos cube.Pos, face cube.Face, held world.Block) bool {
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

// heldBlock is the block p holds in held, if the assist is on for p, p
// sneaks and p may build.
func heldBlock(p *player.Player, held item.Stack, sneaking bool) (world.Block, bool) {
	if !sneaking || held.Empty() || !p.GameMode().AllowsEditing() || !account.On(p, Setting) {
		return nil, false
	}
	b, ok := held.Item().(world.Block)
	return b, ok
}

// target is the cell p, at pos with rotation rot and sneaking or not, places
// held in.
func target(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, held item.Stack, sneaking bool) (Target, bool) {
	b, ok := heldBlock(p, held, sneaking)
	if !ok {
		return Target{}, false
	}
	eye := pos.Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	body := p.H().Type().BBox(p).Translate(pos)
	return Find(tx, eye, rot.Vec3(), body, b)
}

// newPress reports whether a use at now with the client's trigger starts a
// new press of the use button, given the last use of the same press chain
// at last (zero: none). A repeat of a held button
// (TriggerTypeSimulationTick) never does; any other use does once the
// button went pressGap without a use, which also folds a click the client
// sent twice into one.
func newPress(last, now time.Time, trigger uint32) bool {
	if trigger == protocol.TriggerTypeSimulationTick {
		return false
	}
	return last.IsZero() || now.Sub(last) >= pressGap
}

// press records a use of the button by p at now and reports whether it
// starts a new press.
func press(s *core.Session, p *player.Player, now time.Time) bool {
	uses := lastUse.Of(s)
	fresh := newPress(uses[p.H()], now, connOf(sghook.Session(p)).lastTrigger())
	uses[p.H()] = now
	return fresh
}

// Place places p's held block in the cell the assist finds, the way using
// it on the support's face would (the game's building rules and protection
// apply), when p used the held item on the air. It reports whether the use
// was the assist's (p sneaks and holds a block with the assist on): the
// caller cancels the item's own use then. Only the first use of a press
// places; the repeats of a held button are swallowed.
func Place(s *core.Session, p *player.Player) bool {
	held, _ := p.HeldItems()
	if _, ok := heldBlock(p, held, p.Sneaking()); !ok {
		return false
	}
	tx := p.Tx()
	if fresh := press(s, p, time.Now()); fresh {
		if t, ok := target(p, tx, p.Position(), p.Rotation(), held, true); ok {
			p.UseItemOnBlock(t.Support, t.Face, t.ClickPos)
		}
	}
	// a click on the preview: the client put a block of its own next to it
	if g, ok := connOf(sghook.Session(p)).takeGhost(); ok {
		restoreCell(p, tx, g)
	}
	return true
}

// Holding reports whether a use of p's held item on a block continues a
// press that began on the air with the assist: the client, once the placed
// block is under the crosshair, repeats the held button as a use on that
// block. The caller cancels such a use, so a press places one block.
func Holding(s *core.Session, p *player.Player) bool {
	uses := lastUse.Of(s)
	last, ok := uses[p.H()]
	if !ok {
		return false
	}
	now := time.Now()
	if newPress(last, now, protocol.TriggerTypeUnknown) {
		delete(uses, p.H())
		return false
	}
	uses[p.H()] = now
	return true
}

// Update works out p's preview and crosshair mark for a player at
// pos with rotation rot, sneaking or not and holding held, unless it was
// updated less than updateInterval ago (force: always). It reports whether
// the mark changed: the caller sends the HUD again then. A player who looks
// at a block gets neither: using the held block places it on that block.
func Update(s *core.Session, p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, held item.Stack, sneaking, force bool) bool {
	if _, ok := heldBlock(p, held, sneaking); !ok {
		// the cheap way out for nearly every move: nothing to show
		hidePreview(s, p)
		return setMark(s, p, "")
	}
	now := time.Now()
	if !force && now.Sub(lastUpdate.Of(s)[p.H()]) < updateInterval {
		return false
	}
	lastUpdate.Of(s)[p.H()] = now
	var (
		t     Target
		found bool
	)
	if _, _, looking := look.LookedAt(p, tx, pos, rot); !looking {
		t, found = target(p, tx, pos, rot, held, sneaking)
	}
	if !found {
		hidePreview(s, p)
		return setMark(s, p, "")
	}
	b, _ := heldBlock(p, held, sneaking)
	showPreview(s, p, tx, t.Cell(), b)
	return setMark(s, p, t.Mark())
}

// setMark sets p's crosshair mark and reports whether it changed.
func setMark(s *core.Session, p *player.Player, mark string) bool {
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

// preview is the cell a player is shown the block in.
type preview struct {
	cell  cube.Pos
	block world.Block
}

// previewBlock is what fills the preview's liquid layer, so the shown
// block reads as not real yet.
var previewBlock world.Block = block.StainedGlass{Colour: item.ColourGreen()}

// showPreview shows p, only, b in cell with green stained glass on its
// liquid layer (as the hammer's vein preview), sending it again only when
// the cell or the block changed.
func showPreview(s *core.Session, p *player.Player, tx *world.Tx, cell cube.Pos, b world.Block) {
	cur, ok := previews.Of(s)[p.H()]
	if ok && cur.cell == cell && cur.block == b {
		return
	}
	if ok && cur.cell != cell {
		restoreCell(p, tx, cur.cell)
	}
	sghook.ViewFakeBlock(p, cell, b)
	sendLayer(p, cell, previewBlock)
	previews.Of(s)[p.H()] = preview{cell: cell, block: b}
	connOf(sghook.Session(p)).setPreview(cell, true)
}

// hidePreview shows p the real cell again, if p is shown a preview.
func hidePreview(s *core.Session, p *player.Player) {
	cur, ok := previews.Of(s)[p.H()]
	if !ok {
		return
	}
	delete(previews.Of(s), p.H())
	connOf(sghook.Session(p)).setPreview(cube.Pos{}, false)
	if tx := p.Tx(); tx != nil {
		restoreCell(p, tx, cur.cell)
	}
}

// restoreCell shows p the real blocks of both layers at pos.
func restoreCell(p *player.Player, tx *world.Tx, pos cube.Pos) {
	sghook.ViewFakeBlock(p, pos, tx.Block(pos))
	var liquid world.Block = block.Air{}
	if l, ok := tx.Liquid(pos); ok {
		liquid = l
	}
	sendLayer(p, pos, liquid)
}

// sendLayer shows p b on the liquid layer (layer 1) at pos.
func sendLayer(p *player.Player, pos cube.Pos, b world.Block) {
	sghook.WritePacket(sghook.Session(p), &packet.UpdateBlock{
		Position:          protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		NewBlockRuntimeID: world.BlockRuntimeID(b),
		Flags:             packet.BlockUpdateNetwork,
		Layer:             1,
	})
}

var (
	// marks are the crosshair marks players were last given.
	marks = core.NewPlayers[string]()
	// previews are the cells players are shown a block in.
	previews = core.NewPlayers[preview]()
	// lastUpdate is when each player's preview and mark were last worked
	// out.
	lastUpdate = core.NewPlayers[time.Time]()
	// lastUse is when each player last used a block with the assist, the
	// end of the current press so far.
	lastUse = core.NewPlayers[time.Time]()
)

// PreviewCell is the cell p is shown the preview in, if any.
func PreviewCell(s *core.Session, p *player.Player) (cube.Pos, bool) {
	cur, ok := previews.Of(s)[p.H()]
	return cur.cell, ok
}
