// Command conduittex writes the conduit textures of the pack from EnderIO's
// 64×64 sheets in src/ (Unlicense, see third_party/README.md): per kind and
// tier the display entity and item block sheet (layout in tools/conduitgeo),
// and the conduit block texture, a grey sheet with one transparent pixel
// that the model shows (the rest only colours break particles).
//
//	go run ./tools/conduittex tools/conduittex/src assets/resourcepack/skygear
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

// part draws one conduit into dst at x0: the arm strip (16×52) at (x0, 0), the
// core face (scaled to 16×16) at (x0+16, 0) and the strip turned on its side,
// scaled to 32×10, at (x0, 52).
func part(dst *image.RGBA, src image.Image, x0 int) {
	draw.Draw(dst, image.Rect(x0, 0, x0+16, 52), src, image.Pt(0, 0), draw.Src)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// 24×24 to 16×16: the middle of each 1.5 pixel step.
			dst.Set(x0+16+x, y, src.At(40+(x*3+1)/2, 40+(y*3+1)/2))
		}
	}
	for y := 0; y < 10; y++ {
		for x := 0; x < 32; x++ {
			// the strip's length (52) runs along x, its width (16) along y
			dst.Set(x0+x, 52+y, src.At(y*16/10, x*52/32))
		}
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: conduittex src-dir pack-dir")
		os.Exit(2)
	}
	src, pack := os.Args[1], os.Args[2]
	for _, kind := range []string{"energy", "item", "fluid"} {
		for t := 1; t <= 3; t++ {
			name := fmt.Sprintf("conduit_%s_%d.png", kind, t)
			sheet := imgio.Load(filepath.Join(src, name))
			// the texture of the display entity
			im := image.NewRGBA(image.Rect(0, 0, 64, 64))
			part(im, sheet, 0)
			imgio.Save(filepath.Join(pack, "textures/entity/skygear", name), im)
			// the same sheet is the texture of the item block (conduit icons)
			imgio.Save(filepath.Join(pack, "textures/blocks", name), im)
		}
	}
	plates(src, pack)
	// the block of a conduit is a speck the display entity draws over; its
	// texture shows only in the particles of breaking it
	node := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x > 0 || y > 0 { // the model uses the transparent first pixel only
				node.Set(x, y, color.RGBA{58, 60, 60, 255})
			}
		}
	}
	imgio.Save(filepath.Join(pack, "textures/blocks/conduit_node.png"), node)
}

// channelColours are the colours of the 16 channels (dye colours, the order of
// EnderIO's list: green first, the default channel).
var channelColours = [16]color.RGBA{
	{94, 124, 22, 255}, {131, 84, 50, 255}, {60, 68, 170, 255}, {137, 50, 184, 255},
	{22, 156, 156, 255}, {157, 157, 151, 255}, {71, 79, 82, 255}, {243, 139, 170, 255},
	{128, 199, 31, 255}, {254, 216, 61, 255}, {58, 179, 218, 255}, {199, 78, 189, 255},
	{249, 128, 29, 255}, {249, 255, 254, 255}, {29, 29, 33, 255}, {176, 46, 38, 255},
}

// plates writes the textures of the connector entity (conduitplate.go): for each
// channel colour the input and output arrow (EnderIO's arrows, 24×24, tinted):
// at (0, 0) as they are (the input points down, the output up) and at (32, 0)
// turned a quarter counterclockwise (the input points right, the output left),
// for the faces where an arrow runs along the u axis.
func plates(src, pack string) {
	dir := filepath.Join(pack, "textures/entity/skygear")
	for _, dir2 := range []string{"in", "out"} {
		arrow := imgio.Load(filepath.Join(src, "arrow_"+dir2+".png"))
		for i, col := range channelColours {
			im := image.NewRGBA(image.Rect(0, 0, 64, 64))
			for y := 0; y < 24; y++ {
				for x := 0; x < 24; x++ {
					r, g, b, a := arrow.At(x, y).RGBA()
					if a == 0 {
						continue
					}
					c := color.RGBA{
						uint8(uint32(col.R) * (r >> 8) / 255), uint8(uint32(col.G) * (g >> 8) / 255), uint8(uint32(col.B) * (b >> 8) / 255), uint8(a >> 8),
					}
					im.Set(x, y, c)
					// a quarter turn counterclockwise: (x, y) goes to (y, 23-x)
					im.Set(32+y, 23-x, c)
				}
			}
			imgio.Save(filepath.Join(dir, fmt.Sprintf("channel_%s_%d.png", dir2, i)), im)
		}
	}
}
