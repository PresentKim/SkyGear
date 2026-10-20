package chestui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestTitle(t *testing.T) {
	const r9 = "§r§r§r§r§r§r§r§r§r"
	for _, c := range []struct {
		size int
		want string
	}{
		{1, "§1§0" + r9 + "x"},
		{9, "§1§0" + r9 + "x"},
		{10, "§2§0" + r9 + "x"},
		{27, "§3§0" + r9 + "x"},
		{54, "§6§0" + r9 + "x"},
		{55, "§6§1" + r9 + "x"},
		{200, "§6§1" + r9 + "x"},
	} {
		if got := Title(c.size, "x"); got != c.want {
			t.Errorf("Title(%d) = %q, want %q", c.size, got, c.want)
		}
	}
}

func TestStorageOnChange(t *testing.T) {
	s := NewStorage(30, "t")
	var changed []int
	s.OnChange(func(slot int, _, _ item.Stack) { changed = append(changed, slot) })
	_ = s.Inventory().SetItem(29, item.NewStack(item.Apple{}, 3))
	if len(changed) != 1 || changed[0] != 29 {
		t.Fatalf("onChange %v", changed)
	}
}

func TestAddIdentifier(t *testing.T) {
	in, _ := nbt.Marshal(map[string]any{"idlist": []any{map[string]any{"id": "minecraft:pig"}}})
	var out struct {
		IDList []struct {
			ID string `nbt:"id"`
		} `nbt:"idlist"`
	}
	if err := nbt.Unmarshal(addIdentifier(in), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.IDList) != 2 || out.IDList[0].ID != "minecraft:pig" || out.IDList[1].ID != EntityIdentifier {
		t.Fatalf("idlist %+v", out.IDList)
	}
}
