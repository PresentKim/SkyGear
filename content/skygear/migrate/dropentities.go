package migrate

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/presentkim/skygear/common/nbtrewrite"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// A block that Blocks replaces with DropEntities set had a block entity the
// new block has no use for. The world decodes a block entity by the block now
// at its position and logs an error for every block without NBT on every
// load, so the block entities at the positions of such blocks are dropped
// with the rewrite.

// blockEntityTag is the last byte of the key of a chunk's block entities.
const blockEntityTag = 0x31

// chunkPos is the position of a block in a world, as block entities save it.
type chunkPos [3]int32

// subChunkPrefix returns the key of the chunk a sub chunk key belongs to.
func subChunkPrefix(k []byte) []byte { return k[:len(k)-2] }

// dropPositions finds, for each chunk of the world, the positions of the
// blocks whose palette entry r.drops names. The result is keyed by the key of
// the chunk (a sub chunk key without its last 2 bytes).
func dropPositions(log *slog.Logger, dir string, ldb *leveldb.DB, r renamer) (map[string]map[chunkPos]bool, error) {
	if r.drops == nil {
		return nil, nil
	}
	out := map[string]map[chunkPos]bool{}
	it := ldb.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		k, v := it.Key(), it.Value()
		if !nbtrewrite.IsSubChunkKey(k) || !r.mentions(v) {
			continue
		}
		ps, err := subChunkMatches(v, r.drops)
		if err != nil {
			log.Warn("migrate: block entities of a sub chunk not checked", "world", dir, "key", fmt.Sprintf("%x", k), "err", err)
			continue
		}
		if len(ps) == 0 {
			continue
		}
		cx := int32(binary.LittleEndian.Uint32(k))
		cz := int32(binary.LittleEndian.Uint32(k[4:]))
		y0 := int32(int8(k[len(k)-1])) * 16
		set := out[string(subChunkPrefix(k))]
		if set == nil {
			set = map[chunkPos]bool{}
			out[string(subChunkPrefix(k))] = set
		}
		for _, p := range ps {
			set[chunkPos{cx*16 + p[0], y0 + p[1], cz*16 + p[2]}] = true
		}
	}
	return out, it.Error()
}

// subChunkMatches returns the (x, y, z) in the sub chunk of the blocks of its
// first layer (the one the world reads a block entity's block from; a block
// in the second layer has no block entity of its own) whose palette entry name drop accepts.
func subChunkMatches(b []byte, drop func(string) bool) ([][3]int32, error) {
	if len(b) < 1 {
		return nil, nbtrewrite.ErrShort
	}
	rest := b[1:]
	switch b[0] {
	case 1:
	case 8:
		rest = rest[min(1, len(rest)):]
	case 9:
		rest = rest[min(2, len(rest)):]
	default:
		return nil, errors.New("sub chunk version")
	}
	if len(rest) < 1 {
		return nil, nbtrewrite.ErrShort
	}
	bits := int(rest[0] >> 1)
	rest = rest[1:]
	if bits == 0x7f {
		return nil, nil
	}
	words := 0
	if bits != 0 {
		words = 4096 / (32 / bits)
		if bits == 3 || bits == 5 || bits == 6 {
			words++
		}
	}
	if len(rest) < words*4 {
		return nil, nbtrewrite.ErrShort
	}
	indices := rest[:words*4]
	rest = rest[words*4:]
	count := 1
	if bits != 0 {
		if len(rest) < 4 {
			return nil, nbtrewrite.ErrShort
		}
		count = int(int32(binary.LittleEndian.Uint32(rest)))
		rest = rest[4:]
	}
	dec := nbt.NewDecoderWithEncoding(bytes.NewReader(rest), nbt.LittleEndian)
	hit := make([]bool, max(count, 0))
	found := false
	for i := range hit {
		var e map[string]any
		if err := dec.Decode(&e); err != nil {
			return nil, err
		}
		name, _ := e["name"].(string)
		if hit[i] = drop(name); hit[i] {
			found = true
		}
	}
	if !found {
		return nil, nil
	}
	var out [][3]int32
	for x := range int32(16) {
		for z := range int32(16) {
			for y := range int32(16) {
				idx := 0
				if bits != 0 {
					off := int(x<<8|z<<4|y) * bits
					w := binary.LittleEndian.Uint32(indices[off/(32/bits*bits)*4:])
					idx = int(w>>(off%(32/bits*bits))) & (1<<bits - 1)
				}
				if idx < len(hit) && hit[idx] {
					out = append(out, [3]int32{x, y, z})
				}
			}
		}
	}
	return out, nil
}

// dropBlockEntities returns v, the block entities of a chunk (root tags one
// after another), without those at the positions in drop. The tags that stay
// are copied byte for byte. It returns nil when nothing is left.
func dropBlockEntities(v []byte, drop map[chunkPos]bool) ([]byte, error) {
	rd := bytes.NewReader(v)
	dec := nbt.NewDecoderWithEncoding(rd, nbt.LittleEndian)
	var out []byte
	for rd.Len() > 0 {
		start := len(v) - rd.Len()
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		end := len(v) - rd.Len()
		x, okx := m["x"].(int32)
		y, oky := m["y"].(int32)
		z, okz := m["z"].(int32)
		if okx && oky && okz && drop[chunkPos{x, y, z}] {
			continue
		}
		out = append(out, v[start:end]...)
	}
	return out, nil
}
