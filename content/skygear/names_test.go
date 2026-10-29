package skygear

import (
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/slotdeco"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// TestGameItemsHaveKoreanNames: every vanilla item the game names in its
// texts has a Korean name.
func TestGameItemsHaveKoreanNames(t *testing.T) {
	var items []world.Item
	for _, r := range progress.Tree {
		for _, c := range r.Cost {
			items = append(items, c.Item)
		}
	}
	for _, drops := range sievetable.Tables["string"] {
		for _, d := range drops {
			items = append(items, d.Item.Item())
		}
	}
	for _, out := range sgitem.CrushRecipes {
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
		if got := naming.ItemName(i18n.Default, item.NewStack(it, 1)); "minecraft:"+got == name {
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
		if formicon.Of(item.NewStack(it, 1)) == "" {
			t.Errorf("%s has no icon", naming.ItemKey(it))
		}
	}
	if formicon.Of(screens.MenuItem(i18n.Default)) == "" {
		t.Error("menu item has no icon")
	}
}

func TestItemNamesInPlayerLanguage(t *testing.T) {
	saved := naming.PackNames
	t.Cleanup(func() { naming.PackNames = saved })
	if err := naming.LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	essence := item.NewStack(sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, 1)
	en := slotdeco.Strip(packLangFile(t, "en_US")["item.skygear:research_essence_engineering.name"])
	if got := naming.ItemName("en", essence); en == "" || got != en {
		t.Errorf("english name %q, want %q", got, en)
	}
	if got, want := naming.ItemName(i18n.Default, essence), (sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}).Name(); got != want {
		t.Errorf("korean name %q, want %q", got, want)
	}
	if got := naming.ItemName("en", item.NewStack(item.Diamond{}, 1)); got != "Diamond" {
		t.Errorf("english vanilla name %q", got)
	}
	if got := naming.BlockName("en", machine.Machine{Kind: machine.KindCrusher}); got == naming.BlockName(i18n.Default, machine.Machine{Kind: machine.KindCrusher}) {
		t.Errorf("crusher block name %q in both languages", got)
	}
}

func TestItemEntriesNameByTranslationKey(t *testing.T) {
	shown := map[string]any{"value": "공학의 정수"}
	e := localItemName(protocol.ItemEntry{Name: "skygear:research_essence_engineering", ComponentBased: true,
		Data: map[string]any{"components": map[string]any{"minecraft:display_name": shown}}})
	got := e.Data["components"].(map[string]any)["minecraft:display_name"].(map[string]any)["value"]
	if got != "item.skygear:research_essence_engineering.name" {
		t.Errorf("display name %v", got)
	}
	if shown["value"] != "공학의 정수" {
		t.Error("changed the map it was given")
	}
	if block := localItemName(protocol.ItemEntry{Name: "skygear:sieve", Data: map[string]any{}}); len(block.Data) != 0 {
		t.Errorf("a block item got %v", block.Data)
	}
}
