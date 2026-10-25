package oreui

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestStoreValueTree(t *testing.T) {
	v, err := storeValue(map[string]any{"name": "box", "slots": []any{map[string]any{"id": "stone", "amount": 3}}, "open": true, "scale": 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != protocol.DataStorePropertyTypeMap || len(v.MapValue) != 4 {
		t.Fatalf("got %+v", v)
	}
	keys := []string{}
	for _, e := range v.MapValue {
		keys = append(keys, e.Key)
	}
	if keys[0] != "name" || keys[1] != "open" || keys[2] != "scale" || keys[3] != "slots" {
		t.Fatalf("keys %v", keys)
	}
	slot := v.MapValue[3].Value.ListValue[0].MapValue
	if slot[0].Key != "amount" || slot[0].Value.Int64Value != 3 || slot[1].Value.StringValue != "stone" {
		t.Fatalf("slot %+v", slot)
	}
}

func TestStoreValueRejectsUnknownTypes(t *testing.T) {
	if _, err := storeValue(struct{}{}); err == nil {
		t.Fatal("no error")
	}
}
