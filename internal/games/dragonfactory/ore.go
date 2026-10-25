package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
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
	// hammerDust are the dusts the hammer breaks an ore block into.
	hammerDust = 2
	// crusherRaw are the raw ores the crusher always makes of an ore block,
	// crusherBonus the more it makes at crusherBonusChance percent each.
	crusherRaw         = 3
	crusherBonus       = 2
	crusherBonusChance = 50
	// washerDust are the dusts the ore washer makes of a raw ore.
	washerDust = 2
)

// newOreMetals have an ore block and a raw ore of the game's own.
var newOreMetals = []string{MetalTin, MetalLead, MetalSilver, MetalNickel}

var oreBlockHashes = func() map[string]uint64 {
	m := map[string]uint64{}
	for _, metal := range newOreMetals {
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
func (o OreBlock) Hash() (uint64, uint64)    { return oreBlockHashes[o.Metal], 0 }
func (OreBlock) Model() world.BlockModel     { return model.Solid{} }
func (OreBlock) Category() category.Category { return category.Nature() }
func (o OreBlock) Name() string {
	return metalName(o.Metal) + i18n.T(i18n.Default, "name.OreBlock.suffix")
}
func (o OreBlock) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": solidMaterial("dragonfactory_" + o.Metal + "_ore")}}
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
	return i18n.T(i18n.Default, "name.RawOre.prefix") + metalName(r.Metal)
}
func (RawOre) Category() category.Category { return category.Items() }

// SmeltInfo implements item.Smeltable.
func (r RawOre) SmeltInfo() item.SmeltInfo {
	return item.SmeltInfo{Product: item.NewStack(Ingot{Metal: r.Metal}, 1), Experience: 0.7, Ores: true}
}

// rawOf is the raw ore of a metal.
func rawOf(metal string) world.Item {
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
	for _, k := range oreKinds {
		if key := itemKey(k.block); blocks[key] == "" {
			blocks[key] = k.ore
		}
		if key := itemKey(rawOf(k.ore)); raws[key] == "" {
			raws[key] = k.ore
		}
	}
	return blocks, raws
}()

// oreBlockMetal is the metal of an ore block.
func oreBlockMetal(it world.Item) (string, bool) {
	metal, ok := oreBlockMetals[itemKey(it)]
	return metal, ok
}

// rawMetal is the metal of a raw ore.
func rawMetal(it world.Item) (string, bool) {
	metal, ok := rawMetals[itemKey(it)]
	return metal, ok
}

// oreBlocks are the ore blocks of every metal.
func oreBlocks() []world.Item {
	var out []world.Item
	for _, k := range oreKinds {
		out = append(out, k.block)
	}
	return out
}

// hammerDrop is what the hammer breaks b into: an ore block into its dusts,
// the stones one step down (crushRecipes).
func hammerDrop(b world.Block) (item.Stack, bool) {
	it, ok := b.(world.Item)
	if !ok {
		return item.Stack{}, false
	}
	if metal, ok := oreBlockMetal(it); ok {
		return item.NewStack(MetalDust{Metal: metal}, hammerDust), true
	}
	if out, ok := crushed(b); ok {
		return item.NewStack(out, 1), true
	}
	return item.Stack{}, false
}

// washerOutput is what the ore washer makes of a raw ore.
func washerOutput(in item.Stack) (item.Stack, bool) {
	metal, ok := rawMetal(in.Item())
	if !ok {
		return item.Stack{}, false
	}
	return item.NewStack(MetalDust{Metal: metal}, washerDust), true
}

func init() {
	for _, metal := range newOreMetals {
		world.RegisterBlock(OreBlock{Metal: metal})
		world.RegisterItem(OreBlock{Metal: metal})
		world.RegisterItem(RawOre{Metal: metal})
	}
}
