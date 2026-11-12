// Package preview shows players blocks that are not in the world: the cells
// the construction wand would fill, and the blocks the iron hammer's vein
// would break. Only the player is sent the blocks (UpdateBlock on layer 0,
// the block, and layer 1, the liquid layer); the world is unchanged.
//
// A cell to be filled gets Glass (in the feature's colour) on layer 0 and the block it would get on
// layer 1. Glass has no selection or collision box, so the crosshair, the
// clicks and the player go through the cell to what is behind it, and the
// block still shows. A block to be broken keeps its layer 0 and gets Glass
// on layer 1.
package preview

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Glass is the see-through glass of the previews in one of the sixteen
// colours, drawn with the vanilla stained glass of that colour (the atlas
// key GlassTexture points at textures/blocks/glass_<colour>). Its collision
// and selection boxes are a point at the cell's middle, so the client finds
// nothing to collide with or aim at: the crosshair, the clicks and the
// player go through the cell to what is behind it. It is only sent to
// players, never placed in a world, and has no item.
type Glass struct{ Colour item.Colour }

// Each feature shows its preview in its own colour.
var (
	// WandColour is the construction wand's cells to fill.
	WandColour = item.ColourGreen()
	// VeinColour is the iron hammer's blocks to break.
	VeinColour = item.ColourBlue()
)

// GlassTexture is the atlas key of the texture of the glass of colour c.
func GlassTexture(c item.Colour) string { return "skygear_preview_glass_" + c.String() }

var glassHash = block.NextHash()

// EncodeBlock ...
func (g Glass) EncodeBlock() (string, map[string]any) {
	return "skygear:preview_glass_" + g.Colour.String(), nil
}

// Hash ...
func (g Glass) Hash() (uint64, uint64) { return glassHash, uint64(g.Colour.Uint8()) }

// Model ...
func (Glass) Model() world.BlockModel { return model.Empty{} }

// noBox is the boxes Glass is sent with: a point at the cell's middle. An
// empty box (all zero) would mean no box set and send the full cube, so the
// point sits off the corner.
var noBox = cube.Box(0.5, 0.5, 0.5, 0.5, 0.5, 0.5)

// GlassGeometry is the model of Glass (models/blocks/preview_glass.geo.json):
// a cube a tenth of a pixel bigger than the cell on every side. A glass the
// size of the cell lies in the faces of a solid block it shares the cell
// with (the iron hammer's glass on the liquid layer of a block to be
// broken): the client draws the block's faces first and the glass, in the
// same plane, fails the depth test and does not show.
const GlassGeometry = "geometry.skygear_preview_glass"

// Properties are a slightly bigger cube of blended glass with boxes of no
// size.
func (g Glass) Properties() customblock.Properties {
	return customblock.Properties{
		Geometry:     GlassGeometry,
		Textures:     map[string]customblock.Material{"*": customblock.NewMaterial(GlassTexture(g.Colour), customblock.BlendRenderMethod())},
		CollisionBox: noBox,
		SelectionBox: noBox,
	}
}

func init() {
	for _, c := range item.Colours() {
		world.RegisterBlock(Glass{Colour: c})
	}
}

// Look is what a player is shown at a cell: the blocks of its two layers, a
// nil layer being the real one.
type Look struct {
	Block, Liquid world.Block
}

// Fill is the look of an empty cell that would get b: the glass of colour
// c, with b on the liquid layer.
func Fill(c item.Colour, b world.Block) Look { return Look{Block: Glass{Colour: c}, Liquid: b} }

// Mark is the look of a block that is marked (to be broken): the block
// itself, with the glass of colour c on the liquid layer.
func Mark(c item.Colour) Look { return Look{Liquid: Glass{Colour: c}} }

// View is a block a player is sent on one layer of a cell.
type View struct {
	Pos   cube.Pos
	Layer uint32
	Block world.Block
}

// Send shows p a View (the tests replace it).
var Send = func(p *player.Player, v View) { sghook.ViewFakeBlockLayer(p, v.Pos, v.Block, v.Layer) }

// shown are the looks each player is shown, by the preview (owner) that
// shows them.
var shown = core.NewPlayers[map[string]map[cube.Pos]Look]()

// Set makes owner's preview for p the looks: the cells it no longer has
// show what another preview shows there, else the real blocks again, and
// the cells it has show its looks (over another preview's: the last set
// shows).
func Set(s *core.Session, p *player.Player, tx *world.Tx, owner string, looks map[cube.Pos]Look) {
	all := shown.Of(s)[p.H()]
	if all == nil {
		all = map[string]map[cube.Pos]Look{}
		shown.Of(s)[p.H()] = all
	}
	old := all[owner]
	if len(looks) == 0 {
		delete(all, owner)
	} else {
		all[owner] = looks
	}
	for pos, l := range old {
		if _, kept := looks[pos]; kept {
			continue
		}
		if other, ok := shownBy(all, pos); ok {
			send(p, tx, pos, other, l)
		} else {
			send(p, tx, pos, Look{}, l)
		}
	}
	for pos, l := range looks {
		if o, ok := old[pos]; ok && same(o, l) {
			continue
		}
		send(p, tx, pos, l, old[pos])
	}
	if len(all) == 0 {
		delete(shown.Of(s), p.H())
	}
}

// Clear removes owner's preview for p (Set with none).
func Clear(s *core.Session, p *player.Player, tx *world.Tx, owner string) {
	if _, ok := shown.Of(s)[p.H()][owner]; ok {
		Set(s, p, tx, owner, nil)
	}
}

// Shown reports whether owner shows p a preview.
func Shown(s *core.Session, p *player.Player, owner string) bool {
	_, ok := shown.Of(s)[p.H()][owner]
	return ok
}

// shownBy is the look another preview shows at pos, if any.
func shownBy(all map[string]map[cube.Pos]Look, pos cube.Pos) (Look, bool) {
	for _, looks := range all {
		if l, ok := looks[pos]; ok {
			return l, true
		}
	}
	return Look{}, false
}

// Views are what p is sent to show l at pos where it was shown was: each
// layer l sets, and each layer was set that l leaves to the world, real.
func Views(tx *world.Tx, pos cube.Pos, l, was Look) []View {
	var block0, liquid []View
	switch {
	case l.Block != nil:
		block0 = []View{{Pos: pos, Layer: 0, Block: l.Block}}
	case was.Block != nil:
		block0 = []View{{Pos: pos, Layer: 0, Block: tx.Block(pos)}}
	}
	switch {
	case l.Liquid != nil:
		liquid = []View{{Pos: pos, Layer: 1, Block: l.Liquid}}
	case was.Liquid != nil:
		var w world.Block = block.Air{}
		if lq, ok := tx.Liquid(pos); ok {
			w = lq
		}
		liquid = []View{{Pos: pos, Layer: 1, Block: w}}
	}
	// The client moves a block on the liquid layer up to layer 0 when layer
	// 0 becomes air, and it stays there as a real block. So layer 0 is set
	// first when it gets the look's block (the glass), and last when it goes
	// back to the world's, after the liquid layer.
	if l.Block == nil {
		return append(liquid, block0...)
	}
	return append(block0, liquid...)
}

func send(p *player.Player, tx *world.Tx, pos cube.Pos, l, was Look) {
	if tx == nil {
		return
	}
	for _, v := range Views(tx, pos, l, was) {
		Send(p, v)
	}
}

func same(a, b Look) bool { return sameBlock(a.Block, b.Block) && sameBlock(a.Liquid, b.Liquid) }

func sameBlock(a, b world.Block) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return world.BlockRuntimeID(a) == world.BlockRuntimeID(b)
}
