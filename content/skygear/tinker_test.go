package skygear

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

func TestToolStats(t *testing.T) {
	sword := tinker.Tool{Kind: tinker.ToolSword, Head: "steel", Handle: "bone"}
	if sword.AttackDamage() != 4+3+0.5 {
		t.Errorf("sword attack %v", sword.AttackDamage())
	}
	if sword.MaxDurability() != int(700*0.85) {
		t.Errorf("sword durability %d", sword.MaxDurability())
	}
	pick := tinker.Tool{Kind: tinker.ToolPickaxe, Head: "diamond", Handle: "wood"}
	if pick.HarvestLevel() != 4 || pick.ToolType() != item.TypePickaxe || pick.DestroySpeeds()["minecraft:obsidian"] != 8 {
		t.Errorf("diamond pickaxe: level %d speeds %v", pick.HarvestLevel(), pick.DestroySpeeds()["minecraft:obsidian"])
	}
	// levels follow the square root of the xp
	for xp, want := range map[int]int{0: 0, 29: 0, 30: 1, 120: 2, 270: 3, 100000: tinker.ToolMaxLevel} {
		if got := tinker.ToolLevel(xp); got != want {
			t.Errorf("level at %d xp = %d, want %d", xp, got, want)
		}
	}
	if tinker.ToolSlots(0) != tinker.ToolBaseSlots || tinker.ToolSlots(4) != tinker.ToolBaseSlots+4 {
		t.Error("slots")
	}
}

// TestToolStackValues: modifiers and xp live in the stack and survive its NBT.
func TestToolStackValues(t *testing.T) {
	st := item.NewStack(tinker.Tool{Kind: tinker.ToolAxe, Head: "iron", Handle: "wood"}, 1)
	st = tinker.WithToolMods(st, map[string]int{"sharp": 2, "haste": 1})
	st, up := tinker.WithToolXP(st, 35)
	if !up || tinker.ToolLevel(tinker.ToolXP(st)) != 1 {
		t.Fatalf("no level up at 35 xp: %v %d", up, tinker.ToolXP(st))
	}
	back := item.ReadNBT(item.WriteNBT(st, true), nil)
	if tinker.ToolXP(back) != 35 || tinker.ToolMods(back)["sharp"] != 2 || tinker.ToolMods(back)["haste"] != 1 {
		t.Errorf("after NBT: xp %d mods %v", tinker.ToolXP(back), tinker.ToolMods(back))
	}
	axe := tinker.Tool{Kind: tinker.ToolAxe, Head: "iron", Handle: "wood"}
	if d, ok := tinker.TinkerDamage(back); !ok || d <= axe.AttackDamage()+2 {
		t.Errorf("damage %v", d)
	}
	if back.CustomName() == "" || len(back.Lore()) == 0 {
		t.Error("the tool has no name or lore")
	}
}

// TestTinkerTable: making a tool takes the head metal and the handle, a modifier
// takes its material and a slot, and the limits hold.
func TestTinkerTable(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		inv := p.Inventory()
		_, _ = inv.AddItem(item.NewStack(item.IronIngot{}, 3))
		_, _ = inv.AddItem(item.NewStack(item.Stick{}, 1))
		iron, _ := tinker.HeadOf("iron")
		wood, _ := tinker.HandleOf("wood")
		if !tinker.MakeTool(s.Session, p, tx, machinePos, tinker.ToolPickaxe, iron, wood) {
			t.Fatal("could not make a pickaxe from 3 iron and a stick")
		}
		if core.CountPlain(inv, item.IronIngot{}) != 0 || core.CountPlain(inv, item.Stick{}) != 0 {
			t.Error("the materials were not taken")
		}
		if tinker.MakeTool(s.Session, p, tx, machinePos, tinker.ToolPickaxe, iron, wood) {
			t.Error("made a tool from nothing")
		}
		var tool item.Stack
		for _, st := range inv.Slots() {
			if _, ok := st.Item().(tinker.Tool); ok {
				tool = st
			}
		}
		if tool.Empty() {
			t.Fatal("no tool in the inventory")
		}
		p.SetHeldItems(tool, item.Stack{})
		sharp := tinker.ToolModifiers[slices.IndexFunc(tinker.ToolModifiers, func(m tinker.ToolModifier) bool { return m.ID == "sharp" })]
		_, _ = inv.AddItem(item.NewStack(item.NetherQuartz{}, 2*tinker.ToolSlots(0)+2))
		for range tinker.ToolSlots(0) {
			if !tinker.ApplyModifier(s.Session, p, tx, machinePos, sharp) {
				t.Fatal("could not apply sharp")
			}
		}
		if tinker.ApplyModifier(s.Session, p, tx, machinePos, sharp) {
			t.Error("applied a modifier without a free slot")
		}
		held, _ := tinker.ToolOf(p)
		if tinker.ToolMods(held)["sharp"] != tinker.ToolSlots(0) {
			t.Errorf("mods %v", tinker.ToolMods(held))
		}
		if !progress.Advanced(s.Session, advancement.ToolMod) {
			t.Error("the modifier advancement is not done")
		}
		// xp from use raises the level and opens a slot
		tinker.GainXP(s.Session, p, 40)
		if !tinker.ApplyModifier(s.Session, p, tx, machinePos, sharp) {
			t.Error("the level up opened no slot")
		}
	})
}
