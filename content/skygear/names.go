package skygear

import (
	"maps"

	"github.com/df-mc/dragonfly/server"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/timed"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func init() {
	timed.Name = naming.ItemName
}

// EditItemEntries changes the item entries srv sends every player
// (sghook.EditCustomItems): our items are named by their translation key and
// the empty wooden bucket aims at liquids. Call it once, before players join.
func EditItemEntries(srv *server.Server) {
	sghook.EditCustomItems(srv, func(e protocol.ItemEntry) protocol.ItemEntry {
		return sgitem.ClipWoodenBucket(localItemName(e))
	})
}

// localItemName makes the client name an item by its translation key
// ("item.<id>.name", from the pack's texts/*.lang) instead of the Korean name
// Dragonfly puts in minecraft:display_name, so each client shows its own
// language.
func localItemName(e protocol.ItemEntry) protocol.ItemEntry {
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
