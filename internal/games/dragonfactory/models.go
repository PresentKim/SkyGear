package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block/customblock"
)

// Block models of the core, delivery box, drawer and machines (the sieve is
// in sieve.go). Front faces look north (-z); drawers are turned per facing.
// Materials: "*" is the body texture, other names are listed per model.

func alphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

func v(x, y, z float64) [3]float64 { return [3]float64{x, y, z} }

// coreModel: a frame of slabs and corner pillars around a glowing crystal
// ("crystal"). The top stays a full face: the core makes blocks above it.
var coreModel = geoFile(geometry("geometry.dragonfactory_core", []box{
	{from: v(-8, 0, -8), size: v(16, 3, 16)},
	{from: v(-8, 13, -8), size: v(16, 3, 16)},
	{from: v(-8, 3, -8), size: v(3, 10, 3)},
	{from: v(5, 3, -8), size: v(3, 10, 3)},
	{from: v(-8, 3, 5), size: v(3, 10, 3)},
	{from: v(5, 3, 5), size: v(3, 10, 3)},
	{from: v(-3, 3, -3), size: v(6, 10, 6), mat: "crystal"},
	{from: v(-4, 6, -4), size: v(8, 4, 8), mat: "crystal"},
}))

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
	// A body with a drill bit under it.
	KindMiner: geoFile(geometry("geometry.dragonfactory_auto_miner", []box{
		{from: v(-8, 4, -8), size: v(16, 12, 16)},
		{from: v(-5, 2, -5), size: v(10, 2, 10), mat: "accent"},
		{from: v(-3, 0, -3), size: v(6, 2, 6), mat: "accent"},
	})),
	// A body on legs with blades ("accent") around it.
	KindHarvester: geoFile(geometry("geometry.dragonfactory_harvester", []box{
		{from: v(-7, 4, -7), size: v(14, 10, 14)},
		{from: v(-7, 0, -7), size: v(2, 4, 2)},
		{from: v(5, 0, -7), size: v(2, 4, 2)},
		{from: v(-7, 0, 5), size: v(2, 4, 2)},
		{from: v(5, 0, 5), size: v(2, 4, 2)},
		{from: v(-8, 6, -1), size: v(16, 1, 2), mat: "accent"},
		{from: v(-1, 7, -8), size: v(2, 1, 16), mat: "accent"},
		{from: v(-3, 14, -3), size: v(6, 2, 6), mat: "accent"},
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
