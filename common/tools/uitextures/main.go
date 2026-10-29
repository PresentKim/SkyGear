// Command uitextures writes the flat textures of the grey JSON UI theme
// (packs/sgui/textures/ui/sg). Run it from common/: go run ./tools/uitextures
// writes the missing pictures, -f writes all.
package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const dir = "packs/sgui/textures/ui/sg"

// The theme is the grey look of the game's own screens: a light grey window
// with a white edge at the top left and a dark one at the bottom right, mid
// grey buttons, dark inset item slots.
var (
	black       = color.RGBA{16, 16, 16, 255}
	white       = color.RGBA{255, 255, 255, 255}
	panelFill   = color.RGBA{198, 198, 198, 255}
	panelShade  = color.RGBA{85, 85, 85, 255}
	buttonFill  = color.RGBA{66, 66, 66, 255}
	buttonHi    = color.RGBA{92, 92, 92, 255}
	buttonLo    = color.RGBA{40, 40, 40, 255}
	hoverFill   = color.RGBA{84, 84, 128, 255}
	hoverLo     = color.RGBA{56, 56, 92, 255}
	pressedFill = color.RGBA{48, 48, 48, 255}
	slotFill    = color.RGBA{139, 139, 139, 255}
	slotDark    = color.RGBA{55, 55, 55, 255}
	arrowColor  = color.RGBA{245, 245, 245, 255}
	dim         = color.RGBA{0, 0, 0, 140}
)

