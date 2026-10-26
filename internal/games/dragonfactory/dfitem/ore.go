package dfitem

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/blockmodel"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// Ores and their processing follow ATM10: To the Sky
// (docs/design/16-ore-balance.md §3·§4): four chunks of a metal make its ore
// block; a pickaxe mines it into one raw ore and a furnace smelts it into one
// ingot; the hammer breaks it into two dusts; the crusher into three raw ores
// and two more at half a chance each; the ore washer washes a raw ore into
// two dusts. A raw ore and a dust each smelt into an ingot. Iron, gold and
// copper use the vanilla ores and raw ores; tin, lead, silver and nickel
// have their own.
const (
	// HammerDust are the dusts the hammer breaks an ore block into.
	HammerDust = 2
	// CrusherRaw are the raw ores the crusher always makes of an ore block,
	// crusherBonus the more it makes at crusherBonusChance percent each.
	CrusherRaw         = 3
	CrusherBonus       = 2
	CrusherBonusChance = 50
	// washerDust are the dusts the ore washer makes of a raw ore.
	washerDust = 2
)

// NewOreMetals have an ore block and a raw ore of the game's own.
var NewOreMetals = []string{MetalTin, MetalLead, MetalSilver, MetalNickel}

var oreBlockHashes = func() map[string]uint64 {
	m := map[string]uint64{}
	for _, metal := range NewOreMetals {
		m[metal] = block.NextHash()
	}
	return m
}()

// OreBlock is the ore block of a new metal.
type OreBlock struct{ Metal string }

func (o OreBlock) EncodeItem() (string, int16) { return "dragonfactory:" + o.Metal + "_ore", 0 }

func (o OreBlock) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:" + o.Metal + "_ore", nil
}

func (o OreBlock) Hash() (uint64, uint64) { return oreBlockHashes[o.Metal], 0 }

func (OreBlock) Model() world.BlockModel { return model.Solid{} }

func (OreBlock) Category() category.Category { return category.Nature() }

func (o OreBlock) Name() string {
	return metalName(o.Metal) + i18n.T(i18n.Default, "name.OreBlock.suffix")
}

func (o OreBlock) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.SolidMaterial("dragonfactory_" + o.Metal + "_ore")}}
}

// BreakInfo mines like an iron ore: a stone pickaxe or better gives one raw
// ore (the hammer is handled by the game, handler.go).
func (o OreBlock) BreakInfo() block.BreakInfo {
	harvestable := func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}
	return block.BreakInfo{
		Hardness: 3, BlastResistance: 3,
		Harvestable: harvestable,
		Effective:   func(t item.Tool) bool { return t.ToolType() == item.TypePickaxe },
		Drops: func(t item.Tool, _ []item.Enchantment) []item.Stack {
			if !harvestable(t) {
				return nil
			}
			return []item.Stack{item.NewStack(RawOre{Metal: o.Metal}, 1)}
		},
	}
}

// SmeltInfo implements item.Smeltable.
func (o OreBlock) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(Ingot{Metal: o.Metal}, 1), Experience: 0.7, Ores: true}
}

// RawOre is the raw ore of a new metal.
type RawOre struct{ Metal string }

func (r RawOre) EncodeItem() (string, int16) { return "dragonfactory:raw_" + r.Metal, 0 }

func (r RawOre) Name() string {
	return fmt.Sprintf(i18n.T(i18n.Default, "name.RawOre"), metalName(r.Metal))
}

func (RawOre) Category() category.Category { return category.Items() }

// SmeltInfo implements item.Smeltable.
func (r RawOre) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(Ingot{Metal: r.Metal}, 1), Experience: 0.7, Ores: true}
}

// RawOf is the raw ore of a metal.
func RawOf(metal string) world.Item {
	switch metal {
	case OreIron:
		return item.RawIron{}
	case OreGold:
		return item.RawGold{}
	case OreCopper:
		return item.RawCopper{}
	}
	return RawOre{Metal: metal}
}

// oreBlockMetals and rawMetals are the metals of the ore blocks and raw ores,
// by itemKey.
var oreBlockMetals, rawMetals = func() (blocks, raws map[string]string) {
	blocks, raws = map[string]string{}, map[string]string{}
	for _, k := range OreKinds {
		if key := naming.ItemKey(k.Block); blocks[key] == "" {
			blocks[key] = k.Ore
		}
		if key := naming.ItemKey(RawOf(k.Ore)); raws[key] == "" {
			raws[key] = k.Ore
		}
	}
	return blocks, raws
}()

// OreBlockMetal is the metal of an ore block.
func OreBlockMetal(it world.Item) (string, bool) {
	metal, ok := oreBlockMetals[naming.ItemKey(it)]
	return metal, ok
}

// rawMetal is the metal of a raw ore.
func rawMetal(it world.Item) (string, bool) {
	metal, ok := rawMetals[naming.ItemKey(it)]
	return metal, ok
}

// OreBlocks are the ore blocks of every metal.
func OreBlocks() []world.Item {
	var out []world.Item
	for _, k := range OreKinds {
		out = append(out, k.Block)
	}
	return out
}

// HammerDrop is what the hammer breaks b into: an ore block into its dusts,
// the stones one step down (crushRecipes).
func HammerDrop(b world.Block) (item.Stack, bool) {
	it, ok := b.(world.Item)
	if !ok {
		return item.Stack{}, false
	}
	if metal, ok := OreBlockMetal(it); ok {
		return item.NewStack(MetalDust{Metal: metal}, HammerDust), true
	}
	if out, ok := Crushed(b); ok {
		return item.NewStack(out, 1), true
	}
	return item.Stack{}, false
}

// WasherOutput is what the ore washer makes of a raw ore.
func WasherOutput(in item.Stack) (item.Stack, bool) {
	metal, ok := rawMetal(in.Item())
	if !ok {
		return item.Stack{}, false
	}
	return item.NewStack(MetalDust{Metal: metal}, washerDust), true
}
