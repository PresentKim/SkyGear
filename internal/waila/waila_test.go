package waila

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
)

// TestFormat: the title keeps the pack's fixed layout: the prefix, a
// 10-character tool field, then the text with the anchor token.
func TestFormat(t *testing.T) {
	title, sub := Format(Info{
		Item: block.Stone{}, Name: "돌", Lines: []string{"진행 40%"}, Source: "Minecraft",
		Tools: []ToolIcon{{PickaxeWood, MarkRight}},
	})
	if !strings.HasPrefix(title, "_r4ui:A:aa,a;zz,z:") {
		t.Fatalf("title %q", title)
	}
	if len("_r4ui:A:aa,a;zz,z:") != 18 {
		t.Fatal("header must be 18 characters (the pack cuts the first 18)")
	}
	for _, want := range []string{"돌", "\n§7진행 40%", "\n§9§oMinecraft", anchor} {
		if !strings.Contains(title, want) {
			t.Errorf("title %q misses %q", title, want)
		}
	}
	if sub == "0" || sub == "" {
		t.Errorf("subtitle %q, want an item icon", sub)
	}
	if _, sub := Format(Info{Name: "물"}); sub != "0" {
		t.Errorf("no item: subtitle %q, want 0", sub)
	}
}

// TestToolsFor: stone wants a wooden pickaxe; the mark follows the held item.
func TestToolsFor(t *testing.T) {
	got := ToolsFor(block.Stone{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1))
	if len(got) == 0 || got[0] != (ToolIcon{PickaxeWood, MarkRight}) {
		t.Fatalf("stone with an iron pickaxe: %v", got)
	}
	if got := ToolsFor(block.Stone{}, item.Stack{}); len(got) == 0 || got[0].Mark != MarkWrong {
		t.Fatalf("stone by hand: %v", got)
	}
	if got := ToolsFor(block.Dirt{}, item.Stack{}); len(got) == 0 || got[0] != (ToolIcon{Shovel, MarkNone}) {
		t.Fatalf("dirt by hand: %v", got)
	}
	if got := ToolsFor(block.DiamondOre{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)); len(got) == 0 || got[0] != (ToolIcon{PickaxeIron, MarkWrong}) {
		t.Fatalf("diamond ore with a stone pickaxe: %v", got)
	}
}
