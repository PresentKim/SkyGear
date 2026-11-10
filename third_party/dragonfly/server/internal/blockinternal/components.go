package blockinternal

import (
	"maps"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Components returns all the components for the custom block, including permutations and properties.
func Components(identifier string, b world.CustomBlock, blockID int32) map[string]any {
	components := componentsFromProperties(b.Properties())
	builder := NewComponentBuilder(identifier, components, blockID)
	if emitter, ok := b.(block.LightEmitter); ok {
		builder.AddComponent("minecraft:light_emission", map[string]any{
			"emission": int32(emitter.LightEmissionLevel()),
		})
	}
	if diffuser, ok := b.(block.LightDiffuser); ok {
		builder.AddComponent("minecraft:light_dampening", map[string]any{
			"lightLevel": int32(diffuser.LightDiffusionLevel()),
		})
	}
	if _, ok := b.(block.Breakable); ok {
		// The component's value is the seconds the client takes to destroy the block.
		// Bedrock has no per-tool speed component, so the bare-handed
		// duration is the only one a static definition can carry.
		seconds := block.BreakDuration(b, item.Stack{}, block.BreakContext{}).Seconds()
		builder.AddComponent("minecraft:destructible_by_mining", map[string]any{"value": float32(seconds)})
		// skygear patch: the tags of the tools that mine the block faster, as
		// vanilla blocks carry them, so a digger that names the tags (vanilla
		// tools, item.TagDigger) mines it at its speed.
		for _, tag := range toolTags(b.(block.Breakable).BreakInfo()) {
			builder.AddComponent("tag:"+tag, map[string]any{})
		}
	}
	if frictional, ok := b.(block.Frictional); ok {
		builder.AddComponent("minecraft:friction", map[string]any{"value": float32(frictional.Friction())})
	}
	if flammable, ok := b.(block.Flammable); ok {
		info := flammable.FlammabilityInfo()
		builder.AddComponent("minecraft:flammable", map[string]any{
			"flame_odds": int32(info.Encouragement),
			"burn_odds":  int32(info.Flammability),
		})
	}
	if _, ok := b.(block.Activatable); ok {
		// skygear patch: marks the block as interactable, so the client sends the click to
		// the server instead of predicting the held block being placed.
		builder.AddComponent("minecraft:custom_components", map[string]any{
			"hasPlayerInteract": uint8(1),
			"hasPlayerPlacing":  uint8(0),
			"isV1Component":     uint8(1),
		})
	}
	if permutable, ok := b.(block.Permutable); ok {
		// skygear patch: the properties in the order of their names, not of the
		// map. The client numbers a block's states as every combination of the
		// values, the property first by name changing slowest (the order of the
		// vanilla palette), and the server numbers them by registration; sending
		// them sorted keeps the definition in that same order every time.
		states := permutable.States()
		for _, name := range slices.Sorted(maps.Keys(states)) {
			builder.AddProperty(name, states[name])
		}
		for _, permutation := range permutable.Permutations() {
			builder.AddPermutation(permutation.Condition, componentsFromProperties(permutation.Properties))
		}
	}
	if item, ok := b.(world.CustomItem); ok {
		builder.SetMenuCategory(item.Category())
	}
	return builder.Construct()
}

// componentsFromProperties builds a base components map that includes all the common data between a regular block and
// a custom permutation.
func componentsFromProperties(props customblock.Properties) map[string]any {
	components := make(map[string]any)
	if props.NoCollision {
		// skygear patch: collision_box false
		components["minecraft:collision_box"] = map[string]any{"enabled": false, "boxes": []map[string]any{zeroBox()}}
	} else if len(props.CollisionBoxes) > 0 {
		components["minecraft:collision_box"] = collisionBoxComponent(props.CollisionBoxes...)
	} else if props.CollisionBox != (cube.BBox{}) {
		components["minecraft:collision_box"] = collisionBoxComponent(props.CollisionBox)
	}
	if props.NoSelection {
		// skygear patch: selection_box false
		components["minecraft:selection_box"] = map[string]any{"enabled": false, "origin": []float32{0, 0, 0}, "size": []float32{0, 0, 0}}
	} else if props.SelectionBox != (cube.BBox{}) {
		components["minecraft:selection_box"] = selectionBoxComponent(props.SelectionBox)
	}
	if props.Geometry != "" {
		components["minecraft:geometry"] = map[string]any{"identifier": props.Geometry}
	} else if props.Cube {
		components["minecraft:geometry"] = map[string]any{"identifier": "minecraft:geometry.full_block"}
	}
	if geometry, ok := components["minecraft:geometry"].(map[string]any); ok && len(props.BoneVisibility) > 0 {
		// skygear patch: bone_visibility, each bone a Molang expression ("true" and "false" are
		// constants) the client evaluates per block state.
		visibility := make(map[string]any, len(props.BoneVisibility))
		for bone, expression := range props.BoneVisibility {
			visibility[bone] = expression
		}
		geometry["bone_visibility"] = visibility
	}
	if props.MapColour != "" {
		components["minecraft:map_color"] = map[string]any{"value": props.MapColour}
	}
	if props.Textures != nil {
		materials := map[string]any{}
		for target, material := range props.Textures {
			materials[target] = material.Encode()
		}
		components["minecraft:material_instances"] = map[string]any{
			"mappings":  map[string]any{},
			"materials": materials,
		}
	}
	transformation := make(map[string]any)
	if props.Rotation != (cube.Pos{}) {
		transformation["RX"] = int32(props.Rotation.X())
		transformation["RY"] = int32(props.Rotation.Y())
		transformation["RZ"] = int32(props.Rotation.Z())
	}
	if props.Translation != (mgl64.Vec3{}) {
		transformation["TX"] = float32(props.Translation.X())
		transformation["TY"] = float32(props.Translation.Y())
		transformation["TZ"] = float32(props.Translation.Z())
	}
	if props.Scale != (mgl64.Vec3{}) {
		transformation["SX"] = float32(props.Scale.X())
		transformation["SY"] = float32(props.Scale.Y())
		transformation["SZ"] = float32(props.Scale.Z())
	} else if len(transformation) > 0 {
		transformation["SX"] = float32(1.0)
		transformation["SY"] = float32(1.0)
		transformation["SZ"] = float32(1.0)
	}
	if len(transformation) > 0 {
		components["minecraft:transformation"] = transformation
	}
	return components
}

// collisionBoxComponent returns the component data for a collision box, using absolute min/max coordinates in pixels.
func collisionBoxComponent(boxes ...cube.BBox) map[string]any {
	list := make([]map[string]any, 0, len(boxes))
	for _, box := range boxes {
		min, max := box.Min(), box.Max()
		list = append(list, map[string]any{
			"minX": float32(min.X() * 16),
			"minY": float32(min.Y() * 16),
			"minZ": float32(min.Z() * 16),
			"maxX": float32(max.X() * 16),
			"maxY": float32(max.Y() * 16),
			"maxZ": float32(max.Z() * 16),
		})
	}
	return map[string]any{
		"enabled": true,
		"boxes":   list,
	}
}

// zeroBox is an empty box of a collision box component (skygear patch).
func zeroBox() map[string]any {
	return map[string]any{"minX": float32(0), "minY": float32(0), "minZ": float32(0), "maxX": float32(0), "maxY": float32(0), "maxZ": float32(0)}
}

// selectionBoxComponent returns the component data for a selection box, translating coordinates to the origin/size
// format the client expects.
func selectionBoxComponent(box cube.BBox) map[string]any {
	min, max := box.Min(), box.Max()
	originX, originY, originZ := min.X()*16, min.Y()*16, min.Z()*16
	sizeX, sizeY, sizeZ := (max.X()-min.X())*16, (max.Y()-min.Y())*16, (max.Z()-min.Z())*16
	return map[string]any{
		"enabled": true,
		"origin":  []float32{float32(originX) - 8, float32(originY), float32(originZ) - 8},
		"size":    []float32{float32(sizeX), float32(sizeY), float32(sizeZ)},
	}
}

// toolTags are the vanilla block tags of the tools that mine a block with
// info faster: minecraft:is_<tool>_item_destructible for each tool type its
// Effective takes, and for a pickaxe the tier it needs to drop anything
// (stone_pick_diggable, iron_pick_diggable, diamond_pick_diggable).
// (skygear patch)
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
