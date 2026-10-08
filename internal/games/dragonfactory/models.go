package dragonfactory

import (
	"image"
	"image/color"

	"github.com/df-mc/dragonfly/server/block/customblock"
)

// Block models of the core, delivery box, drawer and machines (the sieve is
// in sieve.go). Front faces look north (-z); drawers are turned per facing.
// Materials: "*" is the body texture, other names are listed per model.

func alphaMaterial(texture string) customblock.Material {
	return customblock.NewMaterial(texture, customblock.AlphaTestRenderMethod())
}

// solidTexture draws a 16×16 texture of c with a light edge and speckles.
func solidTexture(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			p := c
			switch {
			case x == 0 || y == 0:
				p = shade(c, 1.25)
			case x == 15 || y == 15:
				p = shade(c, 0.7)
			case (x*7+y*3)%11 == 0:
				p = shade(c, 1.15)
			}
			img.Set(x, y, p)
		}
	}
	return img
}

// centreColour returns the colour at the centre of a texture (the gem of
// frameTexture).
func centreColour(img image.Image) color.RGBA {
	r, g, b, a := img.At(8, 8).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
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

var coreFrameTexture = solidTexture(rgb(70, 60, 75))

// deliveryModel: a crate with a gold-banded lid ("lid").
var deliveryModel = geoFile(geometry("geometry.dragonfactory_delivery_box", []box{
	{from: v(-7, 0, -7), size: v(14, 11, 14)},
	{from: v(-8, 11, -8), size: v(16, 3, 16), mat: "lid"},
	{from: v(-2, 8, -8), size: v(4, 3, 1), mat: "lid"},
}))

var deliveryLidTexture = solidTexture(rgb(220, 180, 60))

// drawerModel: a wooden case with a recessed front panel ("front") and a
// handle ("handle") at the top centre, under the lock mark.
var drawerModel = geoFile(geometry("geometry.dragonfactory_drawer", []box{
	{from: v(-8, 0, -7), size: v(16, 16, 15)},
	{from: v(-8, 0, -8), size: v(16, 2, 1)},
	{from: v(-8, 14, -8), size: v(16, 2, 1)},
	{from: v(-8, 2, -8), size: v(2, 12, 1)},
	{from: v(6, 2, -8), size: v(2, 12, 1)},
	{from: v(-6, 2, -7.5), size: v(12, 12, 0.5), mat: "front"},
	{from: v(-2, 12, -8), size: v(4, 1, 0.5), mat: "handle"},
}))

var drawerHandleTexture = solidTexture(rgb(60, 60, 65))

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
	// A chest whose lid ("accent") has a coin slot.
	KindSellBox: geoFile(geometry("geometry.dragonfactory_sell_box", []box{
		{from: v(-7, 0, -7), size: v(14, 12, 14)},
		{from: v(-8, 12, -8), size: v(16, 3, 7), mat: "accent"},
		{from: v(-8, 12, 1), size: v(16, 3, 7), mat: "accent"},
		{from: v(-8, 12, -1), size: v(5, 3, 2), mat: "accent"},
		{from: v(3, 12, -1), size: v(5, 3, 2), mat: "accent"},
	})),
}

func machineAccentTexture(k *machineKind) image.Image { return solidTexture(centreColour(k.texture)) }

func machineProperties(kind string) customblock.Properties {
	return customblock.Properties{
		Geometry: "geometry.dragonfactory_" + kind,
		Textures: map[string]customblock.Material{
			"*":      alphaMaterial("dragonfactory_" + kind),
			"accent": alphaMaterial("dragonfactory_" + kind + "_accent"),
		},
	}
}
