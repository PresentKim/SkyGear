// Command uitextures writes the textures of the menus
// (assets/resourcepack/sgui/textures/ui/sg: buttons, the window, page arrows, chevrons, the
// chosen slot, the lobby's buttons and icons), which the menu window and
// common/ui draw. Run it from the repository root: go run ./tools/uitextures writes the
// missing pictures, -f writes all.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

const dir = "assets/resourcepack/sgui/textures/ui/sg"

// The theme is the grey look of the game's own screens: a light grey window
// with a white edge at the top left and a dark one at the bottom right, mid
// grey buttons.
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
	arrowColor  = color.RGBA{245, 245, 245, 255}
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

// solid is a 2×2 picture of one colour.
func solid(c color.NRGBA) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for x := range 2 {
		for y := range 2 {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// tint is a white picture in colour c, its alpha kept.
func tint(src image.Image, c color.NRGBA) image.Image {
	b := src.Bounds()
	img := image.NewNRGBA(b)
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			_, _, _, a := src.At(x, y).RGBA()
			img.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, uint8(a >> 8)})
		}
	}
	return img
}

// frame is an 8×8 white edge thick pixels wide around a clear middle, drawn
// tinted and nine-sliced (frame*.json) at any size.
func frame(thick int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for x := range 8 {
		for y := range 8 {
			if min(x, y, 7-x, 7-y) < thick {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return img
}

// flatFace is a state of a lobby button (oui-mc.css .mc-lobby .mc-btn) in
// 10×10, nine-sliced by 4 (flat_*.json): a dark edge, the fill from top rows
// down to bottom rows above the edge, the band in the rows left.
func flatFace(fill, band color.RGBA, top, bottom int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := range 10 {
		for y := range 10 {
			c := band
			switch {
			case x == 0 || y == 0 || x == 9 || y == 9:
				c = color.RGBA{0x1e, 0x1e, 0x1f, 255}
			case y >= top && y < 10-bottom:
				c = fill
			}
			img.Set(x, y, c)
		}
	}
	return img
}

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

// chevron draws the "<" of the back button in pixels, one pixel thick.
func chevron() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 15, 15))
	for i, x := range []int{8, 7, 6, 7, 8} {
		img.SetNRGBA(x, 5+i, color.NRGBA{255, 255, 255, 255})
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

// disc is the ring of a radial menu: a dark translucent disc with a light
// edge, and the hub in the middle drawn lighter with an edge of its own.
func disc(size, hub int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	c := float64(size-1) / 2
	outer, inner := float64(size)/2, float64(hub)/2
	for x := range size {
		for y := range size {
			d := math.Hypot(float64(x)-c, float64(y)-c)
			switch {
			case d > outer:
			case d > outer-2:
				img.SetNRGBA(x, y, color.NRGBA{120, 120, 132, 255})
			case d > inner:
				img.SetNRGBA(x, y, color.NRGBA{30, 30, 36, 220})
			case d > inner-2:
				img.SetNRGBA(x, y, color.NRGBA{120, 120, 132, 255})
			default:
				img.SetNRGBA(x, y, color.NRGBA{52, 52, 60, 235})
			}
		}
	}
	return img
}

// questShape is a quest's shape on the quest canvas, white so the pack tints
// it: the whole shape (edge false) or its edge two pixels thick, drawn at
// 4x and averaged so the slopes are smooth.
func questShape(shape string, edge bool) image.Image {
	const size, ss = 18, 4
	inside := func(x, y, shrink float64) bool {
		r := float64(size)/2 - shrink
		switch shape {
		case "diamond":
			return math.Abs(x)+math.Abs(y) <= r
		case "circle":
			return math.Hypot(x, y) <= r-0.3
		}
		return math.Abs(x) <= r && math.Abs(y) <= r
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for px := range size {
		for py := range size {
			hits := 0
			for sx := range ss {
				for sy := range ss {
					x := float64(px) + (float64(sx)+0.5)/ss - float64(size)/2
					y := float64(py) + (float64(sy)+0.5)/ss - float64(size)/2
					if inside(x, y, 0) && (!edge || !inside(x, y, 2)) {
						hits++
					}
				}
			}
			if hits > 0 {
				img.SetNRGBA(px, py, color.NRGBA{255, 255, 255, uint8(255 * hits / (ss * ss))})
			}
		}
	}
	return img
}

// noEntry is the red ring with a slash a locked quest wears over its icon.
func noEntry() image.Image {
	const size = 16
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	c := float64(size-1) / 2
	for x := range size {
		for y := range size {
			dx, dy := float64(x)-c, float64(y)-c
			d := math.Hypot(dx, dy)
			slash := math.Abs(dx+dy)/math.Sqrt2 <= 1.1 && d <= 6.5
			if (d <= 7.5 && d >= 5.6) || slash {
				img.SetNRGBA(x, y, color.NRGBA{214, 40, 40, 255})
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

func write(name string, img image.Image) {
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
	if err := png.Encode(f, imgio.NRGBA(img)); err != nil {
		panic(err)
	}
}

// sliced writes the picture name with its nine-slice description: base is its
// size and n the width of its edges.
func sliced(name string, img image.Image, n int) {
	write(name, img)
	b := img.Bounds()
	desc := fmt.Sprintf("{ \"base_size\": [ %d, %d ], \"nineslice_size\": %d }\n", b.Dx(), b.Dy(), n)
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(desc), 0o644); err != nil {
		panic(err)
	}
}

var force = len(os.Args) > 1 && os.Args[1] == "-f"

func main() {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	write("panel", roundCorners(bevelBox(16, panelFill, white, panelShade), true, true))
	write("button", bevelBox(16, buttonFill, buttonHi, buttonLo))
	write("button_hover", bevelBox(16, hoverFill, white, hoverLo))
	write("button_pressed", bevelBox(16, pressedFill, buttonLo, buttonFill))
	// the chosen slot of a row (the research or recipe shown): a gold frame,
	// so it does not read as a slot under the mouse
	write("slot_chosen", chosenBox(18))
	write("chevron_left", chevron())
	write("hover_flat", flat(150))
	write("press_flat", flat(112))
	write("arrow_right", arrow(0))
	write("arrow_down", arrow(1))
	write("arrow_left", arrow(2))
	write("arrow_up", arrow(3))
	write("chevron_right", mirror(chevron()))
	write("radial_disc", disc(320, 72))
	for _, s := range []string{"square", "diamond", "circle"} {
		write("quest_"+s, questShape(s, false))
		write("quest_"+s+"_edge", questShape(s, true))
	}
	write("quest_locked", noEntry())
	// A quest canvas cell names its pictures from its code (invui.QuestCode):
	// ql<flag><state> a line piece, qf<shape> a quest's fill, qe<shape><state>
	// its edge. States are open, done and locked; shapes square, diamond and
	// circle.
	lines := []color.NRGBA{{0x9a, 0x9a, 0xa6, 255}, {0x4c, 0xc9, 0x3c, 255}, {0x4a, 0x4a, 0x55, 255}}
	edges := []color.NRGBA{{0xdc, 0xdc, 0xe2, 255}, {0x4c, 0xc9, 0x3c, 255}, {0x6a, 0x6a, 0x74, 255}}
	for st, c := range lines {
		write(fmt.Sprintf("ql1%d", st), solid(c))
		write(fmt.Sprintf("ql0%d", st), solid(color.NRGBA{}))
	}
	for sh, name := range []string{"square", "diamond", "circle"} {
		write(fmt.Sprintf("qf%d", sh), questShape(name, false))
		for st, c := range edges {
			write(fmt.Sprintf("qe%d%d", sh, st), tint(questShape(name, true), c))
		}
	}
	sliced("frame1", frame(1), 1)
	sliced("frame2", frame(2), 2)
	for _, s := range []struct {
		name              string
		fill, hover, band color.RGBA
	}{
		{"lobby", color.RGBA{0x48, 0x49, 0x4a, 255}, color.RGBA{0x5a, 0x5b, 0x5c, 255}, color.RGBA{0x24, 0x25, 0x26, 255}},
		{"light", color.RGBA{0xd0, 0xd1, 0xd4, 255}, color.RGBA{0xe6, 0xe8, 0xeb, 255}, color.RGBA{0x58, 0x58, 0x5a, 255}},
		{"play", color.RGBA{0x3c, 0x85, 0x27, 255}, color.RGBA{0x4a, 0x9b, 0x31, 255}, color.RGBA{0x1d, 0x4d, 0x13, 255}},
		{"pick", color.RGBA{0x28, 0x29, 0x30, 255}, color.RGBA{0x34, 0x36, 0x40, 255}, color.RGBA{0x18, 0x18, 0x1c, 255}},
	} {
		sliced("flat_"+s.name, flatFace(s.fill, s.band, 1, 3), 4)
		sliced("flat_"+s.name+"_hover", flatFace(s.hover, s.band, 1, 3), 4)
		sliced("flat_"+s.name+"_pressed", flatFace(s.hover, s.band, 2, 2), 4)
	}

	// the lobby screens' buttons and windows
	write("launcher_button", launcherBox(color.NRGBA{18, 28, 48, 200}, color.NRGBA{110, 135, 170, 230}, 1))
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
	))
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
	))
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
	))
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
	))
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
	))
}
