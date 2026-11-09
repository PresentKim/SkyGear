package skygear

import (
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

// TestTinkerDiggerSpeeds: a tinker tool tells the client its speed on the
// vanilla and our blocks its kind mines, and only those it harvests.
func TestTinkerDiggerSpeeds(t *testing.T) {
	axe := tinker.Tool{Kind: tinker.ToolAxe, Head: "flint", Handle: "wood"}.DestroySpeeds()
	pick := tinker.Tool{Kind: tinker.ToolPickaxe, Head: "flint", Handle: "wood"}.DestroySpeeds()
	shovel := tinker.Tool{Kind: tinker.ToolShovel, Head: "flint", Handle: "wood"}.DestroySpeeds()
	for _, name := range []string{"minecraft:oak_log", "minecraft:cherry_log", "minecraft:stripped_spruce_log", "minecraft:birch_planks", "skygear:drawer", "skygear:research_table"} {
		if axe[name] == 0 {
			t.Errorf("a tinker axe has no speed on %s", name)
		}
	}
	for _, name := range []string{"minecraft:stone", "minecraft:cobblestone", "minecraft:iron_ore", "skygear:barrel_stone"} {
		if pick[name] == 0 {
			t.Errorf("a tinker pickaxe has no speed on %s", name)
		}
	}
	if pick["minecraft:obsidian"] != 0 {
		t.Error("a flint pickaxe claims obsidian, which it does not harvest")
	}
	for _, name := range []string{"minecraft:dirt", "minecraft:sand", "skygear:dust"} {
		if shovel[name] == 0 {
			t.Errorf("a tinker shovel has no speed on %s", name)
		}
	}
	if axe["minecraft:stone"] != 0 || pick["minecraft:oak_log"] != 0 {
		t.Error("a tinker tool claims another kind's blocks")
	}
}