// bevelBox draws a size×size square with a black edge, a light line inside
// it along the top and left and a dark one along the bottom and right.
func bevelBox(size int, fill, hi, lo color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := range size {
		for y := range size {
			c := fill
			switch {
			case x == 0 || y == 0 || x == size-1 || y == size-1:
				c = black
			case x == 1 || y == 1:
				c = hi
			case x == size-2 || y == size-2:
				c = lo
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// roundCorners makes the corners of img (a bevelBox) round the way the game's
// windows are: three pixels cleared and the step between the edges black.
// top and bottom choose which corners.
func roundCorners(img image.Image, top, bottom bool) image.Image {
	out := img.(*image.RGBA)
	n := out.Bounds().Dx()
	m := out.Bounds().Dy()
	clear := color.RGBA{}
	cut := func(x, y, sx, sy int) {
		for _, p := range [][2]int{{0, 0}, {1, 0}, {0, 1}} {
			out.Set(x+sx*p[0], y+sy*p[1], clear)
		}
		out.Set(x+sx, y+sy, black)
	}
	if top {
		cut(0, 0, 1, 1)
		cut(n-1, 0, -1, 1)
	}
	if bottom {
		cut(0, m-1, 1, -1)
		cut(n-1, m-1, -1, -1)
	}
	return out
}

// insetBox is an item slot: dark along the top and left, white along the
// bottom and right, so it looks pressed in.
func insetBox(size int, fill color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := range size {
		for y := range size {
			c := fill
			switch {
			case x == size-1 || y == size-1:
				c = white
			case x == 0 || y == 0:
				c = slotDark
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// chosenBox is a slot with a two pixel gold frame and a lighter field.
func chosenBox(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := range size {
		for y := range size {
			c := color.RGBA{176, 176, 176, 255}
			switch {
			case x == 0 || y == 0 || x == size-1 || y == size-1:
				c = color.RGBA{214, 158, 24, 255}
			case x == 1 || y == 1 || x == size-2 || y == size-2:
				c = color.RGBA{255, 222, 92, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// bevel is the window's inner edge layer; the window picture draws its own
// bevel, so it is clear.
func bevel() image.Image { return image.NewNRGBA(image.Rect(0, 0, 16, 16)) }

// arrow draws a 16×16 arrow pointing right, rotated by quarter turns.
func arrow(turns int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	set := func(x, y int) {
		for range turns { // rotate 90° clockwise per turn
			x, y = 15-y, x
		}
		img.Set(x, y, arrowColor)
	}
	for x := 2; x <= 9; x++ { // shaft
		for y := 6; y <= 9; y++ {
			set(x, y)
		}
	}
	for i := 0; i <= 5; i++ { // head
		for y := 2 + i; y <= 13-i; y++ {
			set(9+i, y)
		}
	}
	return img
}

// chevron draws the "<" of the back button in pixels, two pixels thick.
func chevron() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	set := func(x, y int) {
		img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		img.SetNRGBA(x+1, y, color.NRGBA{255, 255, 255, 255})
	}
	for i := range 6 {
		set(10-i, 2+i)
		set(5+i, 8+i)
	}
	return img
}

// flat is a plain square, the highlight behind the small buttons of the control bar.
func flat(v uint8) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	return img
}

// launcherBox is a button of the lobby's launcher menu: a translucent dark
// fill with a thin light edge and rounded corners (two pixels cut).
func launcherBox(fill, edge color.NRGBA, thick int) image.Image {
	const n = 16
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for x := range n {
		for y := range n {
			d := min(x, y, n-1-x, n-1-y)
			corner := min(x, n-1-x)+min(y, n-1-y) < 2
			switch {
			case corner:
			case d < thick:
				img.SetNRGBA(x, y, edge)
			default:
				img.SetNRGBA(x, y, fill)
			}
		}
	}
	return img
}

// glyph draws a 16×16 white icon from rows of '#'.
func glyph(rows ...string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y, row := range rows {
		for x, c := range row {
			if c == '#' {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return img
}

// mirror flips an icon left to right.
func mirror(src image.Image) image.Image {
	b := src.Bounds()
	img := image.NewNRGBA(b)
	for x := range b.Dx() {
		for y := range b.Dy() {
			img.Set(b.Dx()-1-x, y, src.At(x, y))
		}
	}
	return img
}

func write(name string, img image.Image, nineslice int) {
	// What is there stays (the pictures may have been changed by hand);
	// run with -f to write everything again.
	if _, err := os.Stat(filepath.Join(dir, name+".png")); err == nil && !force {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	if nineslice > 0 {
		b, _ := json.MarshalIndent(map[string]any{"nineslice_size": nineslice, "base_size": []int{img.Bounds().Dx(), img.Bounds().Dy()}}, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o644); err != nil {
			panic(err)
		}
	}
}

var force = len(os.Args) > 1 && os.Args[1] == "-f"

func main() {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	write("panel", roundCorners(bevelBox(16, panelFill, white, panelShade), true, true), 4)
	// the window has no title bar of its own: the picture is clear
	write("title", bevel(), 2)
	write("clear", bevel(), 0)
	write("button", bevelBox(16, buttonFill, buttonHi, buttonLo), 3)
	write("button_hover", bevelBox(16, hoverFill, white, hoverLo), 3)
	write("button_pressed", bevelBox(16, pressedFill, buttonLo, buttonFill), 3)
	write("panel_bevel", bevel(), 5)
	write("slot", insetBox(18, slotFill), 1)
	// the chosen slot of a row (the research or recipe shown): a gold frame,
	// so it does not read as a slot under the mouse
	write("slot_chosen", chosenBox(18), 0)
	// the dark field the game's own input forms are drawn in
	write("inset", insetBox(8, color.RGBA{60, 60, 60, 255}), 1)
	dimImg := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			dimImg.Set(x, y, dim)
		}
	}
	write("dim", dimImg, 0)
	write("chevron_left", chevron(), 0)
	write("hover_flat", flat(150), 0)
	write("press_flat", flat(112), 0)
	write("arrow_right", arrow(0), 0)
	write("arrow_down", arrow(1), 0)
	write("arrow_left", arrow(2), 0)
	write("arrow_up", arrow(3), 0)
	write("chevron_right", mirror(chevron()), 0)

	// the Ore UI lobby screens' buttons and windows (oui-mc.css .mc-lobby)
	write("launcher_button", launcherBox(color.NRGBA{18, 28, 48, 200}, color.NRGBA{110, 135, 170, 230}, 1), 4)
	write("launcher_button_hover", launcherBox(color.NRGBA{34, 60, 98, 220}, color.NRGBA{150, 210, 255, 255}, 2), 4)
	write("launcher_button_pressed", launcherBox(color.NRGBA{14, 22, 38, 235}, color.NRGBA{120, 170, 220, 255}, 2), 4)
	write("icon_cube", glyph(
		".......##.......",
		".....##..##.....",
		"...##......##...",
		".##..........##.",
		".#.##......##.#.",
		".#...##..##...#.",
		".#.....##.....#.",
		".#.....##.....#.",
		".#.....##.....#.",
		".#.....##.....#.",
		".#.....##.....#.",
		".##....##....##.",
		"...##..##..##...",
		".....##..##.....",
		".......##.......",
	), 0)
	write("icon_people", glyph(
		"................",
		"................",
		"...####.........",
		"..#....#..###...",
		"..#....#.#...#..",
		"..#....#.#...#..",
		"...####...###...",
		"................",
		".########.####..",
		"#........#....#.",
		"#........#....#.",
		"#........#....#.",
		"##########....#.",
		".........######.",
	), 0)
	write("icon_person", glyph(
		"................",
		"......####......",
		".....#....#.....",
		".....#....#.....",
		".....#....#.....",
		"......####......",
		"................",
		"....########....",
		"...#........#...",
		"..#..........#..",
		"..#..........#..",
		"..############..",
	), 0)
	write("icon_crown", glyph(
		"................",
		"................",
		"..#....##....#..",
		"..##..#..#..##..",
		"..#.#.#..#.#.#..",
		"..#..#....#..#..",
		"..#..........#..",
		"..#..........#..",
		"..#..........#..",
		"..############..",
		"................",
		"..############..",
	), 0)
	write("icon_help", glyph(
		"................",
		".....######.....",
		"...##......##...",
		"..#....##....#..",
		"..#...#..#...#..",
		".#.......#....#.",
		".#......#.....#.",
		".#.....#......#.",
		".#.....#......#.",
		".#............#.",
		"..#....#.....#..",
		"..#..........#..",
		"...##......##...",
		".....######.....",
	), 0)
}
