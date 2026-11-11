package clientdef

import (
	"reflect"
	"slices"
	"strings"
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

// extraBlk sends bones and collision boxes for itself and one permutation.
type extraBlk struct{ blk }

const extraCond = "q.block_state('t:s') == 1"

func (extraBlk) ClientExtras() (Extras, map[string]Extras) {
	return Extras{
			CollisionBoxes: []cube.BBox{cube.Box(0, 0, 0, 1, 0.125, 1), cube.Box(0, 0, 0, 0.125, 1, 1)},
			BoneVisibility: map[string]string{"core": "true", "arm": "false", "up": "q.block_state('t:s') == 1"},
		}, map[string]Extras{
			extraCond: {BoneVisibility: map[string]string{"core": "false"}},
		}
}

// boneOnlyBlk has bones in its base only.
type boneOnlyBlk struct{ blk }

func (boneOnlyBlk) ClientExtras() (Extras, map[string]Extras) {
	return Extras{BoneVisibility: map[string]string{"core": "false"}}, nil
}

// wrongCondBlk names a permutation condition the entry lacks.
type wrongCondBlk struct{ blk }

func (wrongCondBlk) ClientExtras() (Extras, map[string]Extras) {
	return Extras{}, map[string]Extras{"nope": {BoneVisibility: map[string]string{"core": "false"}}}
}

func init() {
	world.RegisterBlock(extraBlk{blk{"extra"}})
	world.RegisterBlock(boneOnlyBlk{blk{"boneonly"}})
	world.RegisterBlock(wrongCondBlk{blk{"wrongcond"}})
}

// extraEntry is an entry with a geometry and one permutation with its own.
func extraEntry(name string) protocol.BlockEntry {
	e := blockEntry(name)
	e.Properties["components"].(map[string]any)["minecraft:geometry"] = map[string]any{"identifier": "geometry.test"}
	e.Properties["components"].(map[string]any)["minecraft:collision_box"] = map[string]any{"enabled": true, "origin": []float32{1}}
	e.Properties["permutations"] = []map[string]any{{
		"condition":  extraCond,
		"components": map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.test"}},
	}, {
		"condition":  "other",
		"components": map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.test"}},
	}}
	return e
}

func geometry(components any) map[string]any {
	return components.(map[string]any)["minecraft:geometry"].(map[string]any)
}

// TestBoneVisibilityEncoded: the bones go into minecraft:geometry as strings.
func TestBoneVisibilityEncoded(t *testing.T) {
	out := BlockEntry(extraEntry("clientdef_test:extra"))
	want := map[string]any{
		"identifier": "geometry.test",
		"bone_visibility": map[string]any{
			"core": "true", "arm": "false", "up": "q.block_state('t:s') == 1",
		},
	}
	if got := geometry(out.Properties["components"]); !reflect.DeepEqual(got, want) {
		t.Errorf("minecraft:geometry = %v, want %v", got, want)
	}
}

// TestBoneVisibilityAbsentByDefault: a block with no extras keeps its
// geometry as it was, and the input entry is not modified.
func TestBoneVisibilityAbsentByDefault(t *testing.T) {
	out := BlockEntry(extraEntry("clientdef_test:plain"))
	if want := map[string]any{"identifier": "geometry.test"}; !reflect.DeepEqual(geometry(out.Properties["components"]), want) {
		t.Errorf("minecraft:geometry = %v", geometry(out.Properties["components"]))
	}
	in := extraEntry("clientdef_test:extra")
	BlockEntry(in)
	if _, ok := geometry(in.Properties["components"])["bone_visibility"]; ok {
		t.Error("the input entry was modified")
	}
	if _, ok := geometry(in.Properties["permutations"].([]map[string]any)[0]["components"])["bone_visibility"]; ok {
		t.Error("the input permutation was modified")
	}
}

// TestBoneVisibilityInPermutations: the base components and the permutation
// of the same condition carry their own bone_visibility; another one none.
func TestBoneVisibilityInPermutations(t *testing.T) {
	out := BlockEntry(extraEntry("clientdef_test:extra"))
	perms := out.Properties["permutations"].([]map[string]any)
	if len(perms) != 2 {
		t.Fatalf("%d permutations", len(perms))
	}
	if got := geometry(perms[0]["components"])["bone_visibility"]; !reflect.DeepEqual(got, map[string]any{"core": "false"}) {
		t.Errorf("permutation bone_visibility = %v", got)
	}
	if _, ok := geometry(perms[1]["components"])["bone_visibility"]; ok {
		t.Error("a permutation of another condition got bones")
	}
	if _, ok := perms[0]["components"].(map[string]any)["minecraft:collision_box"]; ok {
		t.Error("a permutation without boxes got a collision box")
	}
}

// TestCollisionBoxes: two boxes, in pixels as float32, enabled, replacing a
// single box.
func TestCollisionBoxes(t *testing.T) {
	out := BlockEntry(extraEntry("clientdef_test:extra"))
	want := map[string]any{"enabled": true, "boxes": []map[string]any{
		{"minX": float32(0), "minY": float32(0), "minZ": float32(0), "maxX": float32(16), "maxY": float32(2), "maxZ": float32(16)},
		{"minX": float32(0), "minY": float32(0), "minZ": float32(0), "maxX": float32(2), "maxY": float32(16), "maxZ": float32(16)},
	}}
	if got := out.Properties["components"].(map[string]any)["minecraft:collision_box"]; !reflect.DeepEqual(got, want) {
		t.Errorf("minecraft:collision_box = %v, want %v", got, want)
	}
}

// TestExtrasPanics: bones without a geometry, and a condition without a
// permutation, panic naming the block.
func TestExtrasPanics(t *testing.T) {
	for _, name := range []string{"clientdef_test:boneonly", "clientdef_test:wrongcond"} {
		func() {
			defer func() {
				r, _ := recover().(string)
				if !strings.Contains(r, name) {
					t.Errorf("%s: panic %q", name, r)
				}
			}()
			BlockEntry(blockEntry(name))
		}()
	}
}
