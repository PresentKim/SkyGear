package dragonfactory

import (
	"image"

	"github.com/df-mc/dragonfly/server/block/customblock"
)

// Block models of the core, delivery box, drawer and machines (the sieve is
// in sieve.go). Cores, the auto miner and the harvester are plain cubes
// with a texture per face (cubeGeometry). Front faces look north (-z); drawers are turned per facing.
// Materials: "*" is the body texture, other names are listed per model.

func alphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

func v(x, y, z float64) [3]float64 { return [3]float64{x, y, z} }

// cubeFaces are the faces of cube blocks (no modelling).
var cubeFaces = []string{"north", "south", "east", "west", "up", "down"}

// cubeParts are the textures of cube blocks: the four sides share one.
// Textures: textures/blocks/<name>_<side|up|down>.png.
var cubeParts = []string{"side", "up", "down"}

// cubeGeometry is one full cube whose faces use the materials named after
// them.
func cubeGeometry(id string) []byte {
	uv := map[string]any{}
	for _, f := range cubeFaces {
		uv[f] = faceUV(16, 16, f)
	}
	return geoFile(map[string]any{
		"description": map[string]any{
			"identifier": id, "texture_width": 16, "texture_height": 16,
			"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
		},
		"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": []any{map[string]any{
			"origin": v(-8, 0, -8), "size": v(16, 16, 16), "uv": uv,
		}}}},
	})
}

// cubeTextures returns the face textures of a cube block named name.
func cubeTextures(name string) map[string]image.Image {
	out := map[string]image.Image{}
	for _, part := range cubeParts {
		out["dragonfactory_"+name+"_"+part] = blockTexture(name + "_" + part)
	}
	return out
}

// cubeProperties returns the properties of a cube block: geometry id and
// the side material on "*", with its own top and bottom.
func cubeProperties(id, name string) customblock.Properties {
	textures := map[string]customblock.Material{
		"*":    alphaMaterial("dragonfactory_" + name + "_side"),
		"up":   alphaMaterial("dragonfactory_" + name + "_up"),
		"down": alphaMaterial("dragonfactory_" + name + "_down"),
	}
	return customblock.Properties{Geometry: id, Textures: textures}
}

// coreModel: all core tiers are cubes with their own face textures.
var coreModel = cubeGeometry("geometry.dragonfactory_core")

// deliveryModel: a crate with a gold-banded lid ("lid").
var deliveryModel = geoFile(geometry("geometry.dragonfactory_delivery_box", []box{
	{from: v(-7, 0, -7), size: v(14, 11, 14)},
	{from: v(-8, 11, -8), size: v(16, 3, 16), mat: "lid"},
	{from: v(-2, 8, -8), size: v(4, 3, 1), mat: "lid"},
}))

// drawerModel: a plain full cube; the north face (the front, turned by
// the facing permutations) is the "front" material, the rest "*".
var drawerModel = geoFile(map[string]any{
	"description": map[string]any{
		"identifier": "geometry.dragonfactory_drawer", "texture_width": 16, "texture_height": 16,
		"visible_bounds_width": 2, "visible_bounds_height": 2, "visible_bounds_offset": []float64{0, 0.5, 0},
	},
	"bones": []any{map[string]any{"name": "root", "pivot": []float64{0, 0, 0}, "cubes": []any{map[string]any{
		"origin": v(-8, 0, -8), "size": v(16, 16, 16),
		"uv": map[string]any{
			"north": faceUV(16, 16, "front"), "south": faceUV(16, 16, ""),
			"east": faceUV(16, 16, ""), "west": faceUV(16, 16, ""),
			"up": faceUV(16, 16, ""), "down": faceUV(16, 16, ""),
		},
	}}}},
})

// machineModels: per kind, a body ("*") and moving parts ("accent").
var machineModels = map[string][]byte{
	// A funnel on a base with two rollers inside.
	KindCrusher: geoFile(geometry("geometry.dragonfactory_crusher", []box{
		{from: v(-8, 0, -8), size: v(16, 10, 16)},
		{from: v(-8, 10, -8), size: v(16, 6, 2)},
		{from: v(-8, 10, 6), size: v(16, 6, 2)},
		{from: v(-8, 10, -6), size: v(2, 6, 12)},
		{from: v(6, 10, -6), size: v(2, 6, 12)},
		{from: v(-6, 10, -4), size: v(12, 3, 3), mat: "accent"},
		{from: v(-6, 10, 1), size: v(12, 3, 3), mat: "accent"},
	})),
	// A chest whose lid ("accent") has a coin slot.
	KindSellBox: geoFile(geometry("geometry.dragonfactory_sell_box", []box{
		{from: v(-7, 0, -7), size: v(14, 12, 14)},
		{from: v(-8, 12, -8), size: v(16, 3, 7), mat: "accent"},
		{from: v(-8, 12, 1), size: v(16, 3, 7), mat: "accent"},
		{from: v(-8, 12, -1), size: v(5, 3, 2), mat: "accent"},
		{from: v(3, 12, -1), size: v(5, 3, 2), mat: "accent"},
	})),
}

func machineProperties(kind string) customblock.Properties {
	return customblock.Properties{
		Geometry: "geometry.dragonfactory_" + kind,
		Textures: map[string]customblock.Material{
			"*":      alphaMaterial("dragonfactory_" + kind),
			"accent": alphaMaterial("dragonfactory_" + kind + "_accent"),
		},
	}
}
