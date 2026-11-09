package world

import "github.com/df-mc/dragonfly/server/block/cube"

// SetBlockLayer stores b at pos on one storage layer and sends it to the chunk's viewers on that layer.
// Layer 0 is the block SetBlock sets and layer 1 the liquid layer; the chunk format and UpdateBlock carry
// more layers, which vanilla does not use. No block, neighbour or redstone updates, no hooks. nil b is air.
// It exists to probe what the client draws on each layer. skygear patch
func (tx *Tx) SetBlockLayer(pos cube.Pos, b Block, layer uint8) {
	w := tx.World()
	if pos.OutOfBounds(w.Range()) {
		return
	}
	if b == nil {
		b = w.conf.Blocks.Air()
	}
	c := tx.chunk(chunkPosFromBlockPos(pos))
	// Column.SetBlock bumps the version, which invalidates the sub chunk cache.
	c.SetBlock(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), layer, w.conf.Blocks.BlockRuntimeID(b))
	c.modified = true
	for _, v := range c.viewers {
		v.ViewBlockUpdate(pos, b, int(layer))
	}
}

// BlockLayer returns the block at pos on one storage layer (air when the sub chunk lacks it). skygear patch
func (tx *Tx) BlockLayer(pos cube.Pos, layer uint8) Block {
	w := tx.World()
	if pos.OutOfBounds(w.Range()) {
		return w.conf.Blocks.Air()
	}
	c := tx.chunk(chunkPosFromBlockPos(pos))
	return w.conf.Blocks.BlockByRuntimeIDOrAir(c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), layer))
}

// SubChunkLayers is how many storage layers the sub chunk holding pos has (0 if not loaded). skygear patch
func (tx *Tx) SubChunkLayers(pos cube.Pos) int {
	w := tx.World()
	if pos.OutOfBounds(w.Range()) {
		return 0
	}
	c, ok := w.chunks[chunkPosFromBlockPos(pos)]
	if !ok {
		return 0
	}
	return len(c.Sub()[c.SubIndex(int16(pos[1]))].Layers())
}
