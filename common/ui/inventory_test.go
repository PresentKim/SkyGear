package ui

import (
	"encoding/json"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestSlotEvent(t *testing.T) {
	a, ok := slotEvent("craft:4", map[string]any{"b": float64(1), "shift": true})
	if !ok || a.Area != "craft" || a.Index != 4 || !a.Right || !a.Shift || a.Double {
		t.Errorf("slot event %+v %v", a, ok)
	}
	if _, ok := slotEvent("nocolon", nil); ok {
		t.Error("an id without an index was read")
	}
	if _, ok := slotEvent("inv:-1", nil); ok {
		t.Error("a negative index was read")
	}
}

func TestDragEventDropsRepeats(t *testing.T) {
	d, ok := dragEvent(map[string]any{"b": float64(0), "ids": []any{"inv:1", "inv:2", "inv:1", "bad"}})
	if !ok || d.Right || len(d.Slots) != 2 || d.Slots[1] != (SlotRef{"inv", 2}) {
		t.Errorf("drag event %+v %v", d, ok)
	}
	if _, ok := dragEvent(map[string]any{"ids": []any{}}); ok {
		t.Error("an empty drag was read")
	}
}

func TestShowInventoryData(t *testing.T) {
	var got map[string]any
	var layout *oreLayout
	oreShown = func(l *oreLayout, root any) {
		layout = l
		b, _ := json.Marshal(root)
		_ = json.Unmarshal(b, &got)
	}
	defer func() { oreShown = nil }()
	ShowInventory(nil, InventoryScreen{
		Title:  "Terminal",
		Search: &InvSearch{Query: "a", Placeholder: "Name", Change: func(*player.Player, *world.Tx, string) {}},
		Rows: []InvRow{
			{Grow: true, Parts: []InvPart{{Area: "net", Cols: 9, Slots: []InvSlot{{Icon: "textures/x", Count: "3k", Tip: "Dirt"}}}}},
			{Parts: []InvPart{{Area: "craft", Cols: 3, Slots: make([]InvSlot, 9)}, {Arrow: true}, {Area: "result", Cols: 1, Big: true, Slots: []InvSlot{{}}}}},
		},
		Side: []Option{{Label: "Sort", Icon: "textures/ui/store_filter_icon", Run: func(*player.Player, *world.Tx) {}}},
		Slot: func(*player.Player, *world.Tx, SlotAct) {},
		Size: "terminal",
	})
	if got["screen"] != "inventory" || layout.size != "terminal" || !layout.inventory {
		t.Fatalf("root %v, layout %+v", got["screen"], layout)
	}
	d := got["data"].(map[string]any)
	rows := d["rows"].([]any)
	net := rows[0].(map[string]any)["parts"].([]any)[0].(map[string]any)
	slot := net["slots"].([]any)[0].(map[string]any)
	if slot["id"] != "net:0" || slot["count"] != "3k" || rows[0].(map[string]any)["grow"] != true {
		t.Errorf("net slot %v", slot)
	}
	if d["field"] != "search" || d["query"] != "a" || d["cursor"] != nil || len(d["side"].([]any)) != 1 {
		t.Errorf("data %v", d)
	}
	if layout.slot == nil || layout.input == nil || layout.close == nil {
		t.Error("the layout misses its runs")
	}
}
