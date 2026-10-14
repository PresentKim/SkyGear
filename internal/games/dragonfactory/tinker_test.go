package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestToolStats(t *testing.T) {
	sword := TinkerTool{Kind: ToolSword, Head: "steel", Handle: "bone"}
	if sword.AttackDamage() != 4+3+0.5 {
		t.Errorf("sword attack %v", sword.AttackDamage())
	}
	if sword.MaxDurability() != int(700*0.85) {
		t.Errorf("sword durability %d", sword.MaxDurability())
	}
	pick := TinkerTool{Kind: ToolPickaxe, Head: "diamond", Handle: "wood"}
	if pick.HarvestLevel() != 4 || pick.ToolType() != item.TypePickaxe || pick.DestroySpeeds()["minecraft:obsidian"] != 8 {
		t.Errorf("diamond pickaxe: level %d speeds %v", pick.HarvestLevel(), pick.DestroySpeeds()["minecraft:obsidian"])
	}
	// levels follow the square root of the xp
	for xp, want := range map[int]int{0: 0, 29: 0, 30: 1, 120: 2, 270: 3, 100000: toolMaxLevel} {
		if got := toolLevel(xp); got != want {
			t.Errorf("level at %d xp = %d, want %d", xp, got, want)
		}
	}
	if toolSlots(0) != toolBaseSlots || toolSlots(4) != toolBaseSlots+4 {
		t.Error("slots")
	}
}

// TestToolStackValues: modifiers and xp live in the stack and survive its NBT.
func TestToolStackValues(t *testing.T) {
	st := item.NewStack(TinkerTool{Kind: ToolAxe, Head: "iron", Handle: "wood"}, 1)
	st = withToolMods(st, map[string]int{"sharp": 2, "haste": 1})
	st, up := withToolXP(st, 35)
	if !up || toolLevel(toolXP(st)) != 1 {
		t.Fatalf("no level up at 35 xp: %v %d", up, toolXP(st))
	}
	back := item.ReadNBT(item.WriteNBT(st, true), nil)
	if toolXP(back) != 35 || toolMods(back)["sharp"] != 2 || toolMods(back)["haste"] != 1 {
		t.Errorf("after NBT: xp %d mods %v", toolXP(back), toolMods(back))
	}
	axe := TinkerTool{Kind: ToolAxe, Head: "iron", Handle: "wood"}
	if d, ok := tinkerDamage(back); !ok || d <= axe.AttackDamage()+2 {
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
		iron, _ := headOf("iron")
		wood, _ := handleOf("wood")
		if !s.makeTool(p, tx, machinePos, ToolPickaxe, iron, wood) {
			t.Fatal("could not make a pickaxe from 3 iron and a stick")
		}
		if countPlain(inv, item.IronIngot{}) != 0 || countPlain(inv, item.Stick{}) != 0 {
			t.Error("the materials were not taken")
		}
		if s.makeTool(p, tx, machinePos, ToolPickaxe, iron, wood) {
			t.Error("made a tool from nothing")
		}
		var tool item.Stack
		for _, st := range inv.Slots() {
			if _, ok := st.Item().(TinkerTool); ok {
				tool = st
			}
		}
		if tool.Empty() {
			t.Fatal("no tool in the inventory")
		}
		p.SetHeldItems(tool, item.Stack{})
		sharp, _ := modifierOf("sharp")
		_, _ = inv.AddItem(item.NewStack(item.NetherQuartz{}, 2*toolSlots(0)+2))
		for range toolSlots(0) {
			if !s.applyModifier(p, tx, machinePos, sharp) {
				t.Fatal("could not apply sharp")
			}
		}
		if s.applyModifier(p, tx, machinePos, sharp) {
			t.Error("applied a modifier without a free slot")
		}
		held, _ := toolOf(p)
		if toolMods(held)["sharp"] != toolSlots(0) {
			t.Errorf("mods %v", toolMods(held))
		}
		if !s.advanced(advToolMod) {
			t.Error("the modifier advancement is not done")
		}
		// xp from use raises the level and opens a slot
		s.gainToolXP(p, 40)
		if !s.applyModifier(p, tx, machinePos, sharp) {
			t.Error("the level up opened no slot")
		}
	})
}

// TestCharacter: experience makes levels and skill points, a point goes into a skill
// up to its limit, and the skills give health and damage.
func TestCharacter(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		r := s.rpgOf(p)
		if r.points() != 0 {
			t.Fatalf("%d points at the start", r.points())
		}
		s.gainXP(p, rpgXPFor(4))
		if rpgLevel(r.XP) != 4 || r.points() != 4 {
			t.Fatalf("level %d points %d at %d xp", rpgLevel(r.XP), r.points(), r.XP)
		}
		r.Skills["vitality"] = 2
		r.Skills["strength"] = 2
		s.applyRPG(p)
		if p.MaxHealth() != 24 {
			t.Errorf("max health %v with 2 vitality", p.MaxHealth())
		}
		if got := s.rpgDamageMul(p); got < 1.119 || got > 1.121 {
			t.Errorf("damage multiplier %v", got)
		}
		if r.points() != 0 {
			t.Errorf("%d points left", r.points())
		}
	})
}
