package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/waila"
)

// LiquidIcon is an item only the WAILA panel shows: the liquid a barrel holds,
// drawn with the barrel's own liquid texture (water, witch water, lava). It is
// in no recipe and not in the creative inventory.
type LiquidIcon struct{ Kind string }

// Liquid icon kinds.
const (
	liquidWater      = "water"
	liquidWitchWater = "witch_water"
	liquidLava       = "lava"
)

var liquidKinds = []string{liquidWater, liquidWitchWater, liquidLava}

func (l LiquidIcon) EncodeItem() (string, int16) { return "dragonfactory:liquid_" + l.Kind, 0 }
func (l LiquidIcon) Name() string                { return i18n.T(i18n.Default, "liquid."+l.Kind) }
func (LiquidIcon) Category() category.Category   { return category.Items() }

// liquidColours are the colours of the liquid names in the panel.
var liquidColours = map[string]string{liquidWater: "§b", liquidWitchWater: "§d", liquidLava: "§c"}

// liquidLine is the panel line of a liquid: its icon and name, and what is
// said of it.
func liquidLine(lang, kind, text string) *waila.ItemLine {
	it := LiquidIcon{Kind: kind}
	return &waila.ItemLine{Item: item.NewStack(it, 1), Name: liquidColours[kind] + customItemName(lang, it) + " ", Text: text}
}

func init() {
	for _, k := range liquidKinds {
		world.RegisterItem(LiquidIcon{Kind: k})
	}
}
