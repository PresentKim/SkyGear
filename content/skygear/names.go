package skygear

import (
	"maps"

	"github.com/df-mc/dragonfly/server"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/timed"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func init() {
	timed.Name = naming.ItemName
}

// EditEntries changes the item and block entries srv sends every player
// (sghook.EditCustomItems, sghook.EditCustomBlocks): our items get their
// components (clientdef), are named by their translation key and the empty
// wooden bucket aims at liquids; our blocks get their client-side components
// (clientdef.BlockEntry). Call it once, before players join.
func EditEntries(srv *server.Server) {
	sghook.EditCustomItems(srv, func(e protocol.ItemEntry) protocol.ItemEntry {
		return sgitem.ClipWoodenBucket(localItemName(clientdef.ItemEntry(e)))
	})
	sghook.EditCustomBlocks(srv, clientdef.BlockEntry)
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
