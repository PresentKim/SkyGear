package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
)

// LiquidIcon is an item only the WAILA panel shows: the liquid a barrel holds,
// drawn with the barrel's own liquid texture (water, witch water, lava). It is
// in no recipe and not in the creative inventory.
type LiquidIcon struct{ Kind string }

// Liquid icon kinds.
const (
	LiquidWater      = "water"
	LiquidWitchWater = "witch_water"
	LiquidLava       = "lava"
)

var LiquidKinds = []string{LiquidWater, LiquidWitchWater, LiquidLava}

func (l LiquidIcon) EncodeItem() (string, int16) { return "skygear:liquid_" + l.Kind, 0 }

func (l LiquidIcon) Name() string { return i18n.T(i18n.Default, "liquid."+l.Kind) }

func (LiquidIcon) Category() category.Category { return category.Items() }

// liquidColours are the colours of the liquid names in the panel.
var liquidColours = map[string]string{LiquidWater: "§b", LiquidWitchWater: "§d", LiquidLava: "§c"}

// LiquidLine is the panel line of a liquid: its icon and name, and what is
// said of it.
func LiquidLine(lang, kind, text string) *waila.ItemLine {
	it := LiquidIcon{Kind: kind}
	return &waila.ItemLine{Item: item.NewStack(it, 1), Name: liquidColours[kind] + naming.CustomItemName(lang, it) + " ", Text: text}
}
