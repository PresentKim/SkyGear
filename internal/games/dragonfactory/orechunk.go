package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
)

// OreChunk is a piece of ore, found by sieving gravel and sand (sievetable.go).
// Four of a kind make the ore block, which a furnace smelts into ingots.
type OreChunk struct{ Ore string }

// oreKinds are the ores with the name of their chunks and the ore block four
// chunks make.
var oreKinds = []struct {
	ore, name string
	block     world.Item
}{
	{OreIron, "철", block.IronOre{}},
	{OreGold, "금", block.GoldOre{}},
	{OreCopper, "구리", block.CopperOre{}},
}

func (c OreChunk) EncodeItem() (string, int16) { return "dragonfactory:ore_chunk_" + c.Ore, 0 }
func (c OreChunk) Name() string {
	for _, k := range oreKinds {
		if k.ore == c.Ore {
			return k.name + " 원석 조각"
		}
	}
	return "원석 조각"
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
