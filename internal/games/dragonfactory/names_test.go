package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
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
		if got := itemName(item.NewStack(it, 1)); "minecraft:"+got == name {
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
