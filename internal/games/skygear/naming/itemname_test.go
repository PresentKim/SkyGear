package naming

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
)

func TestItemNameKey(t *testing.T) {
	for _, c := range []struct {
		st   item.Stack
		want string
	}{
		{item.NewStack(item.Diamond{}, 1), "item.diamond.name"},
		{item.NewStack(block.Stone{}, 1), "tile.stone.stone.name"},
		{item.NewStack(block.Dirt{}, 1), "tile.dirt.name"},
		{item.NewStack(block.Planks{Wood: block.OakWood()}, 1), "tile.planks.oak.name"},
		{item.NewStack(item.Diamond{}, 1).WithCustomName("x"), ""},
	} {
		if got := ItemNameKey(c.st); got != c.want {
			t.Errorf("ItemNameKey(%T) = %q, want %q", c.st.Item(), got, c.want)
		}
	}
	if got := i18n.Plain(UIItemName("ko", item.NewStack(item.Diamond{}, 1))); got != ItemName("ko", item.NewStack(item.Diamond{}, 1)) {
		t.Errorf("UIItemName falls back to %q", got)
	}
}
