package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/skygear/common/i18n"
)

// PetalColours are the colours of the petals, named like the dyes.
var PetalColours = []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black"}

// Petal is a petal of a magic flower in a colour.
type Petal struct{ Colour string }

func (p Petal) EncodeItem() (string, int16) { return "skygear:petal_" + p.Colour, 0 }

func (p Petal) Name() string {
	return i18n.T(i18n.Default, "pcolour."+p.Colour) + i18n.T(i18n.Default, "name.Petal.suffix")
}

func (Petal) Category() category.Category { return category.Items() }

// DyeColour is the dye colour of a petal colour.
func DyeColour(c string) item.Colour {
	return map[string]item.Colour{
		"white": item.ColourWhite(), "orange": item.ColourOrange(), "magenta": item.ColourMagenta(), "light_blue": item.ColourLightBlue(),
		"yellow": item.ColourYellow(), "lime": item.ColourLime(), "pink": item.ColourPink(), "gray": item.ColourGrey(),
		"light_gray": item.ColourLightGrey(), "cyan": item.ColourCyan(), "purple": item.ColourPurple(), "blue": item.ColourBlue(),
		"brown": item.ColourBrown(), "green": item.ColourGreen(), "red": item.ColourRed(), "black": item.ColourBlack(),
	}[c]
}
