package fairy

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Fairy magic: the fairy leaves fairy
// dust, which makes fairy emeralds and the fairy board. The board casts the
// spells that change the island's weather and sun, an emerald each.

func init() {
	world.RegisterItem(sgitem.FairyDust{})
	world.RegisterItem(sgitem.FairyEmerald{})
	world.RegisterItem(Board{})
	FairyRecipes()
}

// Board is held and used to open the spells; it is not placed.
type Board struct{}

func (Board) EncodeItem() (string, int16) { return "skygear:fairy_board", 0 }

func (Board) Name() string { return i18n.T(i18n.Default, "name.FairyBoard") }

func (Board) Category() category.Category { return category.Equipment() }

func (Board) MaxCount() int { return 1 }

// FairyRecipes registers dust, the middle item and dust in a row: an emerald
// makes a fairy emerald, a research table the fairy board. Three wide, they
// only fit a crafting table.
func FairyRecipes() {
	d := item.NewStack(sgitem.FairyDust{}, 1)
	row := func(mid world.Item, out world.Item) {
		recipe.Register(recipe.NewShaped([]recipe.Item{d, item.NewStack(mid, 1), d}, item.NewStack(out, 1), recipe.NewShape(3, 1), "crafting_table"))
	}
	row(item.Emerald{}, sgitem.FairyEmerald{})
	row(progress.ResearchTable{}, Board{})
}
