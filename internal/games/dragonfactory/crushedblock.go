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

// Crushed blocks are the stones of the deep, the Nether and the End broken
// once by the hammer or the crusher, as Ex Deorum makes them in ATM10: To the
// Sky (docs/design/16-ore-balance.md §2.1). A sieve sifts them for their own
// drops; crushed netherrack crushes once more into red sand.
const (
	CrushedDeepslate  = "deepslate"
	CrushedNetherrack = "netherrack"
	CrushedBlackstone = "blackstone"
	CrushedEndStone   = "end_stone"
)

// crushedKinds are the crushed blocks in registration order.
var crushedKinds = []string{CrushedDeepslate, CrushedNetherrack, CrushedBlackstone, CrushedEndStone}

var crushedHashes = func() map[string]uint64 {
	m := map[string]uint64{}
	for _, k := range crushedKinds {
		m[k] = block.NextHash()
	}
	return m
}()

// CrushedBlock is a crushed stone of Kind.
type CrushedBlock struct{ Kind string }

func (c CrushedBlock) EncodeItem() (string, int16) { return "dragonfactory:crushed_" + c.Kind, 0 }
func (c CrushedBlock) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:crushed_" + c.Kind, nil
}
func (c CrushedBlock) Hash() (uint64, uint64)    { return crushedHashes[c.Kind], 0 }
func (CrushedBlock) Model() world.BlockModel     { return model.Solid{} }
func (c CrushedBlock) Name() string              { return i18n.T(i18n.Default, "name.Crushed."+c.Kind) }
func (CrushedBlock) Category() category.Category { return category.Construction() }
func (c CrushedBlock) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": solidMaterial("dragonfactory_crushed_" + c.Kind)}}
}
func (c CrushedBlock) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.6, BlastResistance: 0.6,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(c, 1)} },
	}
}

func init() {
	for _, k := range crushedKinds {
		world.RegisterBlock(CrushedBlock{Kind: k})
		world.RegisterItem(CrushedBlock{Kind: k})
	}
}
