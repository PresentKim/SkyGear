package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

// TestGameItemsHaveKoreanNames: every vanilla item the game names in its
// texts has a Korean name.
func TestGameItemsHaveKoreanNames(t *testing.T) {
	var items []world.Item
	for _, l := range Levels {
		for _, r := range l.Requirements {
			items = append(items, r.Item)
		}
	}
	for _, b := range BuybackItems {
		items = append(items, b.Item)
	}
	for _, u := range urgentItems {
		items = append(items, u.Item)
	}
	for _, r := range coreRecipes {
		items = append(items, r.Output.(world.Item))
	}
	for _, drops := range sieveRecipes {
		for _, d := range drops {
			items = append(items, d.Item.Item())
		}
	}
	for _, out := range crushRecipes {
		items = append(items, out)
	}
	for _, it := range ShopItems {
		items = append(items, it.Stack().Item())
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
	for name := range koreanNames {
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
