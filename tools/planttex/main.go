// planttex writes the textures of the magic plants (magic.go): the bud and the
// bloom of every kind as block textures, the seed of every kind and the petal of
// every colour as item textures. Temporary pictures drawn from one colour per kind.
//
//	go run ./tools/planttex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

type kind struct {
	id   string
	col  color.RGBA
	seed string
}

var petals = []struct {
	name string
	col  color.RGBA
}{
	{"white", color.RGBA{240, 240, 240, 255}}, {"orange", color.RGBA{249, 128, 29, 255}}, {"magenta", color.RGBA{199, 78, 189, 255}},
	{"light_blue", color.RGBA{58, 179, 218, 255}}, {"yellow", color.RGBA{254, 216, 61, 255}}, {"lime", color.RGBA{128, 199, 31, 255}},
	{"pink", color.RGBA{243, 139, 170, 255}}, {"gray", color.RGBA{71, 79, 82, 255}}, {"light_gray", color.RGBA{157, 157, 151, 255}},
	{"cyan", color.RGBA{22, 156, 156, 255}}, {"purple", color.RGBA{137, 50, 184, 255}}, {"blue", color.RGBA{60, 68, 170, 255}},
	{"brown", color.RGBA{131, 84, 50, 255}}, {"green", color.RGBA{94, 124, 22, 255}}, {"red", color.RGBA{176, 46, 38, 255}}, {"black", color.RGBA{40, 40, 46, 255}},
}

func kinds() []kind {
	var out []kind
	for _, p := range petals {
		out = append(out, kind{"flower_" + p.name, p.col, "flower_" + p.name})
	}
	metals := []struct {
		id  string
		col color.RGBA
	}{
		{"iron", color.RGBA{216, 175, 147, 255}}, {"gold", color.RGBA{250, 214, 60, 255}}, {"copper", color.RGBA{212, 106, 64, 255}},
		{"tin", color.RGBA{198, 210, 216, 255}}, {"lead", color.RGBA{96, 100, 140, 255}}, {"silver", color.RGBA{222, 232, 244, 255}}, {"nickel", color.RGBA{196, 196, 150, 255}},
	}
	for _, m := range metals {
		out = append(out, kind{"ore_" + m.id, m.col, "ore_" + m.id})
	}
	gems := []struct {
		id  string
		col color.RGBA
	}{
		{"lapis", color.RGBA{40, 70, 190, 255}}, {"redstone", color.RGBA{200, 30, 20, 255}}, {"quartz", color.RGBA{235, 230, 220, 255}},
		{"coal", color.RGBA{50, 50, 55, 255}}, {"diamond", color.RGBA{90, 230, 220, 255}}, {"emerald", color.RGBA{30, 200, 90, 255}},
	}
	for _, g := range gems {
		out = append(out, kind{"gem_" + g.id, g.col, "gem_" + g.id})
	}
	mobs := []struct {
		id  string
		col color.RGBA
	}{
		{"zombie", color.RGBA{90, 160, 90, 255}}, {"skeleton", color.RGBA{225, 225, 215, 255}}, {"creeper", color.RGBA{80, 200, 70, 255}},
		{"spider", color.RGBA{120, 50, 60, 255}}, {"slime", color.RGBA{110, 210, 100, 255}}, {"enderman", color.RGBA{150, 60, 190, 255}}, {"blaze", color.RGBA{255, 170, 40, 255}},
	}
	for _, m := range mobs {
		out = append(out, kind{"ess_" + m.id, m.col, "ess_" + m.id})
	}
	return out
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

var green = color.RGBA{60, 150, 50, 255}

// bud: a stem and a small closed bud in the colour.
var budShape = []string{
	"................",
	"................",
	"................",
	"................",
	"................",
	".......cc.......",
	"......cCCc......",
	"......cCCc......",
	".......cc.......",
	".......s........",
	".......s........",
	"....l..s..l.....",
	".....ll.sll.....",
	"......lsl.......",
	".......s........",
	"................",
}

// bloom: a stem, leaves and a flower of five petals around a centre.
var bloomShape = []string{
	"................",
	"......pp.pp.....",
	".....pPPpPPp....",
	".....pPPPPPp....",
	"......pPoPp.....",
	".....pPPPPPp....",
	".....pPPpPPp....",
	"......pp.pp.....",
	".......s........",
	".......s........",
	"....l..s..l.....",
	".....ll.sll.....",
	"......lsl.......",
	".......s........",
	".......s........",
	"................",
}

func draw(rows []string, c color.RGBA) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y, row := range rows {
		for x, ch := range row {
			switch ch {
			case 'c', 'p':
				im.Set(x, y, scale(c, 0.7))
			case 'C', 'P':
				im.Set(x, y, c)
			case 'o':
				im.Set(x, y, color.RGBA{250, 210, 60, 255})
			case 's':
				im.Set(x, y, scale(green, 0.8))
			case 'l':
				im.Set(x, y, green)
			}
		}
	}
	return im
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

// seed: a small seed in the colour of the kind.
var seedShape = []string{
	"................",
	"................",
	"................",
	"................",
	"................",
	"......ss........",
	".....sSSs.......",
	".....sSSSs......",
	"....sSSSSs......",
	"....sSSSSs......",
	".....sSSs.......",
	"......ss........",
	"................",
	"................",
	"................",
	"................",
}

// petal: a rounded petal.
var petalShape = []string{
	"................",
	"................",
	"................",
	".......pp.......",
	"......pPPp......",
	".....pPPPPp.....",
	".....pPPPPp.....",
	".....pPPPPp.....",
	"......pPPp......",
	".......pp.......",
	"................",
	"................",
	"................",
	"................",
	"................",
	"................",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: planttex pack-dir")
		os.Exit(2)
	}
	blocks := filepath.Join(os.Args[1], "textures/blocks")
	items := filepath.Join(os.Args[1], "textures/items")
	seeds := map[string]bool{}
	for _, k := range kinds() {
		save(filepath.Join(blocks, "plant_"+k.id+"_0.png"), draw(budShape, k.col))
		save(filepath.Join(blocks, "plant_"+k.id+"_1.png"), draw(bloomShape, k.col))
		if !seeds[k.seed] {
			seeds[k.seed] = true
			c := k.col
			// seeds use 'S' for the body and 's' for the outline
			im := image.NewRGBA(image.Rect(0, 0, 16, 16))
			for y, row := range seedShape {
				for x, ch := range row {
					switch ch {
					case 'S':
						im.Set(x, y, c)
					case 's':
						im.Set(x, y, scale(c, 0.5))
					}
				}
			}
			save(filepath.Join(items, "seed_"+k.seed+".png"), im)
		}
	}
	for _, p := range petals {
		im := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y, row := range petalShape {
			for x, ch := range row {
				switch ch {
				case 'P':
					im.Set(x, y, p.col)
				case 'p':
					im.Set(x, y, scale(p.col, 0.6))
				}
			}
		}
		save(filepath.Join(items, "petal_"+p.name+".png"), im)
	}
}
