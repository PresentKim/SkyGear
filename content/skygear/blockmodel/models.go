// Package blockmodel holds the block definition helpers shared by custom
// blocks: pack materials and cube properties.
package blockmodel

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
)

// Block looks. Models, textures and atlases live in the resource pack
// (assets/resourcepack/skygear); the server only names geometry
// identifiers and atlas keys ("skygear_<file>"). Material "*" is the
// body texture. TestPackHasBlockAssets checks every name exists in the pack.

func AlphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

// SolidMaterial is the material of a natural full block (dust, crushed
// blocks, ores): opaque with ambient occlusion, shaded like vanilla dirt and
// sand. AlphaMaterial turns ambient occlusion off, and such a block looked
// flat and brighter than its neighbours.
func SolidMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.OpaqueRenderMethod())
}

// CubeProperties returns the properties of a plain cube block named name:
// the vanilla full block geometry with the side texture on the four sides
// ("*") and its own top and bottom (textures/blocks/<name>_<side|up|down>.png).
func CubeProperties(name string) customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":    AlphaMaterial("skygear_" + name + "_side"),
		"up":   AlphaMaterial("skygear_" + name + "_up"),
		"down": AlphaMaterial("skygear_" + name + "_down"),
	}}
}

// FrontCubeProperties returns the properties of a plain cube block named
// name with a front: the vanilla full block geometry with the front on the
// south face, the back on the north face, the side texture on east and west
// ("*") and its own top and bottom
// (textures/blocks/<name>_<front|back|side|up|down>.png). The front is south
// because block items are drawn in the inventory with the south face in view.
func FrontCubeProperties(name string) customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":     AlphaMaterial("skygear_" + name + "_side"),
		"south": AlphaMaterial("skygear_" + name + "_front"),
		"north": AlphaMaterial("skygear_" + name + "_back"),
		"up":    AlphaMaterial("skygear_" + name + "_up"),
		"down":  AlphaMaterial("skygear_" + name + "_down"),
	}}
}

// Which tool mines a custom block faster, its BreakInfo.Effective: what it
// is made of decides, as for vanilla blocks (wood the axe; stone, metal and
// machines the pickaxe; sand, dirt and dust the shovel). The tool's speed
// and tier count as for a vanilla block (block.BreakDuration).
//
// The client's definition only carries the bare-hand time
// (minecraft:destructible_by_mining), so it does not finish a faster break
// itself: the server sends the crack speed of the held tool and breaks the
// block once that time is up (player.tickCustomBreak, a Dragonfly patch).
var (
	AxeEffective     = EffectiveFor(item.TypeAxe)
	PickaxeEffective = EffectiveFor(item.TypePickaxe)
	ShovelEffective  = EffectiveFor(item.TypeShovel)
)

// EffectiveFor is a BreakInfo.Effective that is true for tools of type typ,
// a Dragonfly tool or one of ours (tinker tools, the hammer).
func EffectiveFor(typ item.ToolType) func(item.Tool) bool {
	return func(t item.Tool) bool { return t.ToolType() == typ }
}

// WoodEffective is the axe for a wooden block, else the pickaxe.
func WoodEffective(wood bool) func(item.Tool) bool {
	if wood {
		return AxeEffective
	}
	return PickaxeEffective
}
