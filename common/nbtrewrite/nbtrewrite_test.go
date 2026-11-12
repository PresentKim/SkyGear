package nbtrewrite

import (
	"bytes"
	"reflect"
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

func paletteEntry(t *testing.T, name string, states map[string]any) []byte {
	e, err := nbt.MarshalEncoding(map[string]any{"name": name, "states": states, "version": int32(18090528)}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// twoEntrySubChunk is a version 9 sub chunk with one layer of 1 bit indices
// and the palette [air, old].
func twoEntrySubChunk(t *testing.T) (sub, words []byte) {
	words = make([]byte, 128*4)
	for i := range words {
		words[i] = byte(i * 7)
	}
	sub = append([]byte{9, 1, 0, 1 << 1}, words...)
	sub = append(sub, 2, 0, 0, 0)
	sub = append(sub, paletteEntry(t, "minecraft:air", map[string]any{})...)
	sub = append(sub, paletteEntry(t, "a:old", map[string]any{"x": "keep"})...)
	return sub, words
}

func decodeEntries(t *testing.T, b []byte) (out []map[string]any) {
	b = b[4+128*4+4:]
	for len(b) > 0 {
		d := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian)
		var m map[string]any
		if err := d.Decode(&m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
		n, _ := nbt.MarshalEncoding(m, nbt.LittleEndian)
		b = b[len(n):]
	}
	return out
}

func TestSubChunkBlocksReplacesEntry(t *testing.T) {
	sub, words := twoEntrySubChunk(t)
	hook := func(name string) (string, map[string]any, bool) {
		if name == "a:old" {
			return "a:new", map[string]any{"s": "10"}, true
		}
		return "", nil, false
	}
	out, err := SubChunkBlocks(sub, upper, hook)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out[:4+len(words)+4], sub[:4+len(words)+4]) {
		t.Error("header, index words or palette count changed")
	}
	es := decodeEntries(t, out)
	if len(es) != 2 || es[0]["name"] != "minecraft:air" || es[1]["name"] != "a:new" {
		t.Fatalf("entries %v", es)
	}
	if st, _ := es[1]["states"].(map[string]any); len(st) != 1 || st["s"] != "10" {
		t.Errorf("states %v", es[1]["states"])
	}
	if es[1]["version"] != int32(18090528) {
		t.Errorf("version %v", es[1]["version"])
	}
	if again, err := SubChunkBlocks(out, upper, hook); err != nil || !bytes.Equal(again, out) {
		t.Errorf("second pass changed the sub chunk: %v", err)
	}
}

func TestSubChunkBlocksNilIsSubChunk(t *testing.T) {
	sub, _ := twoEntrySubChunk(t)
	rep := func(s string) string { return strings.Replace(s, "old", "renamed", 1) }
	a, err := SubChunk(sub, rep)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SubChunkBlocks(sub, rep, nil)
	if err != nil || !bytes.Equal(a, b) {
		t.Errorf("SubChunkBlocks(nil) differs: %v", err)
	}
}

// TestSubChunkBlocksEveryLayer: version 8 and 9, two layers, the entry in
// both; an entry the hook does not name is untouched.
func TestSubChunkBlocksEveryLayer(t *testing.T) {
	one, _ := twoEntrySubChunk(t)
	body := one[3:]
	sub := append([]byte{9, 2, 0}, body...)
	sub = append(sub, body...)
	out, err := SubChunkBlocks(sub, nil2, func(n string) (string, map[string]any, bool) {
		return "a:new", nil, n == "a:old"
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("a:old")) || bytes.Count(out, []byte("a:new")) != 2 || bytes.Count(out, []byte("minecraft:air")) != 2 {
		t.Errorf("layers not both rewritten")
	}
}

func nil2(s string) string { return s }

// TestNBTStacks: a stack whose name plain accepts keeps its new name, its
// count and the tags that are not the stack's own, and loses its tag, damage
// and block; stacks inside other stacks and lists are found; with a nil plain
// the result is NBTKeys.
func TestNBTStacks(t *testing.T) {
	enc := func(m map[string]any) []byte {
		b, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	old := func(name string, slot byte) map[string]any {
		return map[string]any{"Name": name, "Count": byte(3), "Damage": int16(9), "Slot": slot,
			"Block": map[string]any{"name": name},
			"tag":   map[string]any{"Damage": int32(4), "display": map[string]any{"Name": "x", "Lore": []string{"l"}}, "dragonflyData": []byte{1}}}
	}
	in := enc(map[string]any{
		"Item": old("a:sword", 0),
		"Items": []any{
			old("a:sword", 1),
			map[string]any{"Name": "a:keep", "Count": byte(1), "tag": map[string]any{"Items": []any{old("a:sword", 2)}}},
		},
	})
	up := func(s string) string { return strings.Replace(s, "a:sword", "a:ingot", 1) }
	out, err := NBTStacks(in, up, up, func(n string) bool { return n == "a:sword" })
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Item  map[string]any
		Items []map[string]any
	}
	if err := nbt.UnmarshalEncoding(out, &m, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	for i, st := range []map[string]any{m.Item, m.Items[0], m.Items[1]["tag"].(map[string]any)["Items"].([]any)[0].(map[string]any)} {
		want := map[string]any{"Name": "a:ingot", "Count": byte(3), "Damage": int16(0), "Slot": byte(i)}
		if !reflect.DeepEqual(st, want) {
			t.Errorf("stack %d = %v, want %v", i, st, want)
		}
	}
	if m.Items[1]["Name"] != "a:keep" || m.Items[1]["Count"] != byte(1) {
		t.Errorf("a stack that is not plain changed: %v", m.Items[1])
	}
	if got, _ := NBTStacks(in, up, up, nil); !bytes.Equal(got, mustKeys(t, in, up)) {
		t.Error("a nil plain differs from NBTKeys")
	}
	if _, err := NBTStacks(in[:len(in)-5], up, up, func(string) bool { return true }); err == nil {
		t.Error("a cut value is no error")
	}
}

func mustKeys(t *testing.T, in []byte, f func(string) string) []byte {
	t.Helper()
	b, err := NBTKeys(in, f, f)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
