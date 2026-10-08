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

// keyTexture draws a golden key.
func keyTexture() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	gold, dark := rgb(240, 200, 60), rgb(150, 110, 20)
	for x := 3; x <= 7; x++ { // ring
		for y := 3; y <= 7; y++ {
			if x == 3 || x == 7 || y == 3 || y == 7 {
				img.Set(x, y, gold)
			}
		}
	}
	for i := 0; i < 7; i++ { // blade, diagonal
		img.Set(7+i, 7+i, gold)
		img.Set(8+i, 7+i, dark)
	}
	img.Set(11, 13, gold) // teeth
	img.Set(13, 11, gold)
	return img
}

// lockGlyph is a padlock drawn by the resource pack font
// (assets/resourcepack/dragonfactory/font/glyph_E1.png, first cell).
const lockGlyph = ""

// progressArrowTexture draws a right-pointing arrow whose left step/10 is
// filled (machine progress).
func progressArrowTexture(step int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	empty, full := rgb(70, 74, 88), rgb(236, 168, 48)
	fillTo := 1 + step*14/10
	set := func(x, y int) {
		c := empty
		if x < fillTo {
			c = full
		}
		img.Set(x, y, c)
	}
	for x := 1; x <= 9; x++ {
		for y := 6; y <= 9; y++ {
			set(x, y)
		}
	}
	for i := 0; i <= 5; i++ {
		for y := 2 + i; y <= 13-i; y++ {
			set(9+i, y)
		}
	}
	return img
}
