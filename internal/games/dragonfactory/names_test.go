package dragonfactory

import (
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TestGameItemsHaveKoreanNames: every vanilla item the game names in its
// texts has a Korean name.
func TestGameItemsHaveKoreanNames(t *testing.T) {
	var items []world.Item
	for _, r := range researchTree {
		for _, c := range r.Cost {
			items = append(items, c.Item)
		}
	}
	for _, drops := range sieveTables["string"] {
		for _, d := range drops {
			items = append(items, d.Item.Item())
		}
	}
	for _, out := range crushRecipes {
		items = append(items, out)
	}
	for _, f := range giveItems() {
		items = append(items, f().Item())
	}
	for _, it := range items {
		if _, custom := it.(world.CustomItem); custom {
			continue
		}
		name, _ := it.EncodeItem()
		if got := itemName(i18n.Default, item.NewStack(it, 1)); "minecraft:"+got == name {
			t.Errorf("no Korean name for %s", name)
		}
	}
}

// TestNamedItemsHaveIcons: every vanilla item the game names also has a form
// icon (the lost-and-found and sell tables show both), and the menu item too.
func TestNamedItemsHaveIcons(t *testing.T) {
	for _, key := range i18n.Keys(i18n.Default) {
		name, ok0 := strings.CutPrefix(key, "vanilla.")
		if !ok0 {
			continue
		}
		it, ok := world.ItemByName("minecraft:"+name, 0)
		if !ok || name == "air" {
			continue
		}
		if iconOf(item.NewStack(it, 1)) == "" {
			t.Errorf("%s has no icon", itemKey(it))
		}
	}
	if iconOf(menuItem()) == "" {
		t.Error("menu item has no icon")
	}
}

func TestItemNamesInPlayerLanguage(t *testing.T) {
	saved := packNames
	t.Cleanup(func() { packNames = saved })
	if err := LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	essence := item.NewStack(ResearchEssence{Kind: EssenceEngineering}, 1)
	en := slotdeco.Strip(packLangFile(t, "en_US")["item.dragonfactory:research_essence_engineering.name"])
	if got := itemName("en", essence); en == "" || got != en {
		t.Errorf("english name %q, want %q", got, en)
	}
	if got, want := itemName(i18n.Default, essence), (ResearchEssence{Kind: EssenceEngineering}).Name(); got != want {
		t.Errorf("korean name %q, want %q", got, want)
	}
	if got := itemName("en", item.NewStack(item.Diamond{}, 1)); got != "Diamond" {
		t.Errorf("english vanilla name %q", got)
	}
	if got := blockName("en", Machine{Kind: KindCrusher}); got == blockName(i18n.Default, Machine{Kind: KindCrusher}) {
		t.Errorf("crusher block name %q in both languages", got)
	}
}

func TestItemRegistryNamesByTranslationKey(t *testing.T) {
	shown := map[string]any{"value": "공학의 정수"}
	shared := map[string]any{"components": map[string]any{"minecraft:display_name": shown}}
	pk := &packet.ItemRegistry{Items: []protocol.ItemEntry{
		{Name: "dragonfactory:research_essence_engineering", ComponentBased: true, Data: shared},
		{Name: "dragonfactory:sieve", Data: map[string]any{}},
	}}
	if localItemNames(nil, pk) {
		t.Fatal("dropped the item registry")
	}
	got := pk.Items[0].Data["components"].(map[string]any)["minecraft:display_name"].(map[string]any)["value"]
	if got != "item.dragonfactory:research_essence_engineering.name" {
		t.Errorf("display name %v", got)
	}
	if shown["value"] != "공학의 정수" {
		t.Error("changed the entry Dragonfly shares between players")
	}
}
