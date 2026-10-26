// Command machinetex writes placeholder block textures of machines derived
// from another machine's: every reddish accent pixel of the base is recoloured
// to the new accent, keeping its brightness.
//
//	go run ./tools/machinetex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/presentkim/dragonfactory/tools/internal/imgio"
)

type variant struct {
	name, base string
	faces      []string
	accent     color.RGBA
}

var variants = []variant{
	{"alloy_smelter", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{72, 150, 215, 255}},
	{"mob_simulator", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{160, 80, 200, 255}},
	{"fluid_tank", "generator", []string{"side", "up", "down"}, color.RGBA{70, 130, 230, 255}},
	{"lava_generator", "generator", []string{"side", "up", "down"}, color.RGBA{255, 120, 20, 255}},
	{"crucible", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{255, 70, 30, 255}},
	{"ore_washer", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{60, 190, 200, 255}},
	{"storage_controller", "generator", []string{"side", "up", "down"}, color.RGBA{240, 200, 60, 255}},
	{"storage_link", "generator", []string{"side", "up", "down"}, color.RGBA{150, 160, 180, 255}},
	{"tinker_table", "generator", []string{"side", "up", "down"}, color.RGBA{200, 120, 60, 255}},
	{"energy_cell", "generator", []string{"side", "up", "down"}, color.RGBA{70, 200, 110, 255}},
	{"electric_furnace", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{255, 140, 40, 255}},
	{"auto_crafter", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{210, 170, 90, 255}},
	{"laboratory", "crusher", []string{"front", "back", "side", "up", "down"}, color.RGBA{96, 200, 184, 255}},
	{"item_vacuum", "generator", []string{"side", "up", "down"}, color.RGBA{90, 200, 230, 255}},
	{"tree_feller", "generator", []string{"side", "up", "down"}, color.RGBA{140, 110, 50, 255}},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: machinetex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/blocks")
	for _, v := range variants {
		for _, face := range v.faces {
			src := imgio.Load(filepath.Join(dir, v.base+"_"+face+".png"))
			b := src.Bounds()
			out := image.NewRGBA(b)
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					r, g, bl, a := src.At(x, y).RGBA()
					c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
					if c.A != 0 && int(c.R)-int(c.B) > 45 && c.R > 110 {
						f := float64(c.R) / 215
						c = color.RGBA{clamp(float64(v.accent.R) * f), clamp(float64(v.accent.G) * f), clamp(float64(v.accent.B) * f), c.A}
					}
					out.Set(x, y, c)
				}
			}
			imgio.Save(filepath.Join(dir, v.name+"_"+face+".png"), out)
		}
	}
}

func clamp(v float64) uint8 {
	if v > 255 {
		return 255
	}
	return uint8(v)
}
