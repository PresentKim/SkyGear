package dfitem

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// Hammer crushes blocks by hand: a block it mines drops it one step down
// (stone → cobblestone → gravel → sand), an ore block two dusts (ore.go).
// The stone hammer comes first; the iron hammer (the zero value, the first
// hammer of the game) also breaks the same crushable blocks connected to the
// one it mines while sneaking (vein.go). The stone hammer lasts twice as
// long as a stone tool; the iron hammer does not wear (handler.go
// HandleItemDamage): the vein mining costs food. It mines these blocks at its own speeds, sent to
// clients (the minecraft:digger component, a Dragonfly patch) so they mine as
// fast as the server expects.
type Hammer struct{ Stone bool }

// hammerSpeeds are the mining speeds of the hammer per block. Stone is
// three times as hard as gravel, sand and dirt, so those get a third of
// the speed and every block takes about as long (playtest: soft blocks
// broke far too fast at the same speed); gravel, sand and dust a half more
// (playtest: still slow).
var hammerSpeeds = map[string]int{
	"minecraft:stone":       6,
	"minecraft:cobblestone": 6,
	"minecraft:gravel":      3,
	"minecraft:sand":        3,
	"dragonfactory:dust":    3,
	"minecraft:dirt":        2,
	// the stones of the deep, the Nether and the End by their hardness
	"minecraft:deepslate":              12,
	"minecraft:cobbled_deepslate":      14,
	"minecraft:netherrack":             2,
	"minecraft:blackstone":             6,
	"minecraft:end_stone":              12,
	"minecraft:red_sand":               3,
	"dragonfactory:crushed_netherrack": 2,
	// ore blocks (into dust, ore.go)
	"minecraft:iron_ore":       12,
	"minecraft:gold_ore":       12,
	"minecraft:copper_ore":     12,
	"dragonfactory:tin_ore":    12,
	"dragonfactory:lead_ore":   12,
	"dragonfactory:silver_ore": 12,
	"dragonfactory:nickel_ore": 12,
}

// tier is the tool tier of the hammer's material.
func (h Hammer) tier() item.ToolTier {
	if h.Stone {
		return item.ToolTierStone
	}
	return item.ToolTierIron
}

func (h Hammer) EncodeItem() (string, int16) {
	if h.Stone {
		return "dragonfactory:stone_hammer", 0
	}
	return "dragonfactory:hammer", 0
}

func (h Hammer) Name() string {
	if h.Stone {
		return i18n.T(i18n.Default, "name.Hammer.stone")
	}
	return i18n.T(i18n.Default, "name.Hammer")
}

func (Hammer) Category() category.Category { return category.Equipment() }

func (Hammer) MaxCount() int { return 1 }

func (Hammer) HandEquipped() bool { return true }

func (h Hammer) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{MaxDurability: h.durabilityTimes() * h.tier().Durability, BrokenItem: func() item.Stack { return item.Stack{} }}
}

// durabilityTimes is how many times the durability of a tool of its material
// the hammer has.
func (h Hammer) durabilityTimes() int {
	if h.Stone {
		return 2
	}
	return 4
}

// ToolType, HarvestLevel and BaseMiningEfficiency make the hammer a pickaxe
// of its material for the server's break times and drops.
func (Hammer) ToolType() item.ToolType { return item.TypePickaxe }

func (h Hammer) HarvestLevel() int { return h.tier().HarvestLevel }

func (Hammer) BaseMiningEfficiency(b world.Block) float64 {
	if it, ok := b.(world.Item); ok {
		name, _ := it.EncodeItem()
		if v, ok := hammerSpeeds[name]; ok {
			return float64(v)
		}
	}
	return 1
}

// DestroySpeeds implements item.Digger.
func (Hammer) DestroySpeeds() map[string]int { return hammerSpeeds }

func IsHammer(st item.Stack) bool {
	_, ok := st.Item().(Hammer)
	return ok
}

// IsIronHammer reports whether st is the iron hammer (vein mining).
func IsIronHammer(st item.Stack) bool {
	h, ok := st.Item().(Hammer)
	return ok && !h.Stone
}

// Crushed returns what b turns into when crushed.
func Crushed(b world.Block) (world.Item, bool) {
	it, ok := b.(world.Item)
	if !ok {
		return nil, false
	}
	out, ok := CrushRecipes[naming.ItemKey(it)]
	return out, ok
}
