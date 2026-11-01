package sgitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// DimensionStone opens the second dimension's window when used; it is never
// used up (dimension package).
type DimensionStone struct{}

func (DimensionStone) EncodeItem() (string, int16) { return "skygear:dimension_stone", 0 }

func (DimensionStone) Name() string { return i18n.T(i18n.Default, "name.DimensionStone") }

func (DimensionStone) Category() category.Category { return category.Equipment() }

func (DimensionStone) MaxCount() int { return 1 }

// Use does nothing itself (the game's handler opens the window first): being
// usable makes the client send a use on the air at all.
func (DimensionStone) Use(*world.Tx, item.User, *item.UseContext) bool { return false }
