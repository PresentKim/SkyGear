package clientdef

import (
	"reflect"
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func pickaxe(t item.Tool) bool { return t.ToolType() == item.TypePickaxe }
func always(item.Tool) bool    { return true }
func stoneUp(t item.Tool) bool {
	return pickaxe(t) && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
}

// blk is a custom block with no behaviour; the types below add some.
type blk struct{ id string }

func (b blk) EncodeBlock() (string, map[string]any) { return "clientdef_test:" + b.id, nil }
func (b blk) Hash() (uint64, uint64)                { return 0xc11e0000, uint64(len(b.id)) }
func (blk) Model() world.BlockModel                 { return model.Empty{} }
func (blk) Properties() customblock.Properties      { return customblock.Properties{} }

// oreBlk needs a stone pickaxe to drop anything.
type oreBlk struct{ blk }

func (oreBlk) BreakInfo() block.BreakInfo {
	return block.BreakInfo{Effective: pickaxe, Harvestable: stoneUp}
}

// clickBlk reacts to clicks.
type clickBlk struct{ blk }

func (clickBlk) Activate(cube.Pos, cube.Face, *world.Tx, item.User, *item.UseContext) bool {
	return true
}

func init() {
	world.RegisterBlock(oreBlk{blk{"ore"}})
	world.RegisterBlock(clickBlk{blk{"click"}})
	world.RegisterBlock(blk{"plain"})
}

func blockEntry(name string, props ...string) protocol.BlockEntry {
	var list []map[string]any
	for _, p := range props {
		list = append(list, map[string]any{"name": p, "enum": []any{int32(0)}})
	}
	return protocol.BlockEntry{Name: name, Properties: map[string]any{
		"components": map[string]any{"minecraft:friction": map[string]any{"value": float32(0.5)}},
		"properties": list,
		"block_id":   int32(7),
	}}
}

func names(t *testing.T, e protocol.BlockEntry) []string {
	t.Helper()
	var got []string
	for _, p := range e.Properties["properties"].([]map[string]any) {
		got = append(got, p["name"].(string))
	}
	return got
}

// TestToolTags: a block gets the tags of the tools its BreakInfo takes, and a
// pickaxe block the tier it needs to drop anything.
func TestToolTags(t *testing.T) {
	axe := func(t item.Tool) bool { return t.ToolType() == item.TypeAxe }
	for _, c := range []struct {
		name string
		info block.BreakInfo
		want []string
	}{
		{"wooden", block.BreakInfo{Effective: axe, Harvestable: always}, []string{"minecraft:is_axe_item_destructible"}},
		{"machine", block.BreakInfo{Effective: pickaxe, Harvestable: always}, []string{"minecraft:is_pickaxe_item_destructible"}},
		{"ore", block.BreakInfo{Effective: pickaxe, Harvestable: stoneUp}, []string{"minecraft:is_pickaxe_item_destructible", "stone_pick_diggable"}},
		{"no tool", block.BreakInfo{Effective: func(item.Tool) bool { return false }, Harvestable: always}, nil},
	} {
		if got := toolTags(c.info); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// TestBlockEntryToolTags: the entry of a registered breakable block gains its
// tags as empty compounds, next to what it had.
func TestBlockEntryToolTags(t *testing.T) {
	in := blockEntry("clientdef_test:ore")
	comps := BlockEntry(in).Properties["components"].(map[string]any)
	for _, tag := range []string{"tag:minecraft:is_pickaxe_item_destructible", "tag:stone_pick_diggable"} {
		if v, ok := comps[tag]; !ok || !reflect.DeepEqual(v, map[string]any{}) {
			t.Errorf("%s = %v, %v", tag, v, ok)
		}
	}
	if _, ok := comps["minecraft:friction"]; !ok {
		t.Error("friction lost")
	}
}

// TestInteractFlag: an activatable block is marked interactable, a plain one
// is not.
func TestInteractFlag(t *testing.T) {
	want := map[string]any{"hasPlayerInteract": uint8(1), "hasPlayerPlacing": uint8(0), "isV1Component": uint8(1)}
	got := BlockEntry(blockEntry("clientdef_test:click")).Properties["components"].(map[string]any)
	if !reflect.DeepEqual(got["minecraft:custom_components"], want) {
		t.Errorf("activatable: %v", got["minecraft:custom_components"])
	}
	for _, name := range []string{"clientdef_test:plain", "clientdef_test:ore"} {
		got := BlockEntry(blockEntry(name)).Properties["components"].(map[string]any)
		if v, ok := got["minecraft:custom_components"]; ok {
			t.Errorf("%s has custom_components %v", name, v)
		}
	}
}

// TestPropertiesSortedByName: the properties go out in the order of their
// names, whatever order they came in; the input list stays as it was.
func TestPropertiesSortedByName(t *testing.T) {
	in := blockEntry("clientdef_test:plain", "t:w", "t:a", "t:m", "t:c", "t:z", "t:b", "t:q", "t:e")
	want := []string{"t:a", "t:b", "t:c", "t:e", "t:m", "t:q", "t:w", "t:z"}
	out := BlockEntry(in)
	if got := names(t, out); !slices.Equal(got, want) {
		t.Errorf("properties %v, want %v", got, want)
	}
	if got := names(t, in); got[0] != "t:w" {
		t.Errorf("input list sorted in place: %v", got)
	}
	if !reflect.DeepEqual(out.Properties["properties"].([]map[string]any)[0]["enum"], []any{int32(0)}) {
		t.Error("enum lost")
	}
}

// TestBlockEntryUnchanged: the entry of a block no one registered (a
// data-driven one), and one with nothing due, come back as they were.
func TestBlockEntryUnchanged(t *testing.T) {
	for _, name := range []string{"clientdef_test:data_driven", "clientdef_test:plain"} {
		in := blockEntry(name, "t:a")
		if out := BlockEntry(in); !reflect.DeepEqual(out, blockEntry(name, "t:a")) {
			t.Errorf("%s changed: %v", name, out)
		}
	}
	noProps := protocol.BlockEntry{Name: "clientdef_test:data_driven", Properties: map[string]any{"components": map[string]any{}}}
	if out := BlockEntry(noProps); len(out.Properties) != 1 {
		t.Errorf("properties added: %v", out.Properties)
	}
}
