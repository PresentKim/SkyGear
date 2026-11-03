// Package netcompress holds the packet compression of the network layer.
package netcompress

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Flate is a packet.Compression with the ID of the standard Flate
// compression and a configurable level. Clients cannot tell it from
// packet.FlateCompression: only the encoder differs.
type Flate struct {
	level int
	pool  *sync.Pool
}

// NewFlate returns a Flate compression of the level (1 fastest to 9
// smallest; out-of-range values give the default 6).
func NewFlate(level int) Flate {
	if level < 1 || level > 9 {
		level = 6
	}
	return Flate{level: level, pool: &sync.Pool{New: func() any {
		w, _ := flate.NewWriter(io.Discard, level)
		return w
	}}}
}

// Level returns the compression level.
func (f Flate) Level() int { return f.level }

// EncodeCompression implements packet.Compression.
func (Flate) EncodeCompression() uint16 { return packet.FlateCompression.EncodeCompression() }

// Compress implements packet.Compression.
func (f Flate) Compress(decompressed []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(len(decompressed)/3 + 64)
	w := f.pool.Get().(*flate.Writer)
	defer f.pool.Put(w)
	w.Reset(&buf)
	if _, err := w.Write(decompressed); err != nil {
		return nil, fmt.Errorf("compress flate: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close flate writer: %w", err)
	}
	return buf.Bytes(), nil
}

// Decompress implements packet.Compression.
func (Flate) Decompress(compressed []byte, limit int) ([]byte, error) {
	return packet.FlateCompression.Decompress(compressed, limit)
}
