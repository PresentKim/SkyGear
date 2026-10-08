package dragonfactory

import (
	"image"
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

// TestTexturesExist loads every texture the custom items and blocks use, so
// a missing or renamed PNG fails here instead of when the pack is built.
func TestTexturesExist(t *testing.T) {
	check := func(name string, f func() image.Image) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s: %v", name, r)
			}
		}()
		if img := f(); img.Bounds().Dx() == 0 {
			t.Errorf("%s: empty texture", name)
		}
	}
	items := world.CustomItems()
	if len(items) < 10 {
		t.Fatalf("only %d custom items registered", len(items))
	}
	for _, it := range items {
		name, _ := it.EncodeItem()
		check(name, it.Texture)
		if b, ok := it.(world.CustomBlockBuildable); ok {
			for tex := range b.Textures() {
				check(name+" "+tex, func() image.Image { return b.Textures()[tex] })
			}
		}
	}
}
