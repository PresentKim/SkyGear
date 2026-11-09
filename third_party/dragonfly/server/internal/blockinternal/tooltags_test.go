package blockinternal

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
)

// TestToolTags: a block gets the tags of the tools its BreakInfo takes, and a
// pickaxe block the tier it needs to drop anything. (skygear patch)
func TestToolTags(t *testing.T) {
	pick := func(t item.Tool) bool { return t.ToolType() == item.TypePickaxe }
	axe := func(t item.Tool) bool { return t.ToolType() == item.TypeAxe }
	always := func(item.Tool) bool { return true }
	stoneUp := func(t item.Tool) bool { return pick(t) && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel }
	for _, c := range []struct {
		name string
		info block.BreakInfo
		want []string
	}{
		{"wooden", block.BreakInfo{Effective: axe, Harvestable: always}, []string{"minecraft:is_axe_item_destructible"}},
		{"machine", block.BreakInfo{Effective: pick, Harvestable: always}, []string{"minecraft:is_pickaxe_item_destructible"}},
		{"ore", block.BreakInfo{Effective: pick, Harvestable: stoneUp}, []string{"minecraft:is_pickaxe_item_destructible", "stone_pick_diggable"}},
		{"no tool", block.BreakInfo{Effective: func(item.Tool) bool { return false }, Harvestable: always}, nil},
	} {
		if got := toolTags(c.info); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
