package core

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
)

// oldPlain is the implementation Plain replaced.
func oldPlain(st item.Stack) bool { return st.Comparable(item.NewStack(st.Item(), 1)) }

func TestPlainTable(t *testing.T) {
	base := item.NewStack(item.Diamond{}, 3)
	ench := item.NewStack(item.Diamond{}, 1).WithEnchantments(item.NewEnchantment(enchantment.Unbreaking, 1))
	cases := map[string]item.Stack{
		"plain":   base,
		"named":   base.WithCustomName("x"),
		"lored":   base.WithLore("a"),
		"ench":    ench,
		"timed":   base.WithValue("expiry", int64(5)),
		"anvil":   base.WithAnvilCost(2),
		"damaged": item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1).Damage(5),
		"tool":    item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1),
	}
	for name, st := range cases {
		if got, want := Plain(st), oldPlain(st); got != want {
			t.Errorf("%s: Plain=%v old=%v", name, got, want)
		}
	}
	if Plain(cases["named"]) || !Plain(base) {
		t.Fatal("sanity")
	}
}
