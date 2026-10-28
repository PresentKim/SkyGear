package sgitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/games/skygear/blockmodel"
)

// Crushed blocks are the stones of the deep, the Nether and the End broken
// once by the hammer or the crusher, as Ex Deorum makes them in ATM10: To the
// Sky. A sieve sifts them for their own
// drops; crushed netherrack crushes once more into red sand.
const (
	CrushedDeepslate  = "deepslate"
	CrushedNetherrack = "netherrack"
	CrushedBlackstone = "blackstone"
	CrushedEndStone   = "end_stone"
)

// CrushedKinds are the crushed blocks in registration order.
var CrushedKinds = []string{CrushedDeepslate, CrushedNetherrack, CrushedBlackstone, CrushedEndStone}

var crushedHashes = func() map[string]uint64 {
	m := map[string]uint64{}
	for _, k := range CrushedKinds {
		m[k] = block.NextHash()
	}
	return m
}()

// CrushedBlock is a crushed stone of Kind.
type CrushedBlock struct{ Kind string }

func (c CrushedBlock) EncodeItem() (string, int16) { return "skygear:crushed_" + c.Kind, 0 }

func (c CrushedBlock) EncodeBlock() (string, map[string]any) {
	return "skygear:crushed_" + c.Kind, nil
}

func (c CrushedBlock) Hash() (uint64, uint64) { return crushedHashes[c.Kind], 0 }

func (CrushedBlock) Model() world.BlockModel { return model.Solid{} }

func (c CrushedBlock) Name() string { return i18n.T(i18n.Default, "name.Crushed."+c.Kind) }

func (CrushedBlock) Category() category.Category { return category.Construction() }

func (c CrushedBlock) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{"*": blockmodel.SolidMaterial("skygear_crushed_" + c.Kind)}}
}

func (c CrushedBlock) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 0.6, BlastResistance: 0.6,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(c, 1)} },
	}
}
