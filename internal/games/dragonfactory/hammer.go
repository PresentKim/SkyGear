package dragonfactory

import (
	"image"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
)

// Hammer crushes blocks by hand: a block it mines drops what the crusher
// makes of it (stone → cobblestone → gravel → sand). It mines these blocks
// at stone-tool speed; the speeds are sent to clients (the minecraft:digger
// component, a Dragonfly patch) so they mine as fast as the server expects.
type Hammer struct{}

// hammerSpeeds are the mining speeds of the hammer per block.
var hammerSpeeds = map[string]int{
	"minecraft:stone":       6,
	"minecraft:cobblestone": 6,
	"minecraft:gravel":      6,
	"minecraft:sand":        6,
	"minecraft:dirt":        6,
}

func (Hammer) EncodeItem() (string, int16) { return "dragonfactory:hammer", 0 }
func (Hammer) Name() string                { return "망치" }
func (Hammer) Texture() image.Image        { return itemTexture("hammer") }
func (Hammer) Category() category.Category { return category.Equipment() }
func (Hammer) MaxCount() int               { return 1 }
func (Hammer) HandEquipped() bool          { return true }
func (Hammer) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{MaxDurability: 131, BrokenItem: func() item.Stack { return item.Stack{} }}
}

// ToolType, HarvestLevel and BaseMiningEfficiency make the hammer a stone
// pickaxe for the server's break times and drops.
func (Hammer) ToolType() item.ToolType { return item.TypePickaxe }
func (Hammer) HarvestLevel() int       { return item.ToolTierStone.HarvestLevel }
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

func isHammer(st item.Stack) bool {
	_, ok := st.Item().(Hammer)
	return ok
}

// crushed returns what b turns into when crushed.
func crushed(b world.Block) (world.Item, bool) {
	it, ok := b.(world.Item)
	if !ok {
		return nil, false
	}
	out, ok := crushRecipes[itemKey(it)]
	return out, ok
}

func init() {
	world.RegisterItem(Hammer{})
}
