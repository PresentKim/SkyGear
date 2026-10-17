// Command uitextures writes the flat textures of the dark JSON UI theme
// (assets/resourcepack/dragonfactory/textures/ui/df); the empty upgrade slot
// textures there (slot_empty_*) are drawn by hand. Run it from the
// repository root after changing the colours: go run ./tools/uitextures
// (only the pictures that are missing; go run ./tools/uitextures -f writes all).
package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const dir = "assets/resourcepack/dragonfactory/textures/ui/df"

var (
	panelFill   = color.RGBA{22, 24, 30, 240}
	panelBorder = color.RGBA{64, 70, 86, 255}
	titleFill   = color.RGBA{34, 37, 46, 255}
	accent      = color.RGBA{236, 168, 48, 255}
	buttonFill  = color.RGBA{40, 44, 55, 255}
	buttonEdge  = color.RGBA{68, 74, 92, 255}
	hoverFill   = color.RGBA{56, 62, 78, 255}
	pressedFill = color.RGBA{30, 33, 41, 255}
	tabFill     = color.RGBA{27, 29, 37, 255}
	tabOnFill   = color.RGBA{48, 53, 69, 255}
	slotFill    = color.RGBA{14, 15, 19, 255}
	dim         = color.RGBA{0, 0, 0, 140}
)

// box draws a size×size square with a 1px border.
func box(size int, fill, border color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := range size {
		for y := range size {
			c := fill
			if x == 0 || y == 0 || x == size-1 || y == size-1 {
				c = border
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// accentBox is a button whose bottom edge is the accent colour.
func accentBox(size int, fill, border, bottom color.RGBA) image.Image {
	img := box(size, fill, border).(*image.RGBA)
	for x := 1; x < size-1; x++ {
		img.Set(x, size-2, bottom)
	}
	return img
}

// tabBox is a tab: a box with a bar of the accent colour barH pixels high at
// the top (the selected tab) or none (barH 0). A tab with a bar has no edge
// at the bottom: it is open towards the content, which it belongs to.
func tabBox(size int, fill, border, bar color.RGBA, barH int) image.Image {
	img := box(size, fill, border).(*image.RGBA)
	if barH > 0 {
		for x := 1; x < size-1; x++ {
			img.Set(x, size-1, fill)
		}
	}
	for y := 0; y < barH; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bar)
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
		img.Set(x, y, accent)
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
	write("panel", box(16, panelFill, panelBorder), 2)
	write("title", accentBox(16, titleFill, titleFill, accent), 2)
	write("button", box(16, buttonFill, buttonEdge), 2)
	write("button_hover", accentBox(16, hoverFill, accent, accent), 2)
	write("button_pressed", box(16, pressedFill, accent), 2)
	// tabs: dark and low, lighter under the pointer, the selected one raised
	// with an accent bar on top
	write("tab", box(16, tabFill, buttonEdge), 3)
	write("tab_hover", tabBox(16, hoverFill, accent, accent, 0), 3)
	write("tab_on", tabBox(16, tabOnFill, accent, accent, 3), 3)
	write("slot", box(18, slotFill, panelBorder), 1)
	write("dim", box(4, dim, dim), 0)
	write("arrow_right", arrow(0), 0)
	write("arrow_down", arrow(1), 0)
	write("arrow_left", arrow(2), 0)
	write("arrow_up", arrow(3), 0)
}
