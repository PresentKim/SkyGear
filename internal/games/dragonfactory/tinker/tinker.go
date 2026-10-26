// Package tinker defines the gear crafted at the tinker table: tools made of
// a head and a handle material, their levels, XP and modifiers.
package tinker

import (
	"fmt"
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
)

// Tool kinds.
const (
	ToolSword   = "sword"
	ToolPickaxe = "pickaxe"
	ToolAxe     = "axe"
	ToolShovel  = "shovel"
)

var ToolKinds = []string{ToolSword, ToolPickaxe, ToolAxe, ToolShovel}

// ToolHeadCost is the ingots a head of a kind takes.
var ToolHeadCost = map[string]int{ToolSword: 2, ToolPickaxe: 3, ToolAxe: 3, ToolShovel: 1}

// ToolAttackBonus is what a kind adds to the attack of the head.
var ToolAttackBonus = map[string]float64{ToolSword: 3, ToolPickaxe: 1, ToolAxe: 2.5, ToolShovel: 0.5}

// HeadMaterial is a metal a tool head is made of.
type HeadMaterial struct {
	ID string
	// Item is what a head takes (the ingot).
	Item world.Item
	// Research opens the material at the tinker table.
	Research string
	harvest  int
	Dur      int
	Eff      float64
	Attack   float64
}

var HeadMaterials = []HeadMaterial{
	{"flint", item.Flint{}, "tinker_table", 2, 110, 4, 2},
	{"iron", item.IronIngot{}, "tinker_table", 3, 250, 6, 3},
	{"bronze", dfitem.Ingot{Metal: dfitem.MetalBronze}, "alloy_smelter", 3, 300, 6, 3},
	{"electrum", dfitem.Ingot{Metal: dfitem.MetalElectrum}, "alloy_smelter", 2, 70, 13, 2},
	{"invar", dfitem.Ingot{Metal: dfitem.MetalInvar}, "alloy_smelter", 3, 480, 6, 3.5},
	{"steel", dfitem.Ingot{Metal: dfitem.MetalSteel}, "alloy_smelter", 3, 700, 7.5, 4},
	{"diamond", item.Diamond{}, "diamond_tools", 4, 1561, 8, 4.5},
}

// HandleMaterial is what a tool's handle is made of.
type HandleMaterial struct {
	ID   string
	Item world.Item
	// DurMul and SpeedMul scale the head's durability and mining speed.
	DurMul, SpeedMul float64
	// Attack is added to the attack.
	Attack float64
}

var HandleMaterials = []HandleMaterial{
	{"wood", item.Stick{}, 1, 1, 0},
	{"bone", item.Bone{}, 0.85, 1.12, 0.5},
	{"blaze", item.BlazeRod{}, 1.4, 0.92, 0},
}

func HeadOf(id string) (HeadMaterial, bool) {
	for _, h := range HeadMaterials {
		if h.ID == id {
			return h, true
		}
	}
	return HeadMaterial{}, false
}

func HandleOf(id string) (HandleMaterial, bool) {
	for _, h := range HandleMaterials {
		if h.ID == id {
			return h, true
		}
	}
	return HandleMaterial{}, false
}

// Tool is a tool of a kind with a head and a handle.
type Tool struct{ Kind, Head, Handle string }

func (t Tool) EncodeItem() (string, int16) {
	return fmt.Sprintf("dragonfactory:tool_%s_%s_%s", t.Kind, t.Head, t.Handle), 0
}

// Name is the metal and the kind of tool ("강철 곡괭이").
func (t Tool) Name() string {
	return i18n.T(i18n.Default, "tmat."+t.Head) + " " + i18n.T(i18n.Default, "tool."+t.Kind)
}

func (Tool) Category() category.Category { return category.Equipment() }

func (Tool) MaxCount() int { return 1 }

func (Tool) HandEquipped() bool { return true }

func (t Tool) head() HeadMaterial { h, _ := HeadOf(t.Head); return h }

func (t Tool) handle() HandleMaterial { h, _ := HandleOf(t.Handle); return h }

// MaxDurability is the durability of the tool.
func (t Tool) MaxDurability() int {
	return max(20, int(float64(t.head().Dur)*t.handle().DurMul))
}

func (t Tool) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability:    t.MaxDurability(),
		BrokenItem:       func() item.Stack { return item.Stack{} },
		AttackDurability: 2,
		BreakDurability:  1,
	}
}

