package clientdef

import (
	"maps"
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Digger is a custom item that tells clients how fast it mines certain
// blocks (the minecraft:digger item component). Without it, clients mine with
// a custom item at bare-hand speed whatever the server computes.
type Digger interface {
	// DestroySpeeds maps block names (e.g. "minecraft:stone") to the mining
	// speed of the item on them (a tool tier efficiency, e.g. 4 for stone).
	DestroySpeeds() map[string]int
}

// TagDigger is a custom item that tells clients how fast it mines blocks by
// their tags (the minecraft:digger item component's block tags), so one entry
// covers every vanilla and custom block that carries the tag.
type TagDigger interface {
	// DestroySpeedsByTag are the item's mining speeds, each on the blocks a
	// Molang tag query matches (e.g.
	// "query.any_tag('minecraft:is_pickaxe_item_destructible')").
	DestroySpeedsByTag() []TagSpeed
}

// TagSpeed is a mining speed on the blocks a Molang tag query matches.
type TagSpeed struct {
	Tags  string
	Speed int
}

// ItemEntry returns e with the components of the registered custom item it
// describes: minecraft:use_modifiers for an item.Usable that is no
// item.Consumable (the client sends a use on the air only for an item it
// knows to be usable), minecraft:digger for a Digger or TagDigger and
// minecraft:fuel (seconds) for an item.Fuel that burns. An entry of no custom
// item, or of an item with none of these, is returned unchanged. The maps of e
// are not modified. It panics if a component is due and e has no components.
func ItemEntry(e protocol.ItemEntry) protocol.ItemEntry {
	var it world.CustomItem
	for _, c := range world.CustomItems() {
		if name, _ := c.EncodeItem(); name == e.Name {
			it = c
			break
		}
	}
	if it == nil {
		return e
	}
	add := map[string]any{}
	if _, ok := it.(item.Usable); ok {
		if _, food := it.(item.Consumable); !food {
			add["minecraft:use_modifiers"] = map[string]any{
				"use_duration":      float32(0.05),
				"movement_modifier": float32(1),
			}
		}
	}
	if d := destroySpeeds(it); d != nil {
		add["minecraft:digger"] = map[string]any{
			"use_efficiency": true,
			"destroy_speeds": d,
		}
	}
	if x, ok := it.(item.Fuel); ok && x.FuelInfo().Duration > 0 {
		add["minecraft:fuel"] = map[string]any{
			"duration": float32(x.FuelInfo().Duration.Seconds()),
		}
	}
	if len(add) == 0 {
		return e
	}
	comps, ok := e.Data["components"].(map[string]any)
	if !ok {
		panic("clientdef: item entry " + e.Name + " has no components")
	}
	comps = maps.Clone(comps)
	maps.Copy(comps, add)
	e.Data = maps.Clone(e.Data)
	e.Data["components"] = comps
	return e
}

// destroySpeeds lists the destroy speeds of it: its Digger blocks by sorted
// name, then its TagDigger entries in order. It is nil if there are none.
func destroySpeeds(it world.CustomItem) []any {
	var destroy []any
	if x, ok := it.(Digger); ok {
		speeds := x.DestroySpeeds()
		names := make([]string, 0, len(speeds))
		for name := range speeds {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			destroy = append(destroy, map[string]any{
				"block": map[string]any{"name": name},
				"speed": int32(speeds[name]),
			})
		}
	}
	if x, ok := it.(TagDigger); ok {
		for _, s := range x.DestroySpeedsByTag() {
			destroy = append(destroy, map[string]any{
				"block": map[string]any{"tags": s.Tags},
				"speed": int32(s.Speed),
			})
		}
	}
	return destroy
}
