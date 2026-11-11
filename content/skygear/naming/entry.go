package naming

import (
	"maps"

	"github.com/presentkim/skygear/common/timed"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// A timed item is named as any other item of the game.
func init() {
	timed.Name = ItemName
}

// LocalItemName makes the client name an item by its translation key
// ("item.<id>.name", from the pack's texts/*.lang) instead of the Korean name
// Dragonfly puts in minecraft:display_name, so each client shows its own
// language.
func LocalItemName(e protocol.ItemEntry) protocol.ItemEntry {
	if !e.ComponentBased {
		return e
	}
	comps, ok := e.Data["components"].(map[string]any)
	if !ok {
		return e
	}
	if _, ok := comps["minecraft:display_name"]; !ok {
		return e
	}
	comps = maps.Clone(comps)
	comps["minecraft:display_name"] = map[string]any{"value": "item." + e.Name + ".name"}
	e.Data = maps.Clone(e.Data)
	e.Data["components"] = comps
	return e
}
