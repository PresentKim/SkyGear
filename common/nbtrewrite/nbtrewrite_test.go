package nbtrewrite

import (
	"bytes"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
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

// TestNBTKeys: tag names and strings go through rep, the value keys of item
// stacks through key.
func TestNBTKeys(t *testing.T) {
	m := item.WriteNBT(item.NewStack(item.Stick{}, 1).WithValue("Card", "old:block"), true)
	m["df_tag"] = "old:block"
	in, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NBTKeys(in, upper, strings.ToLower)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := nbt.UnmarshalEncoding(out, &got, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if got["df_tag"] != "new:block" {
		t.Errorf("string value or tag name: %v", got)
	}
	s := item.ReadNBT(got, nil)
	if v, ok := s.Value("card"); !ok || v != "new:block" {
		t.Errorf("stack value = %v, %v", v, ok)
	}
	if _, ok := s.Value("Card"); ok {
		t.Error("stack key not passed through key")
	}
	same, err := NBT(in, upper)
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := NBTKeys(in, upper, upper); !bytes.Equal(same, want) {
		t.Error("NBT differs from NBTKeys(rep, rep)")
	}
}
