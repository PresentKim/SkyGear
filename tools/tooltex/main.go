// tooltex writes the item textures of the tinkered tools (tinker.go): for each
// kind, head metal and handle one 16×16 picture drawn from a bitmap of the kind
// and the colours of the metal and the handle. Temporary pictures.
//
//	go run ./tools/tooltex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// shapes: 'H' head, 'h' head light, 'd' head dark, 'S' handle, 's' handle dark.
var shapes = map[string][]string{
	"sword": {
		"................",
		"..............hh",
		".............hHd",
		"............hHd.",
		"...........hHd..",
		"..........hHd...",
		".........hHd....",
		"..S.....hHd.....",
		"..SS...hHd......",
		"...SS.hHd.......",
		"....SSHd........",
		".....SSS........",
		"....SSs.........",
		"...SSs..........",
		"..SSs...........",
		"..ss............",
	},
	"pickaxe": {
		"................",
		"....hhhhhhh.....",
		"...hHHHHHHHd....",
		"..hHd....SHHd...",
		"..hd....SSd.Hd..",
		"..d....SSs..dd..",
		"......SSs.......",
		".....SSs........",
		"....SSs.........",
		"...SSs..........",
		"..SSs...........",
		".SSs............",
		".Ss.............",
		"................",
		"................",
		"................",
	},
	"axe": {
		"................",
		".......hhhh.....",
		"......hHHHHd....",
		".....hHHHHHHd...",
		".....hHHHSsHd...",
		"......hHSSdd....",
		".......SSs......",
		"......SSs.......",
		".....SSs........",
		"....SSs.........",
		"...SSs..........",
		"..SSs...........",
		".SSs............",
		".Ss.............",
		"................",
		"................",
	},
	"shovel": {
		"................",
		"............hh..",
		"...........hHHd.",
		"...........hHHd.",
		"............dd..",
		"...........Ss...",
		"..........SSs...",
		".........SSs....",
		"........SSs.....",
		".......SSs......",
		"......SSs.......",
		".....SSs........",
		"....SSs.........",
		"...SSs..........",
		"..SSs...........",
		"..ss............",
	},
}

var heads = map[string]color.RGBA{
	"flint": {80, 82, 96, 255}, "iron": {216, 216, 216, 255}, "bronze": {200, 130, 60, 255}, "electrum": {232, 204, 112, 255},
	"invar": {168, 188, 172, 255}, "steel": {122, 130, 142, 255}, "diamond": {90, 230, 220, 255},
}

var handles = map[string]color.RGBA{
	"wood": {150, 110, 60, 255}, "bone": {232, 228, 205, 255}, "blaze": {240, 170, 40, 255},
}

func scale(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 {
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), 255}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tooltex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/items")
	for kind, rows := range shapes {
		for h, hc := range heads {
			for hd, hdc := range handles {
				im := image.NewRGBA(image.Rect(0, 0, 16, 16))
				for y, row := range rows {
					for x, ch := range row {
						switch ch {
						case 'H':
							im.Set(x, y, hc)
						case 'h':
							im.Set(x, y, scale(hc, 1.25))
						case 'd':
							im.Set(x, y, scale(hc, 0.6))
						case 'S':
							im.Set(x, y, hdc)
						case 's':
							im.Set(x, y, scale(hdc, 0.6))
						}
					}
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("tool_%s_%s_%s.png", kind, h, hd)))
				if err != nil {
					panic(err)
				}
				if err := png.Encode(f, im); err != nil {
					panic(err)
				}
				f.Close()
			}
		}
	}
}
