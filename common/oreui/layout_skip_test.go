package oreui

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestSetLayoutSameBytesSkipped(t *testing.T) {
	n := 0
	f := &Form{nonce: "n", write: func(packet.Packet) error { n++; return nil }}
	for _, c := range []struct {
		layout any
		want   int
	}{{"a", 1}, {"a", 1}, {"b", 2}, {"b", 2}, {"a", 3}} {
		if err := f.SetLayout(c.layout); err != nil {
			t.Fatal(err)
		}
		if n != c.want {
			t.Fatalf("layout %v: %d sends, want %d", c.layout, n, c.want)
		}
	}
}
