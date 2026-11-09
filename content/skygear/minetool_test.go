package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// TestCustomBlocksMineFasterWithTheirTool: our blocks break faster with the
// tool of what they are made of, a Dragonfly tool or a tinker one, and no
// faster with another; the bare hand keeps the time the client is told.
func TestCustomBlocksMineFasterWithTheirTool(t *testing.T) {
	axe, pick, shovel := item.Axe{Tier: item.ToolTierStone}, item.Pickaxe{Tier: item.ToolTierStone}, item.Shovel{Tier: item.ToolTierStone}
	tAxe := tinker.Tool{Kind: tinker.ToolAxe, Head: "iron", Handle: "wood"}
	tPick := tinker.Tool{Kind: tinker.ToolPickaxe, Head: "iron", Handle: "wood"}
	tShovel := tinker.Tool{Kind: tinker.ToolShovel, Head: "iron", Handle: "wood"}
	for _, c := range []struct {
		name  string
		b     world.Block
		right []world.Item
		wrong world.Item
	}{
		{"drawer", storage.Drawer{}, []world.Item{axe, tAxe}, pick},
		{"research table", progress.ResearchTable{}, []world.Item{axe, tAxe}, pick},
		{"oak barrel", farm.Barrel{Wood: "oak"}, []world.Item{axe, tAxe}, pick},
		{"stone barrel", farm.Barrel{Wood: farm.StoneWood}, []world.Item{pick, tPick}, axe},
		{"wood hopper", hopper.NewFlowHopper("wood_hopper", cube.FaceDown), []world.Item{axe, tAxe}, pick},
		{"iron hopper", hopper.NewFlowHopper("iron_hopper_reverse", cube.FaceUp), []world.Item{pick, tPick}, axe},
		{"sieve", machine.Machine{Kind: machine.KindSieve}, []world.Item{axe, tAxe}, pick},
		{"crusher", machine.Machine{Kind: machine.KindCrusher}, []world.Item{pick, tPick}, axe},
		{"dust", sgitem.Dust{}, []world.Item{shovel, tShovel, sgitem.Hammer{}}, axe},
	} {
		hand := block.BreakDuration(c.b, item.Stack{}, block.BreakContext{})
		if d := block.BreakDuration(c.b, item.NewStack(c.wrong, 1), block.BreakContext{}); d != hand {
			t.Errorf("%s: %T takes %v, the hand %v", c.name, c.wrong, d, hand)
		}
		for _, it := range c.right {
			if d := block.BreakDuration(c.b, item.NewStack(it, 1), block.BreakContext{}); d >= hand {
				t.Errorf("%s: %#v takes %v, no faster than the hand's %v", c.name, it, d, hand)
			}
		}
	}
}

// TestTinkerDiggerTags: a tinker tool tells the client its speed by the
// tags of its kind's blocks, and leaves out the blocks a pickaxe of its
// level drops nothing from.
func TestTinkerDiggerTags(t *testing.T) {
	query := func(kind, head string) string {
		sp := tinker.Tool{Kind: kind, Head: head, Handle: "wood"}.DestroySpeedsByTag()
		if len(sp) != 1 || sp[0].Speed <= 0 {
			t.Fatalf("%s %s: speeds %v", head, kind, sp)
		}
		return sp[0].Tags
	}
	for kind, tag := range map[string]string{
		tinker.ToolAxe:     "'minecraft:is_axe_item_destructible'",
		tinker.ToolPickaxe: "'minecraft:is_pickaxe_item_destructible'",
		tinker.ToolShovel:  "'minecraft:is_shovel_item_destructible'",
	} {
		if q := query(kind, "flint"); !strings.HasPrefix(q, "query.any_tag(") || !strings.Contains(q, tag) {
			t.Errorf("%s: %s, want %s", kind, q, tag)
		}
	}
	flint, iron, diamond := query(tinker.ToolPickaxe, "flint"), query(tinker.ToolPickaxe, "iron"), query(tinker.ToolPickaxe, "diamond")
	if !strings.Contains(flint, "!query.any_tag('iron_pick_diggable')") || !strings.Contains(flint, "!query.any_tag('diamond_pick_diggable')") || strings.Contains(flint, "stone_pick_diggable") {
		t.Errorf("flint pickaxe: %s", flint)
	}
	if strings.Contains(iron, "iron_pick_diggable") || !strings.Contains(iron, "!query.any_tag('diamond_pick_diggable')") {
		t.Errorf("iron pickaxe: %s", iron)
	}
	if strings.Contains(diamond, "!") {
		t.Errorf("diamond pickaxe: %s", diamond)
	}
	if strings.Contains(query(tinker.ToolAxe, "flint"), "pick") {
		t.Error("an axe query names pickaxe tags")
	}
}
