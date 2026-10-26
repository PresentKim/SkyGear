// metaltex writes the item textures of the metals of the pack (metals.go):
// the ore chunk of each new metal (the copper chunk with its ore pixels
// recoloured), the dust of every metal (a small mound) and the ingots of the
// new metals and alloys. They are temporary pictures, drawn from the colour of
// each metal.
//
//	go run ./tools/metaltex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/presentkim/dragonfactory/tools/internal/imgio"
)

type metal struct {
	name  string
	col   color.RGBA
	chunk bool // a new ore: has a chunk texture
	ingot bool // has an ingot texture (iron, gold and copper use the vanilla ones)
}

var metals = []metal{
	{"iron", color.RGBA{216, 175, 147, 255}, false, false},
	{"gold", color.RGBA{250, 214, 60, 255}, false, false},
	{"copper", color.RGBA{212, 106, 64, 255}, false, false},
	{"tin", color.RGBA{198, 210, 216, 255}, true, true},
	{"lead", color.RGBA{96, 100, 140, 255}, true, true},
	{"silver", color.RGBA{222, 232, 244, 255}, true, true},
	{"nickel", color.RGBA{196, 196, 150, 255}, true, true},
	{"bronze", color.RGBA{196, 128, 58, 255}, false, true},
	{"electrum", color.RGBA{232, 204, 112, 255}, false, true},
	{"invar", color.RGBA{168, 188, 172, 255}, false, true},
	{"steel", color.RGBA{108, 114, 126, 255}, false, true},
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

// ingotShape: 'o' outline, 't' top face, 'f' front face, 's' side face.
var ingotShape = []string{
	"................",
	"................",
	"................",
	"................",
	"....oooooooo....",
	"...ottttttttto..",
	"..otttttttttto..",
	".ofttttttttso...",
	".offfffffffso...",
	".offfffffffso...",
	"..ooooooooooo...",
	"................",
	"................",
	"................",
	"................",
	"................",
}

// dustShape: a mound; digits are shades from light (0) to dark (3).
var dustShape = []string{
	"................",
	"................",
	"................",
	"................",
	"................",
	".......00.......",
	"......0010......",
	".....001110.....",
	"....0011211.....",
	"...00112122.0...",
	"..001121223.1...",
	".0011212233.....",
	".1122122333.....",
	"..2233333.......",
	"................",
	"................",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: metaltex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/items")
	template := imgio.Load(filepath.Join(dir, "ore_chunk_copper.png"))
	for _, m := range metals {
		if m.chunk {
			im := image.NewRGBA(image.Rect(0, 0, 16, 16))
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					r, g, b, a := template.At(x, y).RGBA()
					c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
					if c.A != 0 && int(c.R)-int(c.B) > 40 { // an ore pixel: keep its brightness
						c = scale(m.col, float64(c.R)/212)
					}
					im.Set(x, y, c)
				}
			}
			imgio.Save(filepath.Join(dir, "ore_chunk_"+m.name+".png"), im)
		}
		dust := image.NewRGBA(image.Rect(0, 0, 16, 16))
		shades := []float64{1.18, 1.0, 0.85, 0.7}
		for y, row := range dustShape {
			for x, ch := range row {
				if ch >= '0' && ch <= '3' {
					dust.Set(x, y, scale(m.col, shades[ch-'0']))
				}
			}
		}
		imgio.Save(filepath.Join(dir, "metal_dust_"+m.name+".png"), dust)
		if m.ingot {
			im := image.NewRGBA(image.Rect(0, 0, 16, 16))
			for y, row := range ingotShape {
				for x, ch := range row {
					switch ch {
					case 'o':
						im.Set(x, y, scale(m.col, 0.45))
					case 't':
						im.Set(x, y, scale(m.col, 1.2))
					case 'f':
						im.Set(x, y, scale(m.col, 0.95))
					case 's':
						im.Set(x, y, scale(m.col, 0.7))
					}
				}
			}
			imgio.Save(filepath.Join(dir, "ingot_"+m.name+".png"), im)
		}
	}
}
