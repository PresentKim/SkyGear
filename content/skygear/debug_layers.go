package skygear

import (
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
)

// /sgdebug layers: a manual probe of the storage layers of a sub chunk. A
// real client is asked what it draws on layer 0 (block), layer 1 (liquid) and
// layers 2 and up (unused by vanilla) for the layer probe block, sent in a
// chunk (relog) and in an UpdateBlock (the command, alt), and whether a state
// change shows on those layers. Nothing in the game uses it.

type debugLayers struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"layers"`
	Mode layersEnum     `cmd:"mode"`
}

type layersEnum string

func (layersEnum) Type() string { return "sg_layers" }
func (layersEnum) Options(cmd.Source) []string {
	return []string{"a", "b", "c", "d", "alt", "alt1", "info", "clear"}
}

// layersTargets is the last target (cube.Pos) of each session.
var layersTargets sync.Map

// layerCount is how many layers alt, info and clear look at, from layer 0.
const layerCount = 4

func (c debugLayers) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	mode := string(c.Mode)
	pos, have := cube.Pos{}, false
	if v, ok := layersTargets.Load(s); ok {
		pos, have = v.(cube.Pos), true
	}
	switch mode {
	case "a", "b", "c", "d":
		// In front of the player, 2 blocks away at eye height.
		f := p.Rotation().Direction().Face()
		pos = cube.PosFromVec3(p.Position().Add(mgl64.Vec3{0, p.EyeHeight()})).Side(f).Side(f)
		layersTargets.Store(s, pos)
		clearLayers(tx, pos)
		switch mode {
		case "a":
			tx.SetBlockLayer(pos, LayerProbe{Kind: 0}, 0)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 1}, 1)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 2}, 2)
		case "b":
			// Layer 0 stays air (clearLayers).
			tx.SetBlockLayer(pos, LayerProbe{Kind: 1}, 1)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 2}, 2)
		case "c":
			// Layer 1 is air, the storage is there for layers 2 and 3.
			tx.SetBlockLayer(pos, LayerProbe{Kind: 0}, 0)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 2}, 2)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 1}, 3)
		case "d":
			// Layer 0 is a block that draws nothing, layer 1 an item probe.
			tx.SetBlockLayer(pos, LayerProbe{Kind: probeHidden}, 0)
			tx.SetBlockLayer(pos, LayerProbe{Kind: 1}, 1)
		}
		o.Printf(i18n.T(lang, "skygear.debug.layers."+mode), pos[0], pos[1], pos[2])
		return
	}
	if !have {
		o.Error(i18n.T(lang, "skygear.debug.layers.none"))
		return
	}
	switch mode {
	case "alt":
		n := 0
		for l := uint8(0); l < layerCount; l++ {
			if b, ok := tx.BlockLayer(pos, l).(LayerProbe); ok {
				b.Alt = 1 - b.Alt
				tx.SetBlockLayer(pos, b, l)
				n++
			}
		}
		o.Printf(i18n.T(lang, "skygear.debug.layers.alt"), n, pos[0], pos[1], pos[2])
	case "alt1":
		n := 0
		if b, ok := tx.BlockLayer(pos, 1).(LayerProbe); ok {
			b.Alt = 1 - b.Alt
			tx.SetBlockLayer(pos, b, 1)
			n++
		}
		o.Printf(i18n.T(lang, "skygear.debug.layers.alt1"), n, pos[0], pos[1], pos[2])
	case "info":
		var names [layerCount]any
		for l := range names {
			names[l] = layerLabel(tx.BlockLayer(pos, uint8(l)))
		}
		o.Printf(i18n.T(lang, "skygear.debug.layers.info"), pos[0], pos[1], pos[2], tx.SubChunkLayers(pos), names[0], names[1], names[2], names[3])
	case "clear":
		clearLayers(tx, pos)
		o.Printf(i18n.T(lang, "skygear.debug.layers.clear"), pos[0], pos[1], pos[2])
	}
}

// clearLayers sets air on layers 3, 2 and 1, then on layer 0 (SetBlock moves a
// layer 1 block down to layer 0 when it sets air, so layer 1 goes first).
func clearLayers(tx *world.Tx, pos cube.Pos) {
	for l := uint8(layerCount - 1); l >= 1; l-- {
		tx.SetBlockLayer(pos, nil, l)
	}
	tx.SetBlock(pos, block.Air{}, nil)
}

// layerLabel names what a layer holds for info: "air", "probe <kind>/<alt>" or
// the block's identifier.
func layerLabel(b world.Block) string {
	if _, air := b.(block.Air); air {
		return "air"
	}
	if p, ok := b.(LayerProbe); ok {
		return "probe " + probeKindNames[p.Kind] + "/" + string(rune('0'+p.Alt))
	}
	name, _ := b.EncodeBlock()
	return strings.TrimPrefix(name, "minecraft:")
}
