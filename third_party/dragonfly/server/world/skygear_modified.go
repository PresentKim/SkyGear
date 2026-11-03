package world

import "github.com/df-mc/dragonfly/server/block/cube"

// MarkModified marks the loaded column holding pos modified, so that the next
// save stores it, without sending anything to viewers, updating neighbouring
// blocks or loading the column. Block entities whose data changes in place,
// such as the inventories of containers, use it. A nil Tx does nothing.
// skygear patch
func (tx *Tx) MarkModified(pos cube.Pos) {
	if tx == nil {
		return
	}
	if c, ok := tx.World().chunks[chunkPosFromBlockPos(pos)]; ok {
		c.modified = true
	}
}
