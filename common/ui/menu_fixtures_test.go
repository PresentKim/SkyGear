package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/internal/slottext"
)

// fixtureSlot is a slot of the menu window as mcbe-json-ui-tester's chest
// host reads it (cli.mjs --slots).
type fixtureSlot struct {
	Name  string `json:"name"`
	IDAux int64  `json:"id_aux"`
	Count int    `json:"count"`
}

// TestDumpMenuSlots writes the menu window slots of every fixture as
// <name>.json in $INVUI_SLOTS, for the JSON UI tester.
func TestDumpMenuSlots(t *testing.T) {
	dir := os.Getenv("INVUI_SLOTS")
	if dir == "" {
		t.Skip("INVUI_SLOTS not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range fixtureNames {
		roots := runFixture(t, name)
		root, ok := roots[0].(map[string]any)
		if !ok {
			t.Fatalf("%s: root %T", name, roots[0])
		}
		pg := menuPageOf(root["screen"].(string), root["data"])
		slots := make([]fixtureSlot, len(pg.items))
		for i, it := range pg.items {
			slots[i] = slotOf(it)
		}
		data, err := json.Marshal(slots)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// slotOf is the slot the client shows for it: the name with the "§f" the
// client puts in front of a custom name.
func slotOf(it item.Stack) fixtureSlot {
	if it.Empty() {
		return fixtureSlot{}
	}
	aux, _ := slottext.ItemAux(it.Item())
	return fixtureSlot{Name: "§f" + it.CustomName(), IDAux: aux, Count: it.Count()}
}
