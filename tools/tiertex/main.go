// tiertex writes the item textures of the machine tiers (tier.go): the three
// alloys (ingots), the four control circuits, the three tier kits and the
// configuration copier. They
// are temporary pictures drawn from a colour for each item.
//
//	go run ./tools/tiertex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func scale(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 {
		if v > 255 {
			return 255
		}
		if v < 0 {
			return 0
		}
		return uint8(v)
	}
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), c.A}
}

func save(path string, im image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		panic(err)
	}
}

// draw paints a shape: every letter has a colour in the palette.
func draw(shape []string, palette map[byte]color.RGBA) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y, row := range shape {
		for x := 0; x < len(row); x++ {
			if c, ok := palette[row[x]]; ok {
				im.Set(x, y, c)
			}
		}
	}
	return im
}

// ingotShape: 'o' outline, 't' top face, 'f' front face, 's' side face, 'g' a
// glint.
var ingotShape = []string{
	"................",
	"................",
	"................",
	"................",
	"....oooooooo....",
	"...ottttgtttto..",
	"..otgtttttttto..",
	".ofttttttttso...",
	".offfffgffffso..",
	".offfffffffso...",
	"..ooooooooooo...",
	"................",
	"................",
	"................",
	"................",
	"................",
}

// circuitShape: 'o' outline, 'b' board, 't' traces, 'c' chip, 'h' chip
// highlight, 'p' pins.
var circuitShape = []string{
	"................",
	".oooooooooooooo.",
	".obbbbbbbbbbbbo.",
	".obtttbbbbtttbo.",
	".obbbtbppbtbbbo.",
	".obbbtpccptbbbo.",
	".obtttpchcpttbo.",
	".obbbbpcccpbbbo.",
	".obbbtpccptbbbo.",
	".obtttbppbtttbo.",
	".obbbtbbbbtbbbo.",
	".obttttbbtttbbo.",
	".obbbbbbbbbbbbo.",
	".oooooooooooooo.",
	"................",
	"................",
}

// kitShape: 'o' outline, 'b' box, 'l' lid, 'r' the band, 'g' a gem on the band.
var kitShape = []string{
	"................",
	"................",
	"...oooooooooo...",
	"..olllllllllllo.",
	"..olllllllllllo.",
	"..orrrrrrrrrrro.",
	"..orrrrggrrrrro.",
	"..orrrrggrrrrro.",
	"..obbbbbbbbbbbo.",
	"..obbbbbbbbbbbo.",
	"..obbbbbbbbbbbo.",
	"..obbbbbbbbbbbo.",
	"...oooooooooo...",
	"................",
	"................",
	"................",
}

// copierShape: 'o' outline, 'b' body, 's' screen, 'l' lit part of the screen,
// 'k' button, 'a' antenna.
var copierShape = []string{
	"..............a.",
	".............aa.",
	"....ooooooo.oa..",
	"...obbbbbbbooo..",
	"...obsssssbbbo..",
	"...obslllsbbbo..",
	"...obslllsbbbo..",
	"...obsssssbbbo..",
	"...obbbbbbbbbo..",
	"...obbkbbkbbbo..",
	"...obbbbbbbbbo..",
	"...obbkbbkbbbo..",
	"...obbbbbbbbbo..",
	"...ooooooooooo..",
	"................",
	"................",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tiertex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/items")
	alloys := map[string]color.RGBA{
		"infused":    {200, 70, 60, 255},
		"reinforced": {70, 170, 210, 255},
		"atomic":     {170, 90, 230, 255},
	}
	for name, col := range alloys {
		im := draw(ingotShape, map[byte]color.RGBA{
			'o': scale(col, 0.4), 't': scale(col, 1.2), 'f': scale(col, 0.95), 's': scale(col, 0.7), 'g': scale(col, 1.7),
		})
		save(filepath.Join(dir, "alloy_"+name+".png"), im)
	}
	circuits := []color.RGBA{{70, 150, 80, 255}, {70, 110, 200, 255}, {200, 80, 70, 255}, {230, 190, 70, 255}}
	for i, col := range circuits {
		im := draw(circuitShape, map[byte]color.RGBA{
			'o': {30, 30, 36, 255}, 'b': scale(col, 0.75), 't': scale(col, 1.35), 'p': {190, 190, 200, 255},
			'c': {40, 40, 48, 255}, 'h': {96, 96, 112, 255},
		})
		save(filepath.Join(dir, fmt.Sprintf("circuit_%d.png", i+1)), im)
	}
	save(filepath.Join(dir, "config_copier.png"), draw(copierShape, map[byte]color.RGBA{
		'o': {30, 36, 40, 255}, 'b': {150, 156, 164, 255}, 's': {40, 70, 60, 255}, 'l': {110, 230, 160, 255},
		'k': {220, 170, 60, 255}, 'a': {200, 200, 210, 255},
	}))
	kits := []color.RGBA{{80, 150, 220, 255}, {236, 190, 60, 255}, {160, 80, 220, 255}}
	for i, col := range kits {
		im := draw(kitShape, map[byte]color.RGBA{
			'o': {40, 34, 30, 255}, 'b': {120, 96, 74, 255}, 'l': {150, 120, 92, 255}, 'r': scale(col, 0.9), 'g': scale(col, 1.6),
		})
		save(filepath.Join(dir, fmt.Sprintf("kit_%d.png", i+1)), im)
	}
}
