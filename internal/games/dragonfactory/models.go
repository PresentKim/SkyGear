package dragonfactory

import "github.com/df-mc/dragonfly/server/block/customblock"

// Block looks. Models (models/blocks/*.geo.json), textures and their atlas
// (textures/terrain_texture.json, item_texture.json) live in the resource
// pack assets/resourcepack/dragonfactory; the server only names them here:
// geometry identifiers and material textures (atlas keys
// "dragonfactory_<file>" for textures/blocks/<file>.png). Fronts of the
// fixed cubes look south, which is in view in the inventory; drawers are
// turned per facing. Materials: "*" is the body
// texture, other names are listed per block. TestPackHasBlockAssets checks
// every name exists in the pack.

func alphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

// solidMaterial is the material of a natural full block (dust, crushed
// blocks, ores): opaque with ambient occlusion, shaded like vanilla dirt and
// sand. alphaMaterial turns ambient occlusion off, and such a block looked
// flat and brighter than its neighbours.
func solidMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.OpaqueRenderMethod())
}

// cubeProperties returns the properties of a plain cube block named name:
// the vanilla full block geometry with the side texture on the four sides
// ("*") and its own top and bottom (textures/blocks/<name>_<side|up|down>.png).
func cubeProperties(name string) customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":    alphaMaterial("dragonfactory_" + name + "_side"),
		"up":   alphaMaterial("dragonfactory_" + name + "_up"),
		"down": alphaMaterial("dragonfactory_" + name + "_down"),
	}}
}

// frontCubeProperties returns the properties of a plain cube block named
// name with a front: the vanilla full block geometry with the front on the
// south face, the back on the north face, the side texture on east and west
// ("*") and its own top and bottom
// (textures/blocks/<name>_<front|back|side|up|down>.png). The front is south
// because block items are drawn in the inventory with the south face in view.
func frontCubeProperties(name string) customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":     alphaMaterial("dragonfactory_" + name + "_side"),
		"south": alphaMaterial("dragonfactory_" + name + "_front"),
		"north": alphaMaterial("dragonfactory_" + name + "_back"),
		"up":    alphaMaterial("dragonfactory_" + name + "_up"),
		"down":  alphaMaterial("dragonfactory_" + name + "_down"),
	}}
}
