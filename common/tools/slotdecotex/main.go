// slotdecotex writes the textures of the slotdeco resource pack
// (common packs/slotdeco/textures/ui/df): the slot backgrounds of the
// rarities (rarity_1..4), the tier pips drawn over a slot (front_1..4) and the badges (front_5 warning, front_6 lock) and the
// rarity frames of the selected-item bubble (tip_1..4). They are flat
// temporary pictures; the PNGs can be replaced by hand.
//
//	go run ./tools/slotdecotex packs/slotdeco packs/dfui
//
// The second pack (optional, the dfui theme) gets the pictures of the form
// slots (textures/ui/df/deco_<rarity>_<front>.png, 22×22: a frame and a
// badge in one picture, so a form slot needs one control, df_form.json
// gslot_deco).
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// rarityColours are the colours of the rarities, in the order of
// slotdeco.Rarity (Uncommon to Legendary); they are the colour codes that
// the names use (§a §b §d §6).
var rarityColours = [4]color.NRGBA{
	{85, 255, 85, 255},
	{85, 255, 255, 255},
	{255, 85, 255, 255},
	{255, 170, 0, 255},
}

// tierColours are the colours of the pips of tier 1 to 4: iron, gold,
// diamond, netherite-violet.
var tierColours = [4]color.NRGBA{
	{205, 205, 215, 255},
	{250, 210, 60, 255},
	{90, 235, 235, 255},
	{190, 110, 255, 255},
}

func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// background is a 16×16 slot background: a bright 1px edge, a fainter ring
// inside it and a light fill, so that the item on it stays easy to read.
func background(c color.NRGBA) image.Image {
	const n = 16
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for x := range n {
		for y := range n {
			edge := min(x, y, n-1-x, n-1-y)
			switch edge {
			case 0:
				img.SetNRGBA(x, y, withAlpha(c, 235))
			case 1:
				img.SetNRGBA(x, y, withAlpha(c, 120))
			default:
				img.SetNRGBA(x, y, withAlpha(c, 64))
			}
		}
	}
	return img
}

// frame is a 16×16 frame with a transparent middle, for the bubble.
func frame(c color.NRGBA) image.Image {
	const n = 16
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for x := range n {
		for y := range n {
			switch min(x, y, n-1-x, n-1-y) {
			case 0:
				img.SetNRGBA(x, y, withAlpha(c, 255))
			case 1:
				img.SetNRGBA(x, y, withAlpha(c, 90))
			}
		}
	}
	return img
}

// pips is a 16×16 picture of tier small squares in the top left corner, each
// with a dark edge so that it shows on any item.
func pips(tier int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	dark := color.NRGBA{20, 20, 24, 255}
	for i := range tier {
		x0 := 1 + i*3
		for dx := 0; dx < 4; dx++ {
			for dy := 0; dy < 4; dy++ {
				img.SetNRGBA(x0+dx, 1+dy, dark)
			}
		}
		for dx := 1; dx < 3; dx++ {
			for dy := 1; dy < 3; dy++ {
				img.SetNRGBA(x0+dx, 1+dy, tierColours[tier-1])
			}
		}
	}
	return img
}

// warnBadge is a red disc with a white exclamation mark in the top right.
func warnBadge() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	red, dark, white := color.NRGBA{225, 45, 45, 255}, color.NRGBA{40, 10, 10, 255}, color.NRGBA{255, 255, 255, 255}
	cx, cy := 11, 4
	for x := cx - 4; x <= cx+4; x++ {
		for y := cy - 4; y <= cy+4; y++ {
			d := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			switch {
			case d <= 12:
				img.SetNRGBA(x, y, red)
			case d <= 20:
				img.SetNRGBA(x, y, dark)
			}
		}
	}
	for y := cy - 2; y <= cy+1; y++ {
		img.SetNRGBA(cx, y, white)
	}
	img.SetNRGBA(cx, cy+3, white)
	return img
}

// lockBadge is a small padlock in the top right.
func lockBadge() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	body, dark := color.NRGBA{250, 200, 60, 255}, color.NRGBA{40, 30, 10, 255}
	for x := 8; x <= 14; x++ { // shackle
		img.SetNRGBA(x, 0, dark)
	}
	for y := 0; y <= 3; y++ {
		img.SetNRGBA(8, y, dark)
		img.SetNRGBA(14, y, dark)
	}
	for x := 7; x <= 15; x++ { // body
		for y := 4; y <= 9; y++ {
			c := body
			if x == 7 || x == 15 || y == 4 || y == 9 {
				c = dark
			}
			img.SetNRGBA(x, y, c)
		}
	}
	img.SetNRGBA(11, 6, dark)
	img.SetNRGBA(11, 7, dark)
	return img
}

func write(dir, name string, img image.Image, nineslice int) {
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	f.Close()
	if nineslice > 0 {
		b, _ := json.Marshal(map[string]any{"nineslice_size": nineslice, "base_size": []int{16, 16}})
		if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o644); err != nil {
			panic(err)
		}
	}
}

// frontImage is the front image n (0: none): tier pips 1 to 4, the warning 5
// and the lock 6, each 16×16 at the top left or right of a larger picture.
func frontImage(n int) image.Image {
	switch {
	case n >= 1 && n <= 4:
		return pips(n)
	case n == 5:
		return warnBadge()
	case n == 6:
		return lockBadge()
	}
	return image.NewNRGBA(image.Rect(0, 0, 16, 16))
}

// formDeco is a 22×22 form slot picture: the frame of the rarity (0: none)
// and the front image n over it.
func formDeco(r, n int) image.Image {
	const size = 22
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	if r > 0 {
		c := rarityColours[r-1]
		for x := range size {
			for y := range size {
				switch min(x, y, size-1-x, size-1-y) {
				case 0:
					img.SetNRGBA(x, y, withAlpha(c, 255))
				case 1:
					img.SetNRGBA(x, y, withAlpha(c, 90))
				}
			}
		}
	}
	front := frontImage(n)
	for x := range 16 {
		for y := range 16 {
			if c := front.At(x, y).(color.NRGBA); c.A > 0 {
				img.SetNRGBA(x+3, y+2, c)
			}
		}
	}
	return img
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: slotdecotex <slotdeco pack dir> [<dfui pack dir>]")
		os.Exit(2)
	}
	if len(os.Args) == 3 {
		formDir := filepath.Join(os.Args[2], "textures", "ui", "df")
		for r := 0; r <= 4; r++ {
			for n := 0; n <= 6; n++ {
				if r > 0 || n > 0 {
					write(formDir, fmt.Sprintf("deco_%d_%d", r, n), formDeco(r, n), 0)
				}
			}
		}
	}
	dir := filepath.Join(os.Args[1], "textures", "ui", "df")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	for i, c := range rarityColours {
		write(dir, fmt.Sprintf("rarity_%d", i+1), background(c), 3)
		write(dir, fmt.Sprintf("tip_%d", i+1), frame(c), 3)
	}
	for tier := 1; tier <= 4; tier++ {
		write(dir, fmt.Sprintf("front_%d", tier), pips(tier), 0)
	}
	write(dir, "front_5", warnBadge(), 0)
	write(dir, "front_6", lockBadge(), 0)
}
