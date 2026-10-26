package dragonfactory

import (
	_ "embed"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// Requirement is an item asked for: a research's cost, an expansion's
// price.
type Requirement struct {
	Item  world.Item
	Count int
}

var oakLog = block.Log{Wood: block.OakWood()}

// plain reports whether a stack is a plain item: no added NBT (custom
// name, lore, values such as a time-limited item's expiry). Research,
// machine recipes and prices ask for plain items; anything else is a
// different item to them.
func plain(st item.Stack) bool { return st.Comparable(item.NewStack(st.Item(), 1)) }