func (t Tool) ToolType() item.ToolType {
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

func (t Tool) HarvestLevel() int { return t.head().harvest }

func (t Tool) speed() float64 { return t.head().Eff * t.handle().SpeedMul }

func (t Tool) BaseMiningEfficiency(world.Block) float64 { return t.speed() }

// AttackDamage is the damage of the bare tool; TinkerDamage adds what the stack
// has gained.
func (t Tool) AttackDamage() float64 {
	return t.head().Attack + ToolAttackBonus[t.Kind] + t.handle().Attack
}

// toolBlocks are the blocks the client is told a kind of tool mines faster.
var toolBlocks = map[string][]string{
	ToolPickaxe: {"stone", "cobblestone", "deepslate", "granite", "diorite", "andesite", "sandstone", "stone_bricks", "bricks", "netherrack", "obsidian", "iron_ore", "gold_ore", "copper_ore", "coal_ore", "diamond_ore", "lapis_ore", "redstone_ore", "emerald_ore", "iron_block", "gold_block", "furnace", "hopper", "stone_slab", "cobblestone_wall", "stone_stairs", "bone_block", "ice"},
	ToolAxe:     {"oak_log", "birch_log", "spruce_log", "jungle_log", "acacia_log", "dark_oak_log", "stripped_oak_log", "oak_planks", "birch_planks", "spruce_planks", "crafting_table", "chest", "barrel", "bookshelf", "pumpkin", "melon_block", "wooden_slab", "oak_fence"},
	ToolShovel:  {"dirt", "grass_block", "sand", "gravel", "clay", "farmland", "podzol", "soul_sand", "snow", "mycelium", "coarse_dirt"},
	ToolSword:   {"web", "bamboo"},
}

// DestroySpeeds implements item.Digger.
func (t Tool) DestroySpeeds() map[string]int {
	out := map[string]int{}
	for _, b := range toolBlocks[t.Kind] {
		out["minecraft:"+b] = int(math.Round(t.speed()))
	}
	return out
}

// Modifiers of a tool, by id.
type ToolModifier struct {
	ID string
	// Item and cost are what one level takes.
	Item world.Item
	Cost int
	Max  int
}

var ToolModifiers = []ToolModifier{
	{"sharp", item.NetherQuartz{}, 2, 5},
	{"haste", item.RedstoneWire{}, 4, 5},
	{"reinforced", dfitem.Ingot{Metal: dfitem.MetalSteel}, 1, 5},
	{"fiery", item.BlazePowder{}, 2, 3},
	{"looting", item.LapisLazuli{}, 4, 3},
	{"vampiric", item.SpiderEye{}, 2, 3},
}

const (
	ToolBaseSlots = 3
	ToolMaxLevel  = 10
	toolXPUnit    = 30
	valueMods     = "tinker_mods"
	valueXP       = "tinker_xp"
)

// ToolMods returns the modifiers (id to level) of a tool stack; they are kept as
// a string "id:level,id:level" (a gob-friendly stack value).
func ToolMods(st item.Stack) map[string]int {
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

// WithToolMods stores the modifiers on a stack.
func WithToolMods(st item.Stack, mods map[string]int) item.Stack {
	var parts []string
	for _, m := range ToolModifiers {
		if l := mods[m.ID]; l > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", m.ID, l))
		}
	}
	return st.WithValue(valueMods, strings.Join(parts, ","))
}

// ToolXP is the experience of a tool stack.
func ToolXP(st item.Stack) int {
	v, _ := st.Value(valueXP)
	n, _ := v.(int)
	return n
}

// ToolLevel is the level of a tool: it grows with the square root of its xp.
func ToolLevel(xp int) int { return min(ToolMaxLevel, int(math.Sqrt(float64(xp)/toolXPUnit))) }

// xpForLevel is the xp a level starts at.
func xpForLevel(l int) int { return l * l * toolXPUnit }

// ToolSlots is how many levels of modifiers a tool of a level can hold.
func ToolSlots(level int) int { return ToolBaseSlots + level }

func ModsUsed(mods map[string]int) int {
	n := 0
	for _, l := range mods {
		n += l
	}
	return n
}

// TinkerDamage is the damage of an attack with the stack: the bare tool, plus
// sharpness, plus a bit for the tool's level.
func TinkerDamage(st item.Stack) (float64, bool) {
	t, ok := st.Item().(Tool)
	if !ok {
		return 0, false
	}
	mods := ToolMods(st)
	return t.AttackDamage() + float64(mods["sharp"]) + 0.3*float64(ToolLevel(ToolXP(st))) + 1.5*float64(mods["fiery"]), true
}

// DescribeTool is the name and lore of a tool stack: its level and xp, its
// handle and what is on it.
func DescribeTool(st item.Stack) item.Stack {
	t, ok := st.Item().(Tool)
	if !ok {
		return st
	}
	xp := ToolXP(st)
	lvl := ToolLevel(xp)
	mods := ToolMods(st)
	lang := i18n.Default
	lore := []string{
		fmt.Sprintf(i18n.T(lang, "tinker.lore.level"), lvl, xp, xpForLevel(min(lvl+1, ToolMaxLevel))),
		fmt.Sprintf(i18n.T(lang, "tinker.lore.handle"), i18n.T(lang, "thandle."+t.Handle)),
		fmt.Sprintf(i18n.T(lang, "tinker.lore.slots"), ModsUsed(mods), ToolSlots(lvl)),
	}
	for _, m := range ToolModifiers {
		if l := mods[m.ID]; l > 0 {
			lore = append(lore, fmt.Sprintf("§b%s %s", i18n.T(lang, "tmod."+m.ID), strings.Repeat("I", l)))
		}
	}
	r := toolRarity(lvl)
	return st.WithCustomName(slotdeco.Decorate(r, 0, "§r"+RarityColour(r)+t.Name()+fmt.Sprintf(" §7Lv.%d", lvl))).WithLore(lore...)
}

// WithToolXP adds xp to a tool stack and returns it, with whether it leveled up.
func WithToolXP(st item.Stack, n int) (item.Stack, bool) {
	before := ToolLevel(ToolXP(st))
	st = st.WithValue(valueXP, ToolXP(st)+n)
	return DescribeTool(st), ToolLevel(ToolXP(st)) > before
}

// toolRarity is the rarity of a tool of the level: a tool that has been used
// a lot looks worth more.
func toolRarity(level int) slotdeco.Rarity {
	switch {
	case level >= 10:
		return slotdeco.Legendary
	case level >= 8:
		return slotdeco.Epic
	case level >= 5:
		return slotdeco.Rare
	case level >= 3:
		return slotdeco.Uncommon
	}
	return slotdeco.Common
}

// RarityColour is the colour code of a rarity's name ("§f" for Common).
func RarityColour(r slotdeco.Rarity) string {
	switch r {
	case slotdeco.Uncommon:
		return "§a"
	case slotdeco.Rare:
		return "§b"
	case slotdeco.Epic:
		return "§d"
	case slotdeco.Legendary:
		return "§6"
	}
	return "§f"
}
