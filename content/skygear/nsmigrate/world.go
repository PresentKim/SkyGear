package nsmigrate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

// convertWorld rewrites the old names in an mcdb world folder: block palettes
// of the sub chunks, block entities, entities and any other NBT value that
// holds them. It returns the number of values changed.
func convertWorld(log *slog.Logger, dir string) (int, error) {
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "db"), &opt.Options{BlockSize: 16 * opt.KiB})
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", dir, err)
	}
	defer ldb.Close()

	batch := new(leveldb.Batch)
	changed := 0
	it := ldb.NewIterator(nil, nil)
	for it.Next() {
		k, v := it.Key(), it.Value()
		if !hasOld(v) && !hasDFKey(v) {
			continue
		}
		nv, err := convertWorldValue(k, v)
		if err != nil {
			if !hasOld(v) {
				continue // df_ was only bytes of a value that is no NBT
			}
			log.Warn("namespace conversion: value left as it is", "world", dir, "key", fmt.Sprintf("%x", k), "err", err)
			continue
		}
		if !bytes.Equal(nv, v) {
			batch.Put(append([]byte(nil), k...), nv)
			changed++
		}
	}
	it.Release()
	if err := it.Error(); err != nil {
		return 0, err
	}
	if err := ldb.Write(batch, nil); err != nil {
		return 0, err
	}
	return changed, nil
}

// convertWorldValue rewrites one value: a sub chunk by its block palettes,
// anything else as appended NBT tags.
func convertWorldValue(k, v []byte) ([]byte, error) {
	if isSubChunkKey(k) {
		return rewriteSubChunk(v)
	}
	return rewriteNBT(v, replaceOld)
}

// isSubChunkKey: x, z (and a dimension) followed by '/' and the sub chunk y.
func isSubChunkKey(k []byte) bool {
	return (len(k) == 10 || len(k) == 14) && k[len(k)-2] == '/'
}

// rewriteSubChunk rewrites the NBT block palettes of a disk sub chunk
// (versions 1, 8 and 9) and copies the indices as they are.
func rewriteSubChunk(b []byte) ([]byte, error) {
	if len(b) < 1 {
		return nil, errShort
	}
	out := []byte{b[0]}
	rest := b[1:]
	storages := 1
	switch b[0] {
	case 1:
	case 8, 9:
		head := 1
		if b[0] == 9 {
			head = 2
		}
		if len(rest) < head {
			return nil, errShort
		}
		storages = int(rest[0])
		out = append(out, rest[:head]...)
		rest = rest[head:]
	default:
		return nil, fmt.Errorf("sub chunk version %d", b[0])
	}
	for i := 0; i < storages; i++ {
		if len(rest) < 1 {
			return nil, errShort
		}
		bits := int(rest[0] >> 1)
		out = append(out, rest[0])
		rest = rest[1:]
		if bits == 0x7f {
			continue
		}
		words := 0
		if bits != 0 {
			words = 4096 / (32 / bits)
			if bits == 3 || bits == 5 || bits == 6 {
				words++
			}
		}
		if len(rest) < words*4 {
			return nil, errShort
		}
		out = append(out, rest[:words*4]...)
		rest = rest[words*4:]
		count := 1
		if bits != 0 {
			if len(rest) < 4 {
				return nil, errShort
			}
			count = int(int32(binary.LittleEndian.Uint32(rest)))
			out = append(out, rest[:4]...)
			rest = rest[4:]
		}
		palette, after, err := rewriteNBTRoots(rest, count, replaceOld)
		if err != nil {
			return nil, err
		}
		out = append(out, palette...)
		rest = after
	}
	return append(out, rest...), nil
}
