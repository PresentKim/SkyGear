package progress

// The research table, where research is done.

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/core"
)

// ResearchTable is the block where research is done. It is crafted from two
// crafting tables (open from the start).
type ResearchTable struct{}

var researchTableHash = block.NextHash()

func (ResearchTable) EncodeItem() (string, int16) { return "skygear:research_table", 0 }

func (ResearchTable) EncodeBlock() (string, map[string]any) {
	return "skygear:research_table", nil
}

func (ResearchTable) Hash() (uint64, uint64) { return researchTableHash, 0 }

func (ResearchTable) Model() world.BlockModel { return model.Solid{} }

func (ResearchTable) Name() string { return i18n.T(i18n.Default, "name.ResearchTable") }

func (ResearchTable) Category() category.Category { return category.Items() }

func (ResearchTable) Properties() customblock.Properties {
	return blockmodel.CubeProperties("research_table")
}

func (t ResearchTable) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2.5, BlastResistance: 2.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{item.NewStack(t, 1)} },
	}
}

// Activate opens the research tree.
func (ResearchTable) Activate(_ cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok {
		return false
	}
	s, ok := core.Of(tx)
	if !ok {
		return false
	}
	if !s.Debounced(p) {
		ShowScreen(s, p)
	}
	return true
}

// ShowScreen opens the research screen (the game sets it).
var ShowScreen = func(s *core.Session, p *player.Player) {}

func init() {
	world.RegisterBlock(ResearchTable{})
	world.RegisterItem(ResearchTable{})
	table := item.NewStack(block.CraftingTable{}, 1)
	recipe.Register(recipe.NewShapeless([]recipe.Item{table, table}, item.NewStack(ResearchTable{}, 1), "crafting_table"))
}
