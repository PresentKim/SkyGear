// Command compacttex writes the block textures of the compressed blocks
// (content/skygear/sgitem/compact.go): the base block's colour in a grain
// that gets darker and denser each level, a frame that gets thicker, and as
// many light studs in the middle as the level. They are placeholders until
// the blocks are drawn by hand; a file that exists is left alone.
//
//	go run ./tools/compacttex assets/resourcepack/skygear
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/tools/internal/imgio"
)

// baseColours are the main colours of the base blocks, near the vanilla
// textures.
var baseColours = map[string]color.RGBA{
	"cobblestone": {122, 122, 122, 255},
	"gravel":      {131, 124, 122, 255},
	"sand":        {219, 207, 163, 255},
	"dirt":        {134, 96, 67, 255},
}

func scale(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 { return uint8(min(max(v, 0), 255)) }
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), 255}
}

// grain is a fixed pseudo-random shade at x, y for a seed.
func grain(seed, x, y int) float64 {
	h := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(seed*83492791)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return 0.78 + float64(h%9)*0.055
}

// studs are where the level studs go (top-left pixel of a 2×2 stud), by
// level.
var studs = [][][2]int{
	nil,
	{{7, 7}},
	{{5, 5}, {9, 9}},
	{{4, 4}, {7, 7}, {10, 10}},
	{{5, 5}, {9, 5}, {5, 9}, {9, 9}},
}

func draw(kind string, level int) *image.RGBA {
	col := baseColours[kind]
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	dark := 1 - 0.06*float64(level)
	frame := level // frame thickness: 1 to 4 would eat the face, so cap
	frame = min(frame, 2)
	for y := range 16 {
		for x := range 16 {
			f := grain(level*7+len(kind), x, y) * dark
			edge := min(x, y, 15-x, 15-y)
			switch {
			case edge < frame:
				f *= 0.55
			case edge == frame:
				f *= 1.15 // a light bevel inside the frame
			}
			im.Set(x, y, scale(col, f))
		}
	}
	// Inner rings for the levels above 2, so 3 and 4 differ from afar.
	for r := 3; r < 3+max(0, level-2); r++ {
		for i := r; i <= 15-r; i++ {
			for _, p := range [][2]int{{i, r}, {i, 15 - r}, {r, i}, {15 - r, i}} {
				im.Set(p[0], p[1], scale(col, 0.6*dark))
			}
		}
	}
	stud := scale(col, 1.45)
	if col.R > 200 { // light blocks: dark studs
		stud = scale(col, 0.5)
	}
	for _, s := range studs[level] {
		for dy := range 2 {
			for dx := range 2 {
				im.Set(s[0]+dx, s[1]+dy, stud)
			}
		}
	}
	return im
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: compacttex pack-dir")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures/blocks")
	for _, c := range sgitem.CompactBlocks() {
		path := filepath.Join(dir, fmt.Sprintf("compressed_%s_%d.png", c.Kind, c.Level))
		if _, err := os.Stat(path); err == nil {
			continue
		}
		imgio.Save(path, draw(c.Kind, c.Level))
		fmt.Println("wrote", path)
	}
}
