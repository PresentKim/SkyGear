package dragonfactory

import (
	"fmt"
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
)

// Tinkered tools (docs/design/08-storage-gear-magic.md §2), after Tinkers'
// Construct: a tool is a head of a metal on a handle, made at the tinker table;
// each tool keeps its modifiers and its experience in its stack, levels up as it
// is used and gets more modifier slots with every level.

// Tool kinds.
const (
	ToolSword   = "sword"
	ToolPickaxe = "pickaxe"
	ToolAxe     = "axe"
	ToolShovel  = "shovel"
)

var toolKinds = []string{ToolSword, ToolPickaxe, ToolAxe, ToolShovel}

// toolHeadCost is the ingots a head of a kind takes.
var toolHeadCost = map[string]int{ToolSword: 2, ToolPickaxe: 3, ToolAxe: 3, ToolShovel: 1}

// toolAttackBonus is what a kind adds to the attack of the head.
var toolAttackBonus = map[string]float64{ToolSword: 3, ToolPickaxe: 1, ToolAxe: 2.5, ToolShovel: 0.5}

// headMaterial is a metal a tool head is made of.
type headMaterial struct {
	id string
	// item is what a head takes (the ingot).
	item world.Item
	// research opens the material at the tinker table.
	research string
	harvest  int
	dur      int
	eff      float64
	attack   float64
}

var headMaterials = []headMaterial{
	{"flint", item.Flint{}, "tinker_table", 2, 110, 4, 2},
	{"iron", item.IronIngot{}, "tinker_table", 3, 250, 6, 3},
	{"bronze", Ingot{Metal: MetalBronze}, "alloy_smelter", 3, 300, 6, 3},
	{"electrum", Ingot{Metal: MetalElectrum}, "alloy_smelter", 2, 70, 13, 2},
	{"invar", Ingot{Metal: MetalInvar}, "alloy_smelter", 3, 480, 6, 3.5},
	{"steel", Ingot{Metal: MetalSteel}, "alloy_smelter", 3, 700, 7.5, 4},
	{"diamond", item.Diamond{}, "diamond_tools", 4, 1561, 8, 4.5},
}

// handleMaterial is what a tool's handle is made of.
type handleMaterial struct {
	id   string
	item world.Item
	// durMul and speedMul scale the head's durability and mining speed.
	durMul, speedMul float64
	// attack is added to the attack.
	attack float64
}

var handleMaterials = []handleMaterial{
	{"wood", item.Stick{}, 1, 1, 0},
	{"bone", item.Bone{}, 0.85, 1.12, 0.5},
	{"blaze", item.BlazeRod{}, 1.4, 0.92, 0},
}

func headOf(id string) (headMaterial, bool) {
	for _, h := range headMaterials {
		if h.id == id {
			return h, true
		}
	}
	return headMaterial{}, false
}

func handleOf(id string) (handleMaterial, bool) {
	for _, h := range handleMaterials {
		if h.id == id {
			return h, true
		}
	}
	return handleMaterial{}, false
}

// TinkerTool is a tool of a kind with a head and a handle.
type TinkerTool struct{ Kind, Head, Handle string }

func (t TinkerTool) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:tool_%s_%s_%s", t.Kind, t.Head, t.Handle), 0
}

// Name is the metal and the kind of tool ("강철 곡괭이").
func (t TinkerTool) Name() string {
	return i18n.T(i18n.Default, "tmat."+t.Head) + " " + i18n.T(i18n.Default, "tool."+t.Kind)
}
func (TinkerTool) Category() category.Category { return category.Equipment() }
func (TinkerTool) MaxCount() int               { return 1 }
func (TinkerTool) HandEquipped() bool          { return true }

func (t TinkerTool) head() headMaterial     { h, _ := headOf(t.Head); return h }
func (t TinkerTool) handle() handleMaterial { h, _ := handleOf(t.Handle); return h }

// MaxDurability is the durability of the tool.
func (t TinkerTool) MaxDurability() int {
	return max(20, int(float64(t.head().dur)*t.handle().durMul))
}

func (t TinkerTool) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability:    t.MaxDurability(),
		BrokenItem:       func() item.Stack { return item.Stack{} },
		AttackDurability: 2,
		BreakDurability:  1,
	}
}

func (t TinkerTool) ToolType() item.ToolType {
	switch t.Kind {
	case ToolPickaxe:
		return item.TypePickaxe
	case ToolAxe:
		return item.TypeAxe
	case ToolShovel:
		return item.TypeShovel
	}
	return item.TypeSword
}

func (t TinkerTool) HarvestLevel() int { return t.head().harvest }

func (t TinkerTool) speed() float64 { return t.head().eff * t.handle().speedMul }

func (t TinkerTool) BaseMiningEfficiency(world.Block) float64 { return t.speed() }

// AttackDamage is the damage of the bare tool; tinkerDamage adds what the stack
// has gained.
func (t TinkerTool) AttackDamage() float64 {
	return t.head().attack + toolAttackBonus[t.Kind] + t.handle().attack
}

// toolBlocks are the blocks the client is told a kind of tool mines faster.
var toolBlocks = map[string][]string{
	ToolPickaxe: {"stone", "cobblestone", "deepslate", "granite", "diorite", "andesite", "sandstone", "stone_bricks", "bricks", "netherrack", "obsidian", "iron_ore", "gold_ore", "copper_ore", "coal_ore", "diamond_ore", "lapis_ore", "redstone_ore", "emerald_ore", "iron_block", "gold_block", "furnace", "hopper", "stone_slab", "cobblestone_wall", "stone_stairs", "bone_block", "ice"},
	ToolAxe:     {"oak_log", "birch_log", "spruce_log", "jungle_log", "acacia_log", "dark_oak_log", "stripped_oak_log", "oak_planks", "birch_planks", "spruce_planks", "crafting_table", "chest", "barrel", "bookshelf", "pumpkin", "melon_block", "wooden_slab", "oak_fence"},
	ToolShovel:  {"dirt", "grass_block", "sand", "gravel", "clay", "farmland", "podzol", "soul_sand", "snow", "mycelium", "coarse_dirt"},
	ToolSword:   {"web", "bamboo"},
}

