package dragonfactory

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestWoodenBucketAimsAtLiquids(t *testing.T) {
	shared := map[string]any{"max_stack_size": int32(16)}
	pk := &packet.ItemRegistry{Items: []protocol.ItemEntry{
		{Name: "dragonfactory:wooden_bucket", ComponentBased: true, Data: map[string]any{"components": map[string]any{"item_properties": shared}}},
		{Name: "dragonfactory:wooden_water_bucket", ComponentBased: true, Data: map[string]any{"components": map[string]any{"item_properties": map[string]any{}}}},
	}}
	clipWoodenBucket(nil, pk)
	props := pk.Items[0].Data["components"].(map[string]any)["item_properties"].(map[string]any)
	if props["liquid_clipped"] != true || props["max_stack_size"] != int32(16) {
		t.Fatalf("properties %v", props)
	}
	if _, ok := shared["liquid_clipped"]; ok {
		t.Fatal("changed the entry Dragonfly shares between players")
	}
	if _, ok := pk.Items[1].Data["components"].(map[string]any)["item_properties"].(map[string]any)["liquid_clipped"]; ok {
		t.Fatal("the full bucket aims at liquids")
	}
}
