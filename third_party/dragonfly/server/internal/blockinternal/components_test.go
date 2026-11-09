package blockinternal

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

// skygear patch: tests of Properties.BoneVisibility in the minecraft:geometry component.

func TestBoneVisibilityEncoded(t *testing.T) {
	c := componentsFromProperties(customblock.Properties{
		Geometry:       "geometry.test",
		BoneVisibility: map[string]string{"core": "true", "arm": "false", "up": "q.block_state('t:s') == 1"},
	})
	want := map[string]any{
		"identifier": "geometry.test",
		"bone_visibility": map[string]any{
			"core": "true", "arm": "false", "up": "q.block_state('t:s') == 1",
		},
	}
	if got := c["minecraft:geometry"]; !reflect.DeepEqual(got, want) {
		t.Errorf("minecraft:geometry = %v, want %v", got, want)
	}
}

// TestBoneVisibilityAbsentByDefault: with no BoneVisibility the component is
// what it was before the patch.
func TestBoneVisibilityAbsentByDefault(t *testing.T) {
	c := componentsFromProperties(customblock.Properties{Geometry: "geometry.test"})
	if want := map[string]any{"identifier": "geometry.test"}; !reflect.DeepEqual(c["minecraft:geometry"], want) {
		t.Errorf("minecraft:geometry = %v, want %v", c["minecraft:geometry"], want)
	}
	// without a geometry there is nothing to hide bones of
	c = componentsFromProperties(customblock.Properties{BoneVisibility: map[string]string{"core": "false"}})
	if _, ok := c["minecraft:geometry"]; ok {
		t.Errorf("minecraft:geometry set without a geometry: %v", c)
	}
}

type boneBlock struct{}

func (boneBlock) EncodeBlock() (string, map[string]any) {
	return "t:bones", map[string]any{"t:s": int32(0)}
}
func (boneBlock) Hash() (uint64, uint64)  { return 1, 0 }
func (boneBlock) Model() world.BlockModel { return model.Empty{} }
func (boneBlock) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.test", BoneVisibility: map[string]string{"core": "false"}}
}
func (boneBlock) States() map[string][]any { return map[string][]any{"t:s": {int32(0), int32(1)}} }
func (boneBlock) Permutations() []customblock.Permutation {
	return []customblock.Permutation{{
		Condition: "q.block_state('t:s') == 1",
		Properties: customblock.Properties{
			Geometry:       "geometry.test",
			BoneVisibility: map[string]string{"core": "true"},
		},
	}}
}

// TestBoneVisibilityInPermutations: the base components and each permutation
// carry their own bone_visibility.
func TestBoneVisibilityInPermutations(t *testing.T) {
	out := Components("t:bones", boneBlock{}, 1)
	geo := func(components map[string]any) map[string]any {
		return components["minecraft:geometry"].(map[string]any)["bone_visibility"].(map[string]any)
	}
	if got := geo(out["components"].(map[string]any)); !reflect.DeepEqual(got, map[string]any{"core": "false"}) {
		t.Errorf("base bone_visibility = %v", got)
	}
	perms := out["permutations"].([]map[string]any)
	if len(perms) != 1 {
		t.Fatalf("%d permutations", len(perms))
	}
	if got := geo(perms[0]["components"].(map[string]any)); !reflect.DeepEqual(got, map[string]any{"core": "true"}) {
		t.Errorf("permutation bone_visibility = %v", got)
	}
}

// skygear patch: the properties of a block with several states go out in the
// order of their names, every time (the order the client numbers the states in).

type manyStatesBlock struct{ boneBlock }

func (manyStatesBlock) States() map[string][]any {
	return map[string][]any{
		"t:w": {false, true}, "t:a": {int32(0), int32(1)}, "t:m": {int32(0)}, "t:c": {false, true},
		"t:z": {int32(0)}, "t:b": {int32(0)}, "t:q": {false, true}, "t:e": {int32(0)},
	}
}

func TestPropertiesSortedByName(t *testing.T) {
	want := []string{"t:a", "t:b", "t:c", "t:e", "t:m", "t:q", "t:w", "t:z"}
	for range 20 {
		props := Components("t:many", manyStatesBlock{}, 1)["properties"].([]map[string]any)
		var got []string
		for _, p := range props {
			got = append(got, p["name"].(string))
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("properties %v, want %v", got, want)
		}
		if !reflect.DeepEqual(props[6]["enum"], []any{false, true}) {
			t.Fatalf("t:w enum %v", props[6]["enum"])
		}
	}
}
