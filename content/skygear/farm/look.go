package farm

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Waila adds how full of cocoons the leaves are to their panel
// (waila.Provider).
func (l SilkwormLeaves) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = []string{fmt.Sprintf(i18n.T(v.Lang, "look.cocoon"), l.CocoonPercent())}
	return info
}

// Waila adds a barrel's contents to its panel (waila.Provider): its lines and
// its liquid on the icon line.
func (b Barrel) Waila(v waila.View, info waila.Info) waila.Info {
	s := core.At(v.Tx)
	if s == nil {
		return info
	}
	info.Lines = BarrelLines(s, v.Lang, v.Tx, v.Pos)
	info.Line = BarrelLiquid(s, v.Lang, v.Pos)
	return info
}

// Waila fills in a magic plant's panel (waila.Provider): its seed as the
// name and the icon, and its time or yield.
func (b MagicPlant) Waila(v waila.View, info waila.Info) waila.Info {
	i := PlantKindIndex(b.Kind)
	if i < 0 {
		return info
	}
	k := PlantKinds[i]
	seed := k.SeedStack()
	info.Item, info.NameKey = seed.Item(), ""
	info.Name = i18n.T(v.Lang, "pseed."+k.Seed)
	info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
	if s := core.At(v.Tx); s != nil {
		info.Lines = PlantLines(s, v.Lang, v.Pos, b, k)
	}
	return info
}

// registerLook fills in the panels of the vanilla blocks the farm runs: the
// composter's contents, and a crop's name and time.
func registerLook() {
	waila.RegisterAdapter(func(v waila.View, b world.Block, info waila.Info) waila.Info {
		if c, ok := b.(block.Composter); ok {
			if s := core.At(v.Tx); s != nil {
				info.Lines = ComposterLines(s, v.Lang, v.Pos, c)
			}
		}
		if kind, _, ok := CropOf(b); ok {
			info.Name = i18n.T(v.Lang, CropNameKeys[kind])
			info.Tools = []waila.ToolIcon{{Tool: waila.Crops, Mark: waila.MarkNone}}
			if s := core.At(v.Tx); s != nil {
				if r, ok := StateOf(s).Crops[CropKey(v.Pos)]; ok {
					info.Lines = CropLines(s, v.Lang, r, v.Tx, v.Pos)
				}
			}
		}
		return info
	})
}
