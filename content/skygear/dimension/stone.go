package dimension

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The teleport stone: the research "dimension" gives one, and a fairy
// emerald makes another on a crafting table.
func init() {
	world.RegisterItem(sgitem.DimensionStone{})
	recipe.Register(recipe.NewShapeless([]recipe.Item{item.NewStack(sgitem.FairyEmerald{}, 1)}, item.NewStack(sgitem.DimensionStone{}, 1), "crafting_table"))
}
