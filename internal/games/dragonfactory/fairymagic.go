package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Fairy magic (docs/design/02-dragonfactory.md §19): the fairy leaves fairy
// dust, which makes fairy emeralds and the fairy board. The board casts the
// spells that change the island's weather and sun, an emerald each.

// FairyDust is what the fairy leaves: the stuff of the fairy magic.
type FairyDust struct{}

func (FairyDust) EncodeItem() (string, int16) { return "dragonfactory:fairy_dust", 0 }
func (FairyDust) Name() string                { return i18n.T(i18n.Default, "name.FairyDust") }
func (FairyDust) Category() category.Category { return category.Items() }

// FairyEmerald is spent by every spell of the fairy board.
type FairyEmerald struct{}

func (FairyEmerald) EncodeItem() (string, int16) { return "dragonfactory:fairy_emerald", 0 }
func (FairyEmerald) Name() string                { return i18n.T(i18n.Default, "name.FairyEmerald") }
func (FairyEmerald) Category() category.Category { return category.Items() }

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
	d := item.NewStack(FairyDust{}, 1)
	row := func(mid world.Item, out world.Item) {
		recipe.Register(recipe.NewShaped([]recipe.Item{d, item.NewStack(mid, 1), d}, item.NewStack(out, 1), recipe.NewShape(3, 1), "crafting_table"))
	}
	row(item.Emerald{}, FairyEmerald{})
	row(ResearchTable{}, FairyBoard{})
}

func init() {
	world.RegisterItem(FairyDust{})
	world.RegisterItem(FairyEmerald{})
	world.RegisterItem(FairyBoard{})
	fairyRecipes()
}