// DestroySpeeds implements item.Digger.
func (t TinkerTool) DestroySpeeds() map[string]int {
	out := map[string]int{}
	for _, b := range toolBlocks[t.Kind] {
		out["minecraft:"+b] = int(math.Round(t.speed()))
	}
	return out
}

// Modifiers of a tool, by id.
type toolModifier struct {
	id string
	// item and cost are what one level takes.
	item world.Item
	cost int
	max  int
}

var toolModifiers = []toolModifier{
	{"sharp", item.NetherQuartz{}, 2, 5},
	{"haste", item.RedstoneWire{}, 4, 5},
	{"reinforced", Ingot{Metal: MetalSteel}, 1, 5},
	{"fiery", item.BlazePowder{}, 2, 3},
	{"looting", item.LapisLazuli{}, 4, 3},
	{"vampiric", item.SpiderEye{}, 2, 3},
}

const (
	toolBaseSlots = 3
	toolMaxLevel  = 10
	toolXPUnit    = 30
	valueMods     = "tinker_mods"
	valueXP       = "tinker_xp"
)

// toolMods returns the modifiers (id to level) of a tool stack; they are kept as
// a string "id:level,id:level" (a gob-friendly stack value).
func toolMods(st item.Stack) map[string]int {
	out := map[string]int{}
	v, ok := st.Value(valueMods)
	str, _ := v.(string)
	if !ok || str == "" {
		return out
	}
	for _, part := range strings.Split(str, ",") {
		id, lv, found := strings.Cut(part, ":")
		var n int
		if _, err := fmt.Sscanf(lv, "%d", &n); found && err == nil && n > 0 {
			out[id] = n
		}
	}
	return out
}

// withToolMods stores the modifiers on a stack.
func withToolMods(st item.Stack, mods map[string]int) item.Stack {
	var parts []string
	for _, m := range toolModifiers {
		if l := mods[m.id]; l > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", m.id, l))
		}
	}
	return st.WithValue(valueMods, strings.Join(parts, ","))
}

// toolXP is the experience of a tool stack.
func toolXP(st item.Stack) int {
	v, _ := st.Value(valueXP)
	n, _ := v.(int)
	return n
}

// toolLevel is the level of a tool: it grows with the square root of its xp.
func toolLevel(xp int) int { return min(toolMaxLevel, int(math.Sqrt(float64(xp)/toolXPUnit))) }

// xpForLevel is the xp a level starts at.
func xpForLevel(l int) int { return l * l * toolXPUnit }

// toolSlots is how many levels of modifiers a tool of a level can hold.
func toolSlots(level int) int { return toolBaseSlots + level }

func modsUsed(mods map[string]int) int {
	n := 0
	for _, l := range mods {
		n += l
	}
	return n
}

// tinkerDamage is the damage of an attack with the stack: the bare tool, plus
// sharpness, plus a bit for the tool's level.
func tinkerDamage(st item.Stack) (float64, bool) {
	t, ok := st.Item().(TinkerTool)
	if !ok {
		return 0, false
	}
	mods := toolMods(st)
	return t.AttackDamage() + float64(mods["sharp"]) + 0.3*float64(toolLevel(toolXP(st))) + 1.5*float64(mods["fiery"]), true
}

// describeTool is the name and lore of a tool stack: its level and xp, its
// handle and what is on it.
func describeTool(st item.Stack) item.Stack {
	t, ok := st.Item().(TinkerTool)
	if !ok {
		return st
	}
	xp := toolXP(st)
	lvl := toolLevel(xp)
	mods := toolMods(st)
	lang := i18n.Default
	lore := []string{
		fmt.Sprintf(i18n.T(lang, "tinker.lore.level"), lvl, xp, xpForLevel(min(lvl+1, toolMaxLevel))),
		fmt.Sprintf(i18n.T(lang, "tinker.lore.handle"), i18n.T(lang, "thandle."+t.Handle)),
		fmt.Sprintf(i18n.T(lang, "tinker.lore.slots"), modsUsed(mods), toolSlots(lvl)),
	}
	for _, m := range toolModifiers {
		if l := mods[m.id]; l > 0 {
			lore = append(lore, fmt.Sprintf("§b%s %s", i18n.T(lang, "tmod."+m.id), strings.Repeat("I", l)))
		}
	}
	r := toolRarity(lvl)
	return st.WithCustomName(slotdeco.Decorate(r, 0, "§r"+rarityColour(r)+t.Name()+fmt.Sprintf(" §7Lv.%d", lvl))).WithLore(lore...)
}

// withToolXP adds xp to a tool stack and returns it, with whether it leveled up.
func withToolXP(st item.Stack, n int) (item.Stack, bool) {
	before := toolLevel(toolXP(st))
	st = st.WithValue(valueXP, toolXP(st)+n)
	return describeTool(st), toolLevel(toolXP(st)) > before
}

func init() {
	for _, k := range toolKinds {
		for _, h := range headMaterials {
			for _, hd := range handleMaterials {
				world.RegisterItem(TinkerTool{Kind: k, Head: h.id, Handle: hd.id})
			}
		}
	}
	_ = block.Air{}
}
