package sgitem

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestOpensScreen(t *testing.T) {
	menu := item.NewStack(item.NetherStar{}, 1).WithValue(MenuItemKey, true)
	for _, c := range []struct {
		name       string
		it         item.Stack
		menu, open bool
	}{
		{"menu item", menu, true, true},
		{"dimension stone", item.NewStack(DimensionStone{}, 1), false, true},
		{"plain nether star", item.NewStack(item.NetherStar{}, 1), false, false},
		{"copier", item.NewStack(ConfigCopier{}, 1), false, false},
		{"empty hand", item.Stack{}, false, false},
	} {
		if got := IsMenuItem(c.it); got != c.menu {
			t.Errorf("%s: IsMenuItem = %t", c.name, got)
		}
		if got := OpensScreen(c.it); got != c.open {
			t.Errorf("%s: OpensScreen = %t", c.name, got)
		}
	}
}
