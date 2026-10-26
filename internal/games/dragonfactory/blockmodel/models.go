// Package blockmodel holds the block definition helpers shared by custom
// blocks: pack materials and cube properties.
package blockmodel

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
)

// Block looks. Models (models/blocks/*.geo.json), textures and their atlas
// (textures/terrain_texture.json, item_texture.json) live in the resource
// pack assets/resourcepack/dragonfactory; the server only names them here:
// geometry identifiers and material textures (atlas keys
// "dragonfactory_<file>" for textures/blocks/<file>.png). Fronts of the
// fixed cubes look south, which is in view in the inventory; drawers are
// turned per facing. Materials: "*" is the body
// texture, other names are listed per block. TestPackHasBlockAssets checks
// every name exists in the pack.

func AlphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

// SolidMaterial is the material of a natural full block (dust, crushed
// blocks, ores): opaque with ambient occlusion, shaded like vanilla dirt and
// sand. alphaMaterial turns ambient occlusion off, and such a block looked
// flat and brighter than its neighbours.
func SolidMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.OpaqueRenderMethod())
}

// CubeProperties returns the properties of a plain cube block named name:
// the vanilla full block geometry with the side texture on the four sides
// ("*") and its own top and bottom (textures/blocks/<name>_<side|up|down>.png).
func CubeProperties(name string) customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":    AlphaMaterial("dragonfactory_" + name + "_side"),
		"up":   AlphaMaterial("dragonfactory_" + name + "_up"),
		"down": AlphaMaterial("dragonfactory_" + name + "_down"),
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
		"*":     AlphaMaterial("dragonfactory_" + name + "_side"),
		"south": AlphaMaterial("dragonfactory_" + name + "_front"),
		"north": AlphaMaterial("dragonfactory_" + name + "_back"),
		"up":    AlphaMaterial("dragonfactory_" + name + "_up"),
		"down":  AlphaMaterial("dragonfactory_" + name + "_down"),
	}}
}

// CustomBlockEffective makes no tool faster on custom blocks. The client
// takes its break time from the block definition, which only carries the
// bare-hand time; if the server used a faster tool time, the crack would
// finish while the client still mines and the block would never break.
func CustomBlockEffective(item.Tool) bool { return false }
