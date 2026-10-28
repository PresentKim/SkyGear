package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// growTreeAt grows an oak tree at pos (on dirt or grass) the way bone meal
// on a sapling does.
func growTreeAt(tx *world.Tx, pos cube.Pos) {
	sapling := block.Sapling{Type: block.OakSapling()}
	tx.SetBlock(pos, sapling, nil)
	for range 200 {
		b, ok := tx.Block(pos).(block.Sapling)
		if !ok {
			return
		}
		b.BoneMeal(pos, tx)
	}
}
