package dragonfactory

import "github.com/df-mc/dragonfly/server/block/customblock"

// Block looks. Models (models/blocks/*.geo.json), textures and their atlas
// (textures/terrain_texture.json, item_texture.json) live in the resource
// pack assets/resourcepack/dragonfactory; the server only names them here:
// geometry identifiers and material textures (atlas keys
// "dragonfactory_<file>" for textures/blocks/<file>.png). Front faces look
// north (-z); drawers are turned per facing. Materials: "*" is the body
// texture, other names are listed per block. TestPackHasBlockAssets checks
// every name exists in the pack.

func alphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
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

// machineProperties: per kind, a body ("*") and moving parts ("accent")
// (crusher: a funnel with two rollers; sell box: a chest whose lid has a
// coin slot).
func machineProperties(kind string) customblock.Properties {
	return customblock.Properties{
		Geometry: "geometry.dragonfactory_" + kind,
		Textures: map[string]customblock.Material{
			"*":      alphaMaterial("dragonfactory_" + kind),
			"accent": alphaMaterial("dragonfactory_" + kind + "_accent"),
		},
	}
}
