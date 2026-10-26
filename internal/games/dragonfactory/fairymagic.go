package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

// Fairy magic: the fairy leaves fairy
// dust, which makes fairy emeralds and the fairy board. The board casts the
// spells that change the island's weather and sun, an emerald each.

// FairyBoard is held and used to open the spells; it is not placed.
type FairyBoard struct{}

func (FairyBoard) EncodeItem() (string, int16) { return "dragonfactory:fairy_board", 0 }
func (FairyBoard) Name() string                { return i18n.T(i18n.Default, "name.FairyBoard") }
func (FairyBoard) Category() category.Category { return category.Equipment() }
func (FairyBoard) MaxCount() int               { return 1 }

// fairyRecipes registers dust, the middle item and dust in a row: an emerald
// makes a fairy emerald, a research table the fairy board. Three wide, they
// only fit a crafting table.
func fairyRecipes() {
	d := item.NewStack(dfitem.FairyDust{}, 1)
	row := func(mid world.Item, out world.Item) {
		recipe.Register(recipe.NewShaped([]recipe.Item{d, item.NewStack(mid, 1), d}, item.NewStack(out, 1), recipe.NewShape(3, 1), "crafting_table"))
	}
	row(item.Emerald{}, dfitem.FairyEmerald{})
	row(ResearchTable{}, FairyBoard{})
}

func init() {
	world.RegisterItem(dfitem.FairyDust{})
	world.RegisterItem(dfitem.FairyEmerald{})
	world.RegisterItem(FairyBoard{})
	fairyRecipes()
}
