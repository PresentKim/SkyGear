// windowbuttons writes the item textures of the two buttons of machine
// windows (machine.go, takeAllSlot and putAllSlot): an arrow out of the
// machine (take everything out) and an arrow into it (put everything in).
// They are temporary pictures; the PNGs can be replaced by hand.
//
//	go run ./tools/windowbuttons assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

var (
	edge  = color.RGBA{30, 30, 30, 255}
	box   = color.RGBA{150, 150, 160, 255}
	green = color.RGBA{90, 220, 90, 255}
	amber = color.RGBA{240, 170, 50, 255}
)

// arrow draws a vertical arrow in column x from y0 to y1 (head at y1), 5
// pixels wide at the base of the head.
func arrow(img *image.RGBA, x, y0, y1 int, c color.RGBA) {
	step := 1
	if y1 < y0 {
		step = -1
	}
	for y := y0; y != y1; y += step {
		img.SetRGBA(x, y, c)
	}
	for i := 0; i <= 2; i++ {
		y := y1 - step*i
		for dx := -i; dx <= i; dx++ {
			img.SetRGBA(x+dx, y, c)
		}
	}
}

// frame draws the machine: a small box at the bottom.
func frame(img *image.RGBA) {
	for x := 2; x <= 13; x++ {
		for y := 11; y <= 14; y++ {
			c := box
			if x == 2 || x == 13 || y == 11 || y == 14 {
				c = edge
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func write(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: windowbuttons <pack dir>")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures", "items")
	take, put := image.NewRGBA(image.Rect(0, 0, 16, 16)), image.NewRGBA(image.Rect(0, 0, 16, 16))
	frame(take)
	frame(put)
	arrow(take, 8, 10, 1, green) // out of the machine, up to the player
	arrow(put, 8, 1, 9, amber)   // into the machine, down
	for name, img := range map[string]*image.RGBA{"machine_take_all": take, "machine_put_all": put} {
		if err := write(filepath.Join(dir, name+".png"), img); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
