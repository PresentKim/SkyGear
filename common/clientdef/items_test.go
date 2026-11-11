package clientdef

import (
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// base is the part of a custom item every test item has.
type base struct{ id string }

func (b base) EncodeItem() (string, int16) { return "clientdef_test:" + b.id, 0 }
func (b base) Name() string                { return b.id }
func (b base) Category() category.Category { return category.Items() }
func (base) Texture() image.Image          { return nil }

type fuel struct{ base }

func (fuel) FuelInfo() item.FuelInfo { return item.FuelInfo{Duration: 5 * time.Second} }

type noFuel struct{ base }

func (noFuel) FuelInfo() item.FuelInfo { return item.FuelInfo{} }

type digger struct{ base }

func (digger) DestroySpeeds() map[string]int {
	return map[string]int{"minecraft:stone": 4, "minecraft:dirt": 2, "minecraft:gravel": 3}
}
func (digger) DestroySpeedsByTag() []TagSpeed {
	return []TagSpeed{{Tags: "q1", Speed: 8}, {Tags: "q2", Speed: 6}}
}

type usable struct{ base }

func (usable) Use(*world.Tx, item.User, *item.UseContext) bool { return true }

type food struct{ usable }

func (food) AlwaysConsumable() bool                      { return false }
func (food) ConsumeDuration() time.Duration              { return time.Second }
func (food) Consume(*world.Tx, item.Consumer) item.Stack { return item.Stack{} }

type plain struct{ base }

func init() {
	for _, it := range []world.CustomItem{
		fuel{base{"fuel"}}, noFuel{base{"nofuel"}}, digger{base{"digger"}},
		usable{base{"usable"}}, food{usable{base{"food"}}}, plain{base{"plain"}},
	} {
		world.RegisterItem(it)
	}
}

// entry is the entry Dragonfly builds for the test item id: a components map
// with a display name.
func entry(id string) protocol.ItemEntry {
	return protocol.ItemEntry{
		Name:           "clientdef_test:" + id,
		ComponentBased: true,
		Data: map[string]any{
			"name": "clientdef_test:" + id,
			"components": map[string]any{
				"minecraft:display_name": map[string]any{"value": id},
			},
		},
	}
}

func comps(e protocol.ItemEntry) map[string]any { return e.Data["components"].(map[string]any) }

// TestFuelComponent: a custom fuel sends minecraft:fuel with its burn time in
// seconds, so clients let it into a furnace's fuel slot; a zero burn time
// sends none.
func TestFuelComponent(t *testing.T) {
	f, ok := comps(ItemEntry(entry("fuel")))["minecraft:fuel"].(map[string]any)
	if !ok || len(f) != 1 || f["duration"] != float32(5) {
		t.Fatalf("fuel component %#v", f)
	}
	if _, ok := comps(ItemEntry(entry("nofuel")))["minecraft:fuel"]; ok {
		t.Fatal("fuel component of an item that does not burn")
	}
}

// TestDiggerComponent: the block speeds come by sorted name, the tag speeds
// after them in order, all int32.
func TestDiggerComponent(t *testing.T) {
	d, ok := comps(ItemEntry(entry("digger")))["minecraft:digger"].(map[string]any)
	if !ok || d["use_efficiency"] != true || len(d) != 2 {
		t.Fatalf("digger component %#v", d)
	}
	block := func(k, v string, speed int32) any {
		return map[string]any{"block": map[string]any{k: v}, "speed": speed}
	}
	want := []any{
		block("name", "minecraft:dirt", 2), block("name", "minecraft:gravel", 3),
		block("name", "minecraft:stone", 4), block("tags", "q1", 8), block("tags", "q2", 6),
	}
	if !reflect.DeepEqual(d["destroy_speeds"], want) {
		t.Fatalf("destroy_speeds %#v, want %#v", d["destroy_speeds"], want)
	}
}

// TestUseModifiers: an item usable in the air sends minecraft:use_modifiers,
// a consumable does not.
func TestUseModifiers(t *testing.T) {
	want := map[string]any{"use_duration": float32(0.05), "movement_modifier": float32(1)}
	if got := comps(ItemEntry(entry("usable")))["minecraft:use_modifiers"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("use_modifiers %#v, want %#v", got, want)
	}
	if _, ok := comps(ItemEntry(entry("food")))["minecraft:use_modifiers"]; ok {
		t.Fatal("use_modifiers on a consumable")
	}
}

// TestItemEntryUnchanged: the entry of a vanilla item, of an item with none of
// the components, and the maps of the entry given are left as they are.
func TestItemEntryUnchanged(t *testing.T) {
	for _, e := range []protocol.ItemEntry{entry("plain"), {Name: "minecraft:stick"}} {
		if got := ItemEntry(e); !reflect.DeepEqual(got, e) {
			t.Fatalf("entry %q changed to %#v", e.Name, got)
		}
	}
	e := entry("digger")
	ItemEntry(e)
	if !reflect.DeepEqual(e, entry("digger")) {
		t.Fatal("ItemEntry changed the maps of its argument")
	}
	if got := comps(ItemEntry(e))["minecraft:display_name"]; !reflect.DeepEqual(got, map[string]any{"value": "digger"}) {
		t.Fatalf("display_name %#v lost", got)
	}
}

// TestItemEntryNoComponents: a component due on an entry without components
// panics, naming the item.
func TestItemEntryNoComponents(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r.(string) != "clientdef: item entry clientdef_test:fuel has no components" {
			t.Fatalf("recovered %v", r)
		}
	}()
	ItemEntry(protocol.ItemEntry{Name: "clientdef_test:fuel"})
}
