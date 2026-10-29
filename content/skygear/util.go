package skygear

import (
	_ "embed"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// Requirement is an item asked for: a research's cost, an expansion's
// price.
type Requirement struct {
	Item  world.Item
	Count int
}

var oakLog = block.Log{Wood: block.OakWood()}
