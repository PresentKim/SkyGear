// Command elevatortex writes the block textures of the wool elevators
// (content/skygear/elevator): wool of each colour with a darker rim on the
// sides, and on top an ender pearl between an up and a down arrow. They are
// drawn from the dye colours, with no picture of another game.
//
//	go run ./tools/elevatortex assets/resourcepack/skygear
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/tools/internal/imgio"
)

// top is the emblem on the top face: 'a' arrow, 'o' the dark socket around
// the pearl, 'p' pearl, 'h' its shine, 'd' its dark side. Other cells are
// wool.
var top = []string{
	"................",
	".......aa.......",
	"......aaaa......",
	".....aaaaaa.....",
	"................",
	"......oooo......",
	".....oppppo.....",
	".....ophppo.....",
	".....opppdo.....",
	".....opdddo.....",
	"......oooo......",
	"................",
	".....aaaaaa.....",
	"......aaaa......",
	".......aa.......",
	"................",
}

func scale(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 {
		return uint8(min(max(v, 0), 255))
	}
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), 255}
}

// luma is the brightness of c, 0 to 255.
func luma(c color.RGBA) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// grain is a fixed pseudo-random shade for the wool at x, y.
func grain(x, y int) float64 {
	h := uint32(x*73856093) ^ uint32(y*19349663)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return 0.9 + float64(h%5)*0.04
}

// wool draws the wool of col with a darker rim.
func wool(col color.RGBA) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			f := grain(x, y)
			if x == 0 || y == 0 || x == 15 || y == 15 {
				f *= 0.7
			}
			im.Set(x, y, scale(col, f))
		}
	}
	return im
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: elevatortex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/blocks")
	dark, light := color.RGBA{36, 40, 48, 255}, color.RGBA{226, 228, 232, 255}
	pearl := color.RGBA{32, 150, 128, 255}
	for _, c := range item.Colours() {
		col := c.RGBA()
		// The arrows stand out from the wool: light on dark wool.
		arrow := dark
		if luma(col) < 80 {
			arrow = light
		}
		side := wool(col)
		imgio.Save(filepath.Join(dir, "elevator_"+c.String()+"_side.png"), side)
		up := wool(col)
		for y, row := range top {
			for x, ch := range []rune(row) {
				switch ch {
				case 'a':
					up.Set(x, y, arrow)
				case 'o':
					up.Set(x, y, dark)
				case 'p':
					up.Set(x, y, pearl)
				case 'h':
					up.Set(x, y, scale(pearl, 1.6))
				case 'd':
					up.Set(x, y, scale(pearl, 0.6))
				}
			}
		}
		imgio.Save(filepath.Join(dir, "elevator_"+c.String()+"_up.png"), up)
	}
}
