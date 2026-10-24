package chestui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
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

func TestLocalizeSlot(t *testing.T) {
	saved := Localize
	t.Cleanup(func() { Localize = saved })
	Localize = func(lang string, it item.Stack) (string, []string, bool) {
		return lang + ":" + it.CustomName(), []string{lang}, true
	}
	s := NewStorage(9, "t")
	_ = s.Inventory().SetItem(0, item.NewStack(item.Apple{}, 1).WithCustomName("ko name"))
	w := &window{inv: s.Inventory()}
	rid, _, _ := world.ItemRuntimeID(item.Apple{})
	shared := map[string]any{"display": map[string]any{"Name": "ko name", "Lore": []string{"old"}}}
	apple := protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: rid}, NBTData: shared}
	localizeSlot("en", w, 0, &apple)
	display := apple.NBTData["display"].(map[string]any)
	if display["Name"] != "en:ko name" || len(display["Lore"].([]string)) != 1 || display["Lore"].([]string)[0] != "en" {
		t.Fatalf("display %v", display)
	}
	if shared["display"].(map[string]any)["Name"] != "ko name" {
		t.Fatal("changed the NBT it was given")
	}
	other := protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: rid + 1}, NBTData: map[string]any{}}
	localizeSlot("en", w, 0, &other)
	if _, ok := other.NBTData["display"]; ok {
		t.Fatal("localized a stack of another item")
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
