package sgitem

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestWoodenBucketAimsAtLiquids(t *testing.T) {
	shared := map[string]any{"max_stack_size": int32(16)}
	empty := ClipWoodenBucket(protocol.ItemEntry{Name: "skygear:wooden_bucket", ComponentBased: true,
		Data: map[string]any{"components": map[string]any{"item_properties": shared}}})
	full := ClipWoodenBucket(protocol.ItemEntry{Name: "skygear:wooden_water_bucket", ComponentBased: true,
		Data: map[string]any{"components": map[string]any{"item_properties": map[string]any{}}}})
	props := empty.Data["components"].(map[string]any)["item_properties"].(map[string]any)
	if props["liquid_clipped"] != true || props["max_stack_size"] != int32(16) {
		t.Fatalf("properties %v", props)
	}
	if _, ok := shared["liquid_clipped"]; ok {
		t.Fatal("changed the properties map it was given")
	}
	if _, ok := full.Data["components"].(map[string]any)["item_properties"].(map[string]any)["liquid_clipped"]; ok {
		t.Fatal("the full bucket aims at liquids")
	}
	bare := ClipWoodenBucket(protocol.ItemEntry{Name: "skygear:wooden_bucket", ComponentBased: true,
		Data: map[string]any{"components": map[string]any{}}})
	if bare.Data["components"].(map[string]any)["item_properties"].(map[string]any)["liquid_clipped"] != true {
		t.Fatal("a bucket without item properties does not aim at liquids")
	}
}
