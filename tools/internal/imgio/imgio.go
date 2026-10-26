// Package imgio reads and writes the PNG files of the texture tools. The tools
// run by hand, so an error panics.
package imgio

import (
	"fmt"
	"image"
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

// Save writes im to path as a PNG.
func Save(path string, im image.Image) {
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
