// Package tinker is the gear of the game, after Tinkers' Construct: tools
// made at the tinker table (table.go) of a head and a handle material, which
// keep their modifiers and experience in their stack, level up as they are
// used and get more modifier slots with every level; vein mining with the
// hammer (vein.go) and picking the right tool for a block (autotool.go).
package tinker

import (
	"fmt"
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/slotdeco"
	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Tool kinds.
const (
	ToolSword   = "sword"
	ToolPickaxe = "pickaxe"
	ToolAxe     = "axe"
	ToolShovel  = "shovel"
)

// ToolKinds are every kind of tool, registered so old ones load; the tools
// are the island's and do not fight (the second dimension's relics do), so
// MadeKinds, the ones the table makes, have no sword.
var (
	ToolKinds = []string{ToolSword, ToolPickaxe, ToolAxe, ToolShovel}
	MadeKinds = []string{ToolPickaxe, ToolAxe, ToolShovel}
)

// ToolHeadCost is the ingots a head of a kind takes.
var ToolHeadCost = map[string]int{ToolSword: 2, ToolPickaxe: 3, ToolAxe: 3, ToolShovel: 1}

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
}

var HeadMaterials = []HeadMaterial{
	{"flint", item.Flint{}, "tinker_table", 2, 110, 4},
	{"iron", item.IronIngot{}, "tinker_table", 3, 250, 6},
	{"bronze", sgitem.Ingot{Metal: sgitem.MetalBronze}, "alloy_smelter", 3, 300, 6},
	{"electrum", sgitem.Ingot{Metal: sgitem.MetalElectrum}, "alloy_smelter", 2, 70, 13},
	{"invar", sgitem.Ingot{Metal: sgitem.MetalInvar}, "alloy_smelter", 3, 480, 6},
	{"steel", sgitem.Ingot{Metal: sgitem.MetalSteel}, "alloy_smelter", 3, 700, 7.5},
	{"diamond", item.Diamond{}, "diamond_tools", 4, 1561, 8},
}

// HandleMaterial is what a tool's handle is made of.
type HandleMaterial struct {
	ID   string
	Item world.Item
	// DurMul and SpeedMul scale the head's durability and mining speed.
	DurMul, SpeedMul float64
}

var HandleMaterials = []HandleMaterial{
	{"wood", item.Stick{}, 1, 1},
	{"bone", item.Bone{}, 0.85, 1.12},
	{"blaze", item.BlazeRod{}, 1.4, 0.92},
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
	return fmt.Sprintf("skygear:tool_%s_%s_%s", t.Kind, t.Head, t.Handle), 0
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

// UseOnBlock lets an axe strip logs (and any other strippable block) the
// way a Dragonfly axe does, durability included; other kinds do nothing.
func (t Tool) UseOnBlock(pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	if t.Kind != ToolAxe {
		return false
	}
	return item.Axe{}.UseOnBlock(pos, face, clickPos, tx, user, ctx)
}

// StripsBark makes an axe shave bark fibers off the logs it strips
// (handwork.StripFiber).
func (t Tool) StripsBark() bool { return t.Kind == ToolAxe }

// AttackDamage is a tool's hit: no more than a hand's, the tools do not fight.
func (t Tool) AttackDamage() float64 { return 1 }

// toolTags are the block tags a kind of tool mines faster, as the vanilla
// tool of its type: the minecraft:is_<tool>_item_destructible tag, which our
// blocks get from their BreakInfo too (a Dragonfly patch), and the older
// material tags of vanilla blocks.
var toolTags = map[string][]string{
	ToolPickaxe: {"minecraft:is_pickaxe_item_destructible", "stone", "metal", "rail"},
	ToolAxe:     {"minecraft:is_axe_item_destructible", "wood", "pumpkin"},
	ToolShovel:  {"minecraft:is_shovel_item_destructible", "dirt", "sand", "gravel", "grass", "snow"},
	ToolSword:   {"minecraft:is_sword_item_destructible"},
}

// pickTierTags are the tags of the blocks a pickaxe drops nothing from
// below a harvest level, so it gets no speed on them (block.BreakDuration).
var pickTierTags = []struct {
	tag   string
	level int
}{
	{"stone_pick_diggable", item.ToolTierStone.HarvestLevel},
	{"iron_pick_diggable", item.ToolTierIron.HarvestLevel},
	{"diamond_pick_diggable", item.ToolTierDiamond.HarvestLevel},
}

// DestroySpeedsByTag implements item.TagDigger: the tool's speed on every
// block, vanilla or ours, that carries a tag of its kind, as the server
// times it, so the client mines them as fast as the server expects.
func (t Tool) DestroySpeedsByTag() []item.TagSpeed {
	tags := toolTags[t.Kind]
	if len(tags) == 0 {
		return nil
	}
	query := "query.any_tag('" + strings.Join(tags, "', '") + "')"
	if t.Kind == ToolPickaxe {
		for _, p := range pickTierTags {
			if t.HarvestLevel() < p.level {
				query += " && !query.any_tag('" + p.tag + "')"
			}
		}
	}
	return []item.TagSpeed{{Tags: query, Speed: int(math.Round(t.speed()))}}
}

// Modifiers of a tool, by id.
type ToolModifier struct {
	ID string
	// Item and cost are what one level takes.
	Item world.Item
	Cost int
	Max  int
}

// ToolModifiers are the modifiers a tool takes. The fighting ones (sharpness,
// fire, looting, vampiric) are gone: ToolMods drops them from old tools, which
// frees their slots.
var ToolModifiers = []ToolModifier{
	{"haste", item.RedstoneWire{}, 4, 5},
	{"reinforced", sgitem.Ingot{Metal: sgitem.MetalSteel}, 1, 5},
}

// isToolModifier reports whether id is a modifier tools still take.
func isToolModifier(id string) bool {
	for _, m := range ToolModifiers {
		if m.ID == id {
			return true
		}
	}
	return false
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
		if _, err := fmt.Sscanf(lv, "%d", &n); found && err == nil && n > 0 && isToolModifier(id) {
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

// The tools register as items: every kind with every head and handle.
func init() {
	for _, k := range ToolKinds {
		for _, h := range HeadMaterials {
			for _, hd := range HandleMaterials {
				world.RegisterItem(Tool{Kind: k, Head: h.ID, Handle: hd.ID})
			}
		}
	}
}
