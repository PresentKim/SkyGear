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
	for _, c := range coreTypes {
		for _, tier := range c.Tiers {
			for _, e := range tier.Add {
				items = append(items, e.Block.(world.Item))
			}
		}
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
