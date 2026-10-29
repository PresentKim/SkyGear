// Command essencetex writes the item textures of the research essences
// (content/skygear/researchessence.go): a glass bottle filled
// with a liquid of the colour of each essence. They are temporary pictures,
// drawn from the colour of each essence.
//
//	go run ./tools/essencetex assets/resourcepack/skygear
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

// essences are the colours of the research essences, by kind.
var essences = map[string]color.RGBA{
	"engineering": {214, 52, 52, 255},
	"alchemy":     {236, 140, 36, 255},
	"farming":     {232, 202, 48, 255},
	"hunting":     {64, 172, 72, 255},
	"magic":       {156, 76, 214, 255},
}

// shape is a bottle: 'o' outline, 'c' cork, 'g' glass, 'l' liquid, 'h' the
// shine on the liquid, 'd' the dark side of the liquid.
var shape = []string{
	"................",
	"......cccc......",
	"......cccc......",
	".....oggggo.....",
	".....oggggo.....",
	"....oggggggo....",
	"...ogllllllgo...",
	"..oglhhllllldo..",
	"..oglhllllllldo.",
	"..ogllllllllldo.",
	"..ogllllllllldo.",
	"..oglllllllddo..",
	"...ogllllddddo..",
	"....oggggggo....",
	".....oooooo.....",
	"................",
}

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

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: essencetex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/items")
	for kind, col := range essences {
		im := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y, row := range shape {
			for x, ch := range []rune(row) {
				switch ch {
				case 'o':
					im.Set(x, y, color.RGBA{70, 78, 90, 255})
				case 'c':
					im.Set(x, y, color.RGBA{150, 104, 62, 255})
				case 'g':
					im.Set(x, y, color.RGBA{196, 220, 232, 150})
				case 'l':
					im.Set(x, y, col)
				case 'h':
					im.Set(x, y, scale(col, 1.45))
				case 'd':
					im.Set(x, y, scale(col, 0.72))
				}
			}
		}
		imgio.Save(filepath.Join(dir, "research_essence_"+kind+".png"), im)
	}
}
