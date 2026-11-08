package skygear

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
)

var _ world.ColumnHandler = (*worldHandler)(nil)

// HandleColumnLoad puts the bundles of a loaded column into the grids.
func (h *worldHandler) HandleColumnLoad(tx *world.Tx, _ world.ChunkPos, blockEntities map[cube.Pos]world.Block) {
	conduit.LoadColumn(h.s.Session, tx, blockEntities)
}

// HandleColumnUnload takes the bundles of a column out of the grids, before
// the column is saved.
func (h *worldHandler) HandleColumnUnload(tx *world.Tx, _ world.ChunkPos, blockEntities map[cube.Pos]world.Block) {
	conduit.UnloadColumn(h.s.Session, tx, blockEntities)
}
