// bucketex writes the item textures of the wooden buckets
// (internal/games/dragonfactory/woodenbucket.go): an empty one and one with
// water, 16×16 pictures drawn from a bitmap. Temporary pictures.
//
//	go run ./tools/bucketex assets/resourcepack/dragonfactory
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// 'W' plank, 'w' plank light, 'd' plank dark, 'b' band, '~' water, '.' empty.
var bucket = []string{
	"................",
	"................",
	"...bbbbbbbbbb...",
	"..bw~~~~~~~~wb..",
	"..bwwwwwwwwwwb..",
	"..wWWWWWWWWWWd..",
	"..wWWWWWWWWWWd..",
	"..wWWWWWWWWWWd..",
	"..bbbbbbbbbbbb..",
	"..wWWWWWWWWWWd..",
	"..wWWWWWWWWWWd..",
	"...wWWWWWWWWd...",
	"...wWWWWWWWWd...",
	"...bbbbbbbbbb...",
	"....dddddddd....",
	"................",
}

var colours = map[byte]color.RGBA{
	'W': {160, 116, 62, 255},
	'w': {190, 146, 86, 255},
	'd': {112, 78, 40, 255},
	'b': {96, 96, 104, 255},
}

func write(path string, water bool) error {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y, row := range bucket {
		for x := 0; x < 16; x++ {
			c := row[x]
			switch {
			case c == '~' && water:
				img.SetRGBA(x, y, color.RGBA{58, 110, 220, 255})
			case c == '~':
				img.SetRGBA(x, y, colours['d'])
			case c != '.':
				img.SetRGBA(x, y, colours[c])
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: bucketex <resourcepack dir>")
		os.Exit(2)
	}
	dir := filepath.Join(os.Args[1], "textures", "items")
	for name, water := range map[string]bool{"wooden_bucket": false, "wooden_water_bucket": true} {
		if err := write(filepath.Join(dir, name+".png"), water); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
