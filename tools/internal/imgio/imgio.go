// Package imgio reads and writes the PNG files of the texture tools. The tools
// run by hand, so an error panics.
package imgio

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// Load decodes the PNG at path.
func Load(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		panic(fmt.Errorf("%s: %w", path, err))
	}
	return im
}

// NRGBA returns im as non-premultiplied RGBA. The client draws grayscale PNGs
// (color types 0 and 4) translucent, so the tools never encode those: gray v
// becomes (v, v, v) with the same alpha (the decoder already returns gray+alpha
// as NRGBA).
func NRGBA(im image.Image) *image.NRGBA {
	b := im.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			switch c := im.At(x, y).(type) {
			case color.Gray:
				out.SetNRGBA(x, y, color.NRGBA{c.Y, c.Y, c.Y, 255})
			default:
				out.Set(x, y, c)
			}
		}
	}
	return out
}

// Save writes im to path as an RGB or RGBA PNG.
func Save(path string, im image.Image) {
	im = NRGBA(im)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, im); err != nil {
		_ = f.Close()
		panic(fmt.Errorf("%s: %w", path, err))
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}
