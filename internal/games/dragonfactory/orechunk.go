package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// OreChunk is a piece of ore, found by sieving gravel and crushed deepslate
// (sievetable.go). Four of a kind make the ore block (ore.go); a chunk itself
// is neither smelted nor processed.
type OreChunk struct{ Ore string }

// oreKinds are the ores with the name of their chunks and the ore block four
// chunks make.
var oreKinds = []struct {
	ore, name string
	block     world.Item
}{
	{OreIron, i18n.T(i18n.Default, "dragonfactory.orechunk.init.1"), block.IronOre{}},
	{OreGold, i18n.T(i18n.Default, "dragonfactory.orechunk.init.2"), block.GoldOre{}},
	{OreCopper, i18n.T(i18n.Default, "dragonfactory.orechunk.init.3"), block.CopperOre{}},
	{MetalTin, i18n.T(i18n.Default, "metal.tin"), OreBlock{Metal: MetalTin}},
	{MetalLead, i18n.T(i18n.Default, "metal.lead"), OreBlock{Metal: MetalLead}},
	{MetalSilver, i18n.T(i18n.Default, "metal.silver"), OreBlock{Metal: MetalSilver}},
	{MetalNickel, i18n.T(i18n.Default, "metal.nickel"), OreBlock{Metal: MetalNickel}},
}

func (c OreChunk) EncodeItem() (string, int16) { return "dragonfactory:ore_chunk_" + c.Ore, 0 }
func (c OreChunk) Name() string {
	for _, k := range oreKinds {
		if k.ore == c.Ore {
			return k.name + i18n.T(i18n.Default, "name.OreChunk.suffix")
		}
	}
	return i18n.T(i18n.Default, "name.OreChunk")
}
func (OreChunk) Category() category.Category { return category.Items() }

// oreChunkRecipes registers the recipes that make the ore blocks from four
// chunks (open with the research of the metal).
func oreChunkRecipes() {
	for _, k := range oreKinds {
		c := item.NewStack(OreChunk{Ore: k.ore}, 1)
		recipe.Register(recipe.NewShapeless([]recipe.Item{c, c, c, c}, item.NewStack(k.block, 1), "crafting_table"))
	}
}

func init() {
	for _, k := range oreKinds {
		world.RegisterItem(OreChunk{Ore: k.ore})
	}
	oreChunkRecipes()
}
