package chunk_test

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

var cacheRange = cube.Range{-64, 319}

func mixedChunk(br world.BlockRegistry) *chunk.Chunk {
	c := chunk.New(br, cacheRange)
	n := uint32(len(br.Blocks()))
	// Sub chunk 0 stays air only; the others mix several palette entries.
	for y := int16(0); y < 200; y += 3 {
		c.SetBlock(uint8(y%16), y, uint8((y/16)%16), 0, uint32(y)%n)
	}
	return c
}

func TestDiskEncodeDeterministic(t *testing.T) {
	br := world.NewBlockRegistry()
	br.Finalize()
	c := mixedChunk(br)
	a := chunk.Encode(c, chunk.DiskEncoding)
	b := chunk.Encode(c, chunk.DiskEncoding)
	if len(a.SubChunks) != len(b.SubChunks) {
		t.Fatalf("sub chunk counts differ")
	}
	for i := range a.SubChunks {
		if !bytes.Equal(a.SubChunks[i], b.SubChunks[i]) {
			t.Fatalf("sub chunk %d differs between two encodings", i)
		}
	}
}

func TestDiskEncodeRoundTrip(t *testing.T) {
	br := world.NewBlockRegistry()
	br.Finalize()
	c := mixedChunk(br)
	d := chunk.Encode(c, chunk.DiskEncoding)
	got, err := chunk.DiskDecode(br, d, cacheRange)
	if err != nil {
		t.Fatal(err)
	}
	for y := cacheRange[0]; y <= cacheRange[1]; y += 1 {
		for x := uint8(0); x < 16; x += 5 {
			if want, have := c.Block(x, int16(y), 3, 0), got.Block(x, int16(y), 3, 0); want != have {
				t.Fatalf("block at %d,%d: %d != %d", x, y, have, want)
			}
		}
	}
	for y := int16(0); y < 200; y += 3 {
		x, z := uint8(y%16), uint8((y/16)%16)
		if want, have := c.Block(x, y, z, 0), got.Block(x, y, z, 0); want != have {
			t.Fatalf("block at %d,%d,%d: %d != %d", x, y, z, have, want)
		}
	}
}
