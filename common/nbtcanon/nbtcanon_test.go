package nbtcanon

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// TestOneFormPerItem: the same tree encodes to the same bytes whatever the
// order of the NBT, and NBT turns the form back into an NBT of the same tree.
func TestOneFormPerItem(t *testing.T) {
	m := map[string]any{"Name": "a:b", "Count": byte(1), "Damage": int16(3),
		"tag": map[string]any{"display": map[string]any{"Name": "n", "Lore": []any{"x", "y"}}, "Ench": []int32{1, 2}}}
	first := Encode(mustNBT(t, m))
	for range 20 {
		if got := Encode(mustNBT(t, m)); !bytes.Equal(got, first) {
			t.Fatal("two encodings of one tree differ")
		}
	}
	back, err := NBT(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(Encode(back), first) {
		t.Error("the NBT of a form does not encode to that form")
	}
	if _, err := NBT([]byte{cMap, 0xff}); err == nil {
		t.Error("a broken form was read")
	}
}

func mustNBT(t *testing.T, m map[string]any) []byte {
	t.Helper()
	b, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
