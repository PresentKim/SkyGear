package dragonfactory

import (
	"image"
	"image/color"
)

// Placeholder textures are generated in code until real art exists
// (docs/tech/addon.md §6: Go textures go into the auto-generated pack).

// frameTexture draws a framed square: border, inner fill and an optional
// centred gem of size gem.
func frameTexture(border, fill, gem color.RGBA, gemSize int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	lo, hi := 8-gemSize/2, 8+(gemSize+1)/2
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			c := fill
			switch {
			case x == 0 || y == 0 || x == 15 || y == 15:
				c = border
			case gemSize > 0 && x >= lo && x < hi && y >= lo && y < hi:
				c = gem
			case (x+y)%5 == 0:
				c = shade(fill, 0.85)
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func shade(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{R: uint8(float64(c.R) * f), G: uint8(float64(c.G) * f), B: uint8(float64(c.B) * f), A: c.A}
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }
