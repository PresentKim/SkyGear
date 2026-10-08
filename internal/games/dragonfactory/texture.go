package dragonfactory

import (
	"embed"
	"image"
	"image/png"
	"sync"
)

// Textures are PNG files in textures/ (items/ for item icons, blocks/ for
// block model materials), embedded in the binary and put into the
// auto-generated resource pack by Dragonfly. Replace a file to change a
// texture; no code draws textures (docs/tech/addon.md).
//
//go:embed textures
var textureFiles embed.FS

var textureCache sync.Map

func loadTexture(path string) image.Image {
	if img, ok := textureCache.Load(path); ok {
		return img.(image.Image)
	}
	f, err := textureFiles.Open(path)
	if err != nil {
		panic("dragonfactory: missing texture " + path)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic("dragonfactory: bad texture " + path + ": " + err.Error())
	}
	textureCache.Store(path, img)
	return img
}

// itemTexture returns textures/items/<name>.png.
func itemTexture(name string) image.Image { return loadTexture("textures/items/" + name + ".png") }

// blockTexture returns textures/blocks/<name>.png.
func blockTexture(name string) image.Image { return loadTexture("textures/blocks/" + name + ".png") }

// Glyphs drawn by the resource pack font
// (assets/resourcepack/dragonfactory/font/glyph_E1.png): a padlock in the
// first cell and the void upgrade's mark in the second.
const (
	lockGlyph = "\ue100"
	voidGlyph = "\ue101"
)
