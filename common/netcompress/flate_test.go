package netcompress

import (
	"bytes"
	"compress/flate"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// chunkBatch returns data shaped like a batch of chunk payloads: runs of
// block palette indices with some noise.
func chunkBatch() []byte {
	r := rand.New(rand.NewPCG(1, 2))
	b := make([]byte, 0, 1<<17)
	for len(b) < 1<<17 {
		v := byte(r.IntN(12))
		n := 1 + r.IntN(40)
		for range n {
			b = append(b, v)
		}
		if r.IntN(4) == 0 {
			b = append(b, byte(r.IntN(256)))
		}
	}
	return b
}

func TestFlateLevelRoundTrip(t *testing.T) {
	data := chunkBatch()
	for level := 1; level <= 9; level++ {
		c := NewFlate(level)
		if c.EncodeCompression() != packet.FlateCompression.EncodeCompression() {
			t.Fatalf("level %d: compression ID differs from Flate", level)
		}
		out, err := c.Compress(data)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(flate.NewReader(bytes.NewReader(out)))
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("level %d: standard decoder: err=%v equal=%v", level, err, bytes.Equal(got, data))
		}
		got, err = c.Decompress(out, len(data)+1)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("level %d: own decoder: %v", level, err)
		}
	}
}

func TestFlateLevelClamped(t *testing.T) {
	for _, l := range []int{-5, 0, 10, 99} {
		out, err := NewFlate(l).Compress([]byte("hello hello hello"))
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(flate.NewReader(bytes.NewReader(out)))
		if string(got) != "hello hello hello" {
			t.Fatalf("level %d: %q", l, got)
		}
	}
}

func BenchmarkFlateLevels(b *testing.B) {
	data := chunkBatch()
	for _, level := range []int{1, 3, 6} {
		c := NewFlate(level)
		b.Run("level"+string(rune('0'+level)), func(b *testing.B) {
			var n int
			for b.Loop() {
				out, _ := c.Compress(data)
				n = len(out)
			}
			b.ReportMetric(float64(n), "bytes")
		})
	}
}

// A batch compressed by a client with the Flate ID decodes on a connection
// whose compression is an installed Flate (the decoder checks that the
// algorithm found by the ID is the connection's own).
func TestInstalledFlateDecodesClientBatch(t *testing.T) {
	c := Install(3)
	defer packet.RegisterCompression(packet.FlateCompression)

	var buf bytes.Buffer
	enc := packet.NewEncoder(&buf)
	enc.EnableCompression(packet.FlateCompression, 0)
	want := []byte("login packet payload")
	if err := enc.Encode([][]byte{want}); err != nil {
		t.Fatal(err)
	}
	dec := packet.NewDecoder(&buf)
	dec.EnableCompression(c, 1<<20)
	got, err := dec.Decode()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || !bytes.Equal(got[0], want) {
		t.Fatalf("decoded %q, want %q", got, want)
	}
}
