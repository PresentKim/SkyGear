package world

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

// skygear patch: tests of Tx.SetBlockLayer, BlockLayer and SubChunkLayers.

type layerViewer struct {
	NopViewer
	pos    []cube.Pos
	blocks []Block
	layers []int
}

func (v *layerViewer) ViewBlockUpdate(pos cube.Pos, b Block, layer int) {
	v.pos, v.blocks, v.layers = append(v.pos, pos), append(v.blocks, b), append(v.layers, layer)
}

func TestSetBlockLayer(t *testing.T) {
	w := hookWorld(NopProvider{}, nil)
	defer w.Close()
	air := w.conf.Blocks.Air()
	pos := cube.Pos{3, 64, 5}
	v := &layerViewer{}
	b := hookBlock{} // no state: the runtime ID is all a layer keeps

	<-w.exec(func(tx *Tx) {
		if n := tx.SubChunkLayers(pos); n != 0 {
			t.Fatalf("layers before the chunk is loaded: %d, want 0", n)
		}
		col := tx.chunk(chunkPosFromBlockPos(pos))
		col.viewers = append(col.viewers, v)
		version := col.version

		tx.SetBlockLayer(pos, b, 3)
		if n := tx.SubChunkLayers(pos); n < 4 {
			t.Fatalf("layers = %d, want >= 4", n)
		}
		if got := tx.BlockLayer(pos, 3); got != b {
			t.Fatalf("layer 3 = %#v, want %#v", got, b)
		}
		for l := uint8(0); l < 3; l++ {
			if got := tx.BlockLayer(pos, l); got != air {
				t.Fatalf("layer %d = %#v, want air", l, got)
			}
		}
		if tx.Block(pos) != air {
			t.Fatalf("Tx.Block sees the layer 3 block")
		}
		if col.version == version {
			t.Fatalf("column version not bumped: the sub chunk cache would go stale")
		}
		if !col.modified {
			t.Fatalf("column not marked modified")
		}
		if len(v.layers) != 1 || v.layers[0] != 3 || v.pos[0] != pos || v.blocks[0] != b {
			t.Fatalf("viewer got %v %v %v, want one update of layer 3", v.pos, v.blocks, v.layers)
		}

		// nil is air; out of bounds does nothing.
		tx.SetBlockLayer(pos, nil, 3)
		if got := tx.BlockLayer(pos, 3); got != air {
			t.Fatalf("layer 3 after nil = %#v, want air", got)
		}
		tx.SetBlockLayer(cube.Pos{0, 100000, 0}, b, 1)
		if len(v.layers) != 2 {
			t.Fatalf("out of bounds sent an update")
		}

		// The chunk round trips with the block on layer 3.
		tx.SetBlockLayer(pos, b, 3)
		x, y, z := uint8(pos[0]), int16(pos[1]), uint8(pos[2])
		rid := w.conf.Blocks.BlockRuntimeID(b)
		d := chunk.Encode(col.Chunk, chunk.DiskEncoding)
		disk, err := chunk.DiskDecode(w.conf.Blocks, d, w.Range())
		if err != nil {
			t.Fatal(err)
		}
		if got := disk.Block(x, y, z, 3); got != rid {
			t.Fatalf("disk round trip: layer 3 = %d, want %d", got, rid)
		}
		n := chunk.Encode(col.Chunk, chunk.NetworkEncoding)
		var raw []byte
		for _, s := range n.SubChunks {
			raw = append(raw, s...)
		}
		raw = append(raw, n.Biomes...)
		net, err := chunk.NetworkDecode(w.conf.Blocks, raw, len(n.SubChunks), w.Range())
		if err != nil {
			t.Fatal(err)
		}
		if got := net.Block(x, y, z, 3); got != rid {
			t.Fatalf("network round trip: layer 3 = %d, want %d", got, rid)
		}
		if got := net.Block(x, y, z, 0); got != w.conf.Blocks.BlockRuntimeID(air) {
			t.Fatalf("network round trip: layer 0 = %d, want air", got)
		}
	})
}
