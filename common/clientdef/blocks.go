package clientdef

import (
	"cmp"
	"maps"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// BlockEntry returns e with the client-side parts of the registered custom
// block it describes that Dragonfly's own entry does not carry:
//
//   - for a block.Breakable, a "tag:<tag>" component for each tool tag of its
//     BreakInfo (minecraft:is_<tool>_item_destructible for the tools that
//     mine it faster and, for a pickaxe block, the tier it needs to drop
//     anything), so a digger that names tags (TagDigger, vanilla tools) mines
//     it at its speed;
//   - for a block.Activatable, minecraft:custom_components marking the block
//     as interactable, so the client sends a click to the server instead of
//     predicting that the held block is placed;
//   - for a block that implements ExtraProperties, its collision boxes
//     (minecraft:collision_box with a list of boxes, replacing a single one)
//     and the visibility of its geometry's bones (bone_visibility in
//     minecraft:geometry), in its base components and in the components of the
//     permutation with the same condition;
//   - the properties in the order of their names, not of the map Dragonfly
//     iterates. The client numbers a block's states as every combination of
//     the values, the property first by name changing slowest (the order of
//     the vanilla palette), and the server numbers them by registration;
//     sorted, the definition keeps that order every time.
//
// An entry of no registered custom block (a data-driven one) is returned
// unchanged. The maps and the property list of e are not modified. It panics,
// naming the block, if a shape it needs is missing.
func BlockEntry(e protocol.BlockEntry) protocol.BlockEntry {
	e.Properties = maps.Clone(e.Properties)
	if comps, ok := e.Properties["components"].(map[string]any); ok {
		e.Properties["components"] = maps.Clone(comps)
	}
	if b, ok := world.CustomBlocks()[e.Name]; ok {
		if br, ok := b.(block.Breakable); ok {
			if tags := toolTags(br.BreakInfo()); len(tags) > 0 {
				comps := blockComponents(e)
				for _, tag := range tags {
					comps["tag:"+tag] = map[string]any{}
				}
			}
		}
		if x, ok := b.(ExtraProperties); ok {
			base, perms := x.ClientExtras()
			applyExtras(e, blockComponents(e), base)
			applyPermutationExtras(&e, perms)
		}
		if _, ok := b.(block.Activatable); ok {
			blockComponents(e)["minecraft:custom_components"] = map[string]any{
				"hasPlayerInteract": uint8(1),
				"hasPlayerPlacing":  uint8(0),
				"isV1Component":     uint8(1),
			}
		}
	}
	if props, ok := e.Properties["properties"]; ok {
		e.Properties["properties"] = sortedProperties(e.Name, props)
	}
	return e
}

// Extras are the parts of a custom block's client definition that
// customblock.Properties cannot hold.
type Extras struct {
	// CollisionBoxes are the boxes the player collides with, in the client's
	// block space (0 to 1 over the block), up to 16. Empty leaves the
	// collision box as Properties sets it.
	CollisionBoxes []cube.BBox
	// BoneVisibility maps bones of the block's geometry to whether they are
	// drawn: "true", "false" or a Molang expression. A bone not listed is
	// drawn. It needs a geometry.
	BoneVisibility map[string]string
}

// ExtraProperties is implemented by a custom block that sends Extras: for its
// base definition and for each permutation, by the permutation's condition.
type ExtraProperties interface {
	ClientExtras() (base Extras, permutations map[string]Extras)
}

// applyExtras puts x into comps, a components map BlockEntry owns.
func applyExtras(e protocol.BlockEntry, comps map[string]any, x Extras) {
	if len(x.CollisionBoxes) > 0 {
		list := make([]map[string]any, 0, len(x.CollisionBoxes))
		for _, box := range x.CollisionBoxes {
			lo, hi := box.Min(), box.Max()
			list = append(list, map[string]any{
				"minX": float32(lo.X() * 16),
				"minY": float32(lo.Y() * 16),
				"minZ": float32(lo.Z() * 16),
				"maxX": float32(hi.X() * 16),
				"maxY": float32(hi.Y() * 16),
				"maxZ": float32(hi.Z() * 16),
			})
		}
		comps["minecraft:collision_box"] = map[string]any{"enabled": true, "boxes": list}
	}
	if len(x.BoneVisibility) > 0 {
		geo, ok := comps["minecraft:geometry"].(map[string]any)
		if !ok {
			panic("clientdef: block entry " + e.Name + " has bone visibility but no geometry")
		}
		geo = maps.Clone(geo)
		visibility := make(map[string]any, len(x.BoneVisibility))
		for bone, expression := range x.BoneVisibility {
			visibility[bone] = expression
		}
		geo["bone_visibility"] = visibility
		comps["minecraft:geometry"] = geo
	}
}

// applyPermutationExtras puts the extras of each condition into the
// components of the permutation of e with that condition, on copies.
func applyPermutationExtras(e *protocol.BlockEntry, perms map[string]Extras) {
	if len(perms) == 0 {
		return
	}
	list, ok := e.Properties["permutations"].([]map[string]any)
	if !ok {
		panic("clientdef: block entry " + e.Name + " has extras for permutations but none of the expected type")
	}
	list = slices.Clone(list)
	done := 0
	for i, p := range list {
		cond, _ := p["condition"].(string)
		x, ok := perms[cond]
		if !ok {
			continue
		}
		comps, ok := p["components"].(map[string]any)
		if !ok {
			panic("clientdef: permutation " + cond + " of block entry " + e.Name + " has no components")
		}
		p = maps.Clone(p)
		comps = maps.Clone(comps)
		applyExtras(*e, comps, x)
		p["components"] = comps
		list[i] = p
		done++
	}
	if done != len(perms) {
		panic("clientdef: block entry " + e.Name + " has extras for a permutation condition it does not have")
	}
	e.Properties["permutations"] = list
}

// blockComponents returns the components map of e, which BlockEntry has
// already cloned.
func blockComponents(e protocol.BlockEntry) map[string]any {
	comps, ok := e.Properties["components"].(map[string]any)
	if !ok {
		panic("clientdef: block entry " + e.Name + " has no components")
	}
	return comps
}

// sortedProperties returns a copy of the property list props sorted by name.
func sortedProperties(block string, props any) any {
	name := func(p map[string]any) string {
		s, ok := p["name"].(string)
		if !ok {
			panic("clientdef: block entry " + block + " has a property without a name")
		}
		return s
	}
	switch l := props.(type) {
	case []map[string]any:
		l = slices.Clone(l)
		slices.SortStableFunc(l, func(a, b map[string]any) int { return cmp.Compare(name(a), name(b)) })
		return l
	case []any:
		l = slices.Clone(l)
		slices.SortStableFunc(l, func(a, b any) int {
			am, aok := a.(map[string]any)
			bm, bok := b.(map[string]any)
			if !aok || !bok {
				panic("clientdef: block entry " + block + " has a property that is no compound")
			}
			return cmp.Compare(name(am), name(bm))
		})
		return l
	}
	panic("clientdef: block entry " + block + " has properties of an unexpected type")
}

// toolTags are the vanilla block tags of the tools that mine a block with
// info faster: minecraft:is_<tool>_item_destructible for each tool type its
// Effective takes, and for a pickaxe the tier it needs to drop anything
// (stone_pick_diggable, iron_pick_diggable, diamond_pick_diggable).
func toolTags(info block.BreakInfo) []string {
	if info.Effective == nil {
		return nil
	}
	var tags []string
	for _, t := range []struct {
		tag  string
		tool item.Tool
	}{
		{"minecraft:is_pickaxe_item_destructible", item.Pickaxe{Tier: item.ToolTierDiamond}},
		{"minecraft:is_axe_item_destructible", item.Axe{Tier: item.ToolTierDiamond}},
		{"minecraft:is_shovel_item_destructible", item.Shovel{Tier: item.ToolTierDiamond}},
		{"minecraft:is_hoe_item_destructible", item.Hoe{Tier: item.ToolTierDiamond}},
		{"minecraft:is_sword_item_destructible", item.Sword{Tier: item.ToolTierDiamond}},
		{"minecraft:is_shears_item_destructible", item.Shears{}},
	} {
		if info.Effective(t.tool) {
			tags = append(tags, t.tag)
		}
	}
	if info.Harvestable == nil || info.Harvestable(item.ToolNone{}) || !info.Effective(item.Pickaxe{Tier: item.ToolTierDiamond}) {
		return tags
	}
	for _, t := range []struct {
		tag  string
		tier item.ToolTier
	}{
		{"", item.ToolTierWood},
		{"stone_pick_diggable", item.ToolTierStone},
		{"iron_pick_diggable", item.ToolTierIron},
		{"diamond_pick_diggable", item.ToolTierDiamond},
	} {
		if info.Harvestable(item.Pickaxe{Tier: t.tier}) {
			if t.tag != "" {
				tags = append(tags, t.tag)
			}
			break
		}
	}
	return tags
}
