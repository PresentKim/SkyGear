package nbtrewrite

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func upper(s string) string {
	if s == "old:block" {
		return "new:block"
	}
	return s
}

func TestSubChunkRewritesThePalette(t *testing.T) {
	entry, err := nbt.MarshalEncoding(map[string]any{"name": "old:block", "version": int32(1)}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	sub := append([]byte{9, 1, 0, 0 << 1}, entry...)
	if !IsSubChunkKey([]byte("\x00\x00\x00\x00\x00\x00\x00\x00/\x00")) {
		t.Fatal("a sub chunk key is not recognised")
	}
	out, err := SubChunk(sub, upper)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("new:block")) || bytes.Contains(out, []byte("old:block")) {
		t.Errorf("palette not rewritten: %q", out)
	}
	if !bytes.Equal(out[:4], sub[:4]) {
		t.Errorf("header changed: %v", out[:4])
	}
}

func TestNBTRewritesNestedStrings(t *testing.T) {
	in, err := nbt.MarshalEncoding(map[string]any{"Items": []any{map[string]any{"Name": "old:block"}}, "Other": "kept"}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NBT(in, upper)
	if err != nil {
		t.Fatal(err)
	}
	if s := string(out); !strings.Contains(s, "new:block") || !strings.Contains(s, "kept") {
		t.Errorf("rewritten %q", s)
	}
}
