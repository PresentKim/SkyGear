// Package preview shows players blocks that are not in the world: the cells
// the construction wand and the bridging assist would fill, and the blocks
// the iron hammer's vein would break. Only the player is sent the blocks
// (UpdateBlock on layer 0, the block, and layer 1, the liquid layer); the
// world is unchanged.
//
// A cell to be filled gets Glass on layer 0 and the block it would get on
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
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Glass is the see-through green glass of the previews: drawn with the
// vanilla green stained glass (the atlas key GlassTexture points at
// textures/blocks/glass_green), with no collision or selection box. It is
// only sent to players, never placed in a world, and has no item.
type Glass struct{}

// GlassTexture is the atlas key of Glass's texture.
const GlassTexture = "skygear_preview_glass"

var glassHash = block.NextHash()

// EncodeBlock ...
func (Glass) EncodeBlock() (string, map[string]any) { return "skygear:preview_glass", nil }

// Hash ...
func (Glass) Hash() (uint64, uint64) { return glassHash, 0 }

// Model ...
func (Glass) Model() world.BlockModel { return model.Empty{} }

// Properties are a cube of blended glass with no boxes.
func (Glass) Properties() customblock.Properties {
	return customblock.Properties{
		Cube:        true,
		Textures:    map[string]customblock.Material{"*": customblock.NewMaterial(GlassTexture, customblock.BlendRenderMethod())},
		NoCollision: true,
		NoSelection: true,
	}
}

func init() { world.RegisterBlock(Glass{}) }

// Look is what a player is shown at a cell: the blocks of its two layers, a
// nil layer being the real one.
type Look struct {
	Block, Liquid world.Block
}

// Fill is the look of an empty cell that would get b: Glass, with b on the
// liquid layer.
func Fill(b world.Block) Look { return Look{Block: Glass{}, Liquid: b} }

// Mark is the look of a block that is marked (to be broken): the block
// itself, with Glass on the liquid layer.
func Mark() Look { return Look{Liquid: Glass{}} }

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
	var out []View
	switch {
	case l.Block != nil:
		out = append(out, View{Pos: pos, Layer: 0, Block: l.Block})
	case was.Block != nil:
		out = append(out, View{Pos: pos, Layer: 0, Block: tx.Block(pos)})
	}
	switch {
	case l.Liquid != nil:
		out = append(out, View{Pos: pos, Layer: 1, Block: l.Liquid})
	case was.Liquid != nil:
		var liquid world.Block = block.Air{}
		if w, ok := tx.Liquid(pos); ok {
			liquid = w
		}
		out = append(out, View{Pos: pos, Layer: 1, Block: liquid})
	}
	return out
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
