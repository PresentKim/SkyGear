package farm

import "image"

// The pictures of these items come from the resource pack. Dragonfly calls
// Texture only when it builds its own pack, which is off (cmd/skygear/main.go
// and hooktest.Join), so the method only satisfies world.CustomItem.

func (Barrel) Texture() image.Image         { return nil }
func (CookedSilkworm) Texture() image.Image { return nil }
func (PlantSeed) Texture() image.Image      { return nil }
func (Silkworm) Texture() image.Image       { return nil }
func (SilkwormLeaves) Texture() image.Image { return nil }
